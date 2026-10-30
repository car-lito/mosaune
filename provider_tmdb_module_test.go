package main

import (
	"context"
	"errors"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"strings"
	"testing"
	"time"
)

func useTMDBModuleFixture(t *testing.T, p *tmdbProvider, installed bool) {
	t.Helper()
	ctx := context.Background()
	if installed {
		pack, e := packageformat.Parse(providerCatalogueExample, trustedProviderKey(), version)
		if e != nil {
			t.Fatal(e)
		}
		if e = p.installPackage(ctx, pack, "", false); e != nil {
			t.Fatal(e)
		}
	}
	if e := p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
}

func TestTMDBModulePauseAndRemovalSurviveRestart(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	useTMDBModuleFixture(t, p, true)
	pack, e := p.installedPackage(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.removePackage(ctx, pack.Checksum); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected network") })
	if p.extensionStatus()["enabled"] || providerAvailable(p) {
		t.Fatal("removed module resurrected")
	}
	checks := []func() error{
		func() error { _, e := p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Film"}); return e },
		func() error { _, e := p.Details(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}); return e },
		func() error {
			_, e := p.Discover(ctx, "fixture", DiscoveryQuery{Kind: "films", Query: "Film", Page: 1})
			return e
		},
		func() error { _, e := p.Shelf(ctx, "fixture", "upcoming", "films"); return e },
		func() error { _, e := p.Profile(ctx, "fixture", "42", 1); return e },
		func() error {
			_, e := p.Posters(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}, "fr")
			return e
		},
		func() error { _, e := p.Artwork(ctx, "fixture", "/poster.jpg"); return e },
		func() error { _, e := p.Artwork(ctx, "fixture", "yt:abcdefghijk"); return e },
		func() error { _, e := p.AcquisitionID(ctx, "fixture", "series", "42"); return e },
	}
	for _, check := range checks {
		if e := check(); !errors.Is(e, errProviderModuleUnavailable) {
			t.Fatal(e)
		}
	}
	// Explicitly re-enable the supplied WASI module; safe mode still blocks all remote paths.
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	c.extensionSafeMode = true
	for _, check := range checks {
		if e := check(); !errors.Is(e, errProviderModuleUnavailable) {
			t.Fatal("safe-mode", e)
		}
	}
	if calls != 0 {
		t.Fatal("fallback network calls", calls)
	}
	c.extensionSafeMode = false
	if e = p.setExtension(ctx, false); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	if c.provider("tmdb").(*tmdbProvider).extensionStatus()["enabled"] {
		t.Fatal("pause lost on restart")
	}
}

func TestTMDBModuleMigrationPreservesProviderDenial(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	pack, err := packageformat.Parse(providerUpdateExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, pack, "", false); err != nil {
		t.Fatal(err)
	}
	if e := p.setProviderEnabled(ctx, false); e != nil {
		t.Fatal(e)
	}
	changeTMDBAccess(t, p, false)
	if _, e := c.db.Exec("UPDATE provider_runtime SET enabled=0 WHERE id='tmdb'; PRAGMA user_version=27"); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	kept, err := p.installedPackage(ctx)
	if err != nil || kept.Checksum != pack.Checksum {
		t.Fatal("migration replaced installed package", err)
	}
	status := p.extensionStatus()
	if !status["enabled"] || status["providerEnabled"] || status["effective"] || status["serviceAccess"] || status["loaded"] {
		t.Fatal(status)
	}
	if _, e = p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Film"}); !errors.Is(e, errProviderDisabled) {
		t.Fatal(e)
	}
}

func TestTMDBModulePauseRejectsLateResponse(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	entered := make(chan context.Context, 1)
	release := make(chan struct{})
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		entered <- r.Context()
		<-release
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"results":[{"id":42,"title":"Late"}]}`))}, nil
	})
	finished := make(chan error, 1)
	go func() { _, e := p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Late"}); finished <- e }()
	var request context.Context
	select {
	case request = <-entered:
	case <-time.After(15 * time.Second):
		t.Fatal("call did not start")
	}
	paused := make(chan error, 1)
	go func() { paused <- p.setExtension(ctx, false) }()
	select {
	case <-request.Done():
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("request not cancelled")
	}
	close(release)
	select {
	case e := <-paused:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pause stuck")
	}
	select {
	case e := <-finished:
		if !errors.Is(e, errProviderModuleUnavailable) {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("caller stuck")
	}
	var count int
	if e := c.db.QueryRow("SELECT COUNT(*) FROM provider_cache WHERE provider='tmdb'").Scan(&count); e != nil || count != 0 {
		t.Fatal("late cache", count, e)
	}
}
