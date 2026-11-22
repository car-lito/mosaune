package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type runtimeProviderFixture struct {
	MetadataProvider
	id                       string
	owner                    *catalog
	events                   *[]string
	startError, errorOnClose error
	cancel                   context.CancelFunc
}

func (p *runtimeProviderFixture) Info() ProviderInfo {
	return ProviderInfo{ID: p.id, ContractVersion: 1}
}
func (p *runtimeProviderFixture) StartMetadataProvider(ctx context.Context) error {
	*p.events = append(*p.events, "start:"+p.id)
	if len(p.owner.providers) != 0 {
		return errors.New("partially published composition")
	}
	if p.cancel != nil {
		p.cancel()
	}
	return p.startError
}
func (p *runtimeProviderFixture) CloseMetadataProvider() error {
	*p.events = append(*p.events, "close:"+p.id)
	return p.errorOnClose
}
func runtimeFactory(id string, events *[]string) metadataProviderFactory {
	return metadataProviderFactory{ID: id, Create: func(c *catalog) (metadataProviderRuntime, error) {
		*events = append(*events, "create:"+id)
		return &runtimeProviderFixture{id: id, owner: c, events: events}, nil
	}}
}

func TestProviderRuntimeCompositionAndEmptyCatalogue(t *testing.T) {
	events := []string{}
	c, err := openCatalogWithProviders(t.TempDir(), []metadataProviderFactory{runtimeFactory("first", &events), runtimeFactory("second", &events)})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.providers) != 2 || c.provider("first") == nil || c.provider("tmdb") != nil {
		t.Fatal("composition replaced by built-ins")
	}
	if err = c.close(); err != nil {
		t.Fatal(err)
	}
	want := []string{"create:first", "start:first", "create:second", "start:second", "close:second", "close:first"}
	if !reflect.DeepEqual(events, want) {
		t.Fatal(events)
	}
	c, err = openCatalogWithProviders(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.providers) != 0 {
		t.Fatal("empty composition restored providers")
	}
	if err = c.close(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderRuntimeFailedStartupClosesEveryInstance(t *testing.T) {
	for _, stage := range []string{"construct", "start", "identity", "typed-nil", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			c := &catalog{}
			events := []string{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("fixture startup failure")
			closeFailure := errors.New("fixture close failure")
			second := runtimeFactory("second", &events)
			second.Create = func(owner *catalog) (metadataProviderRuntime, error) {
				events = append(events, "create:second")
				p := &runtimeProviderFixture{id: "second", owner: owner, events: &events, errorOnClose: closeFailure}
				switch stage {
				case "construct":
					return p, failure
				case "start":
					p.startError = failure
				case "identity":
					p.id = "mismatch"
				case "typed-nil":
					return (*runtimeProviderFixture)(nil), nil
				case "cancel":
					p.cancel = cancel
				}
				return p, nil
			}
			err := c.startMetadataProviders(ctx, []metadataProviderFactory{runtimeFactory("first", &events), second, runtimeFactory("never", &events)})
			if err == nil || len(c.providers) != 0 {
				t.Fatal("failed composition published", err)
			}
			if stage != "typed-nil" && !errors.Is(err, closeFailure) {
				t.Fatal("cleanup error lost", err)
			}
			if (stage == "construct" || stage == "start") && !errors.Is(err, failure) {
				t.Fatal("startup cause lost", err)
			}
			if stage == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost", err)
			}
			if events[len(events)-1] != "close:first" {
				t.Fatal("earlier adapter not closed", events)
			}
			if stage != "typed-nil" && events[len(events)-2] != "close:second" && events[len(events)-2] != "close:mismatch" {
				t.Fatal("failed adapter not closed", events)
			}
			for _, event := range events {
				if event == "create:never" {
					t.Fatal("started after failure")
				}
			}
		})
	}
}

func TestProviderRuntimePreflightAndRepeatedStart(t *testing.T) {
	c := &catalog{}
	events := []string{}
	for _, factories := range [][]metadataProviderFactory{
		{runtimeFactory("first", &events), runtimeFactory("first", &events)},
		{runtimeFactory("../other", &events)},
		{{ID: "first"}},
	} {
		if err := c.startMetadataProviders(context.Background(), factories); err == nil {
			t.Fatal("invalid composition accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.startMetadataProviders(ctx, []metadataProviderFactory{runtimeFactory("first", &events)}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatal("preflight allocated resources", events)
	}
	if err := c.startMetadataProviders(context.Background(), []metadataProviderFactory{runtimeFactory("first", &events)}); err != nil {
		t.Fatal(err)
	}
	defer closeMetadataProviders(c.providers)
	if err := c.startMetadataProviders(context.Background(), []metadataProviderFactory{runtimeFactory("second", &events)}); err == nil {
		t.Fatal("live composition overwritten")
	}
}

func TestProviderRuntimeTMDBShutdownKeepsPreferences(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	lease, done, err := beginProvider(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if err = c.close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(context.Cause(lease), errProviderStopped) {
		t.Fatal("shutdown did not cancel call")
	}
	if _, _, err = beginProvider(ctx, p); !errors.Is(err, errProviderStopped) {
		t.Fatal("new work accepted after shutdown", err)
	}
	if _, err = providerCommit(lease); !errors.Is(err, errProviderStopped) {
		t.Fatal("late result accepted", err)
	}
	if providerAvailable(p) {
		t.Fatal("closed instance available")
	}
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	if !providerEnabled(p) || !providerAvailable(p) {
		t.Fatal("shutdown changed persistent preference")
	}
	if err = p.setProviderEnabled(ctx, false); err != nil {
		t.Fatal(err)
	}
	if err = c.close(); err != nil {
		t.Fatal(err)
	}
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	if providerEnabled(c.provider("tmdb")) {
		t.Fatal("startup lost pause")
	}
}

func TestProviderRuntimeInvalidStoredStateFailsWithoutReset(t *testing.T) {
	c, dir := testCatalog(t)
	// Simulate damaged on-disk data, bypassing the fixture's normal SQL guard
	// only for this write. Production constraints remain unchanged.
	if _, err := c.db.Exec(`PRAGMA ignore_check_constraints=ON; UPDATE provider_runtime SET enabled='corrupt' WHERE id='tmdb'; PRAGMA ignore_check_constraints=OFF;`); err != nil {
		t.Fatal(err)
	}
	c.close()
	if broken, err := openCatalog(dir); err == nil {
		broken.close()
		t.Fatal("invalid state silently accepted")
	}
	// The error path closes both DB handles: reopen the same catalogue using an
	// explicit empty composition and prove the invalid value was not overwritten.
	c, err := openCatalogWithProviders(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	var raw string
	if err = c.db.QueryRow(`SELECT enabled FROM provider_runtime WHERE id='tmdb'`).Scan(&raw); err != nil || raw != "corrupt" {
		t.Fatal("state reset", raw, err)
	}
}
