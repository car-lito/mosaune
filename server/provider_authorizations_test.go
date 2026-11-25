package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/authorization"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
	"strings"
	"sync"
	"testing"
)

type authorizedProviderFixture struct {
	*configurationProviderFixture
	access providerServiceAccess
}

func (p *authorizedProviderFixture) ServiceAccess() providerServiceAccess { return p.access }
func (p *authorizedProviderFixture) Configuration(ctx context.Context) (configurationBinding, error) {
	binding, err := p.configurationProviderFixture.Configuration(ctx)
	binding.descriptor.Services = []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"read", "write"}}}
	return binding, err
}
func (p *authorizedProviderFixture) lifecycle() *providerLifecycle { return p.access.lifecycle }
func (p *authorizedProviderFixture) setProviderEnabled(ctx context.Context, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s := p.lifecycle()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled = !enabled
	if !enabled {
		for _, cancel := range s.calls {
			cancel(errProviderDisabled)
		}
	}
	return nil
}
func alternateServiceFixture(t *testing.T, c *catalog) *authorizedProviderFixture {
	t.Helper()
	ctx := context.Background()
	d := protocol.Descriptor{ID: "fixture.settings", Version: "1.0.0", Capabilities: []string{"fixture.reader.v1"}, Services: []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"read", "write"}}}}
	if err := (authorization.SQLStore{DB: c.db}).Register(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := (configuration.SQLStore{DB: c.db}).Register(ctx, d.ID, configSchema()); err != nil {
		t.Fatal(err)
	}
	bound, _ := (configuration.SQLStore{DB: c.db}).For(d.ID)
	p := &authorizedProviderFixture{configurationProviderFixture: &configurationProviderFixture{bound: bound}, access: providerServiceAccess{db: c.db, owner: d.ID, lifecycle: &providerLifecycle{}, services: []extensionServiceView{{Name: "fixture.remote.v1", Label: "Lire la source test", Description: "Description de l’adaptateur approuvé.", Operations: []string{"read"}}}}}
	if err := p.access.load(ctx); err != nil {
		t.Fatal(err)
	}
	c.providers = append(c.providers, p)
	return p
}

func TestProviderAuthorizationAlternateAdapterHTTPAndLifecycle(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := alternateServiceFixture(t, c)
	ctx := context.Background()
	const path = "/api/v1/extensions/fixture.settings/authorizations"
	var state authorization.State
	read := func() {
		t.Helper()
		w := admin.request("GET", path, nil)
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &state) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	read()
	if providerAvailable(p) {
		t.Fatal("registration granted permission")
	}
	if len(state.Requested) != 1 || len(state.Requested[0].Operations) != 1 || state.Requested[0].Operations[0] != "read" {
		t.Fatal("unsupported operation offered", state)
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision, Granted: []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"write"}}}}); w.Code != 400 {
		t.Fatal("unsupported operation granted", w.Code)
	}
	grant := []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"read"}}}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision, Granted: grant}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !providerAvailable(p) {
		t.Fatal("permission ineffective")
	}
	if w := admin.request("GET", "/api/v1/extensions/fixture.settings/configuration", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"authorizationsManaged":true`) || !strings.Contains(w.Body.String(), "Lire la source test") {
		t.Fatal(w.Code, w.Body.String())
	}
	read()
	lease, done, err := beginProvider(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if _, err = c.db.Exec(`CREATE TRIGGER fail_fixture_access BEFORE UPDATE ON extension_authorizations BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision}); w.Code != 500 {
		t.Fatal(w.Code)
	}
	if !providerAvailable(p) || context.Cause(lease) != nil {
		t.Fatal("failed persistence changed lifecycle")
	}
	if _, err = c.db.Exec(`DROP TRIGGER fail_fixture_access`); err != nil {
		t.Fatal(err)
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !errors.Is(context.Cause(lease), errProviderAccessDenied) || providerAvailable(p) {
		t.Fatal("revocation did not cancel provider")
	}
	if _, _, err = beginProvider(ctx, p); !errors.Is(err, errProviderAccessDenied) {
		t.Fatal("new call after revoke", err)
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision, Granted: grant}); w.Code != 409 {
		t.Fatal("stale grant", w.Code)
	}
	reloaded := p.access
	reloaded.lifecycle = &providerLifecycle{}
	if err = reloaded.load(ctx); err != nil || !reloaded.lifecycle.accessDenied {
		t.Fatal("reload lost denial", err)
	}
	read()
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision, Granted: grant}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, err = providerCommit(lease); !errors.Is(err, errProviderAccessDenied) {
		t.Fatal("regrant revived old commit", err)
	}
	if !providerAvailable(c.provider("tmdb")) {
		t.Fatal("other provider affected")
	}
	if w := admin.request("PUT", "/api/v1/metadata/providers/fixture-provider/state", map[string]bool{"enabled": false}); w.Code != 200 || providerEnabled(p) {
		t.Fatal("state not dispatched", w.Code)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	for _, method := range []string{"GET", "PUT"} {
		if w := reader.request(method, path, map[string]any{}); w.Code != 403 {
			t.Fatal("non-admin", w.Code)
		}
	}
	if w := reader.request("PUT", "/api/v1/metadata/providers/fixture-provider/state", map[string]bool{"enabled": true}); w.Code != 403 {
		t.Fatal("non-admin state", w.Code)
	}
}

func TestProviderAuthorizationDescriptionsAndUnmanagedOwners(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := alternateServiceFixture(t, c)
	d := protocol.Descriptor{ID: p.access.owner, Services: []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"read", "write"}}, {Name: "arbitrary.network.v1", Operations: []string{"get"}}}}
	views := c.serviceViews(d)
	if len(views) != 1 || views[0].Label != "Lire la source test" || len(views[0].Operations) != 1 || views[0].Operations[0] != "read" {
		t.Fatal("view not filtered", views)
	}
	d.ID = "fixture.unmanaged"
	d.Version = "1.0.0"
	d.Capabilities = []string{"fixture.reader.v1"}
	if err := (authorization.SQLStore{DB: c.db}).Register(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if len(c.serviceViews(d)) != 0 {
		t.Fatal("row exposed services")
	}
	for _, method := range []string{"GET", "PUT"} {
		if w := admin.request(method, "/api/v1/extensions/fixture.unmanaged/authorizations", map[string]any{}); w.Code != 404 {
			t.Fatal("row exposed grants", w.Code)
		}
	}
	p.access.lifecycle = nil
	if w := admin.request("GET", "/api/v1/extensions/fixture.settings/authorizations", nil); w.Code != 404 {
		t.Fatal("missing lifecycle accepted", w.Code)
	}
}

func TestProviderAuthorizationConcurrentEdits(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	p := alternateServiceFixture(t, c)
	ctx := context.Background()
	state, err := p.access.read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	patch := authorization.Patch{Revision: &state.Revision, Granted: []protocol.ServiceRequirement{{Name: "fixture.remote.v1", Operations: []string{"read"}}}}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- p.access.apply(ctx, patch) }()
	}
	wg.Wait()
	close(errs)
	passed, conflicted := 0, 0
	for err := range errs {
		if err == nil {
			passed++
		} else if errors.Is(err, authorization.ErrConflict) {
			conflicted++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || conflicted != 1 || !providerAvailable(p) {
		t.Fatal(passed, conflicted)
	}
}
