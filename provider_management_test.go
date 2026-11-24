package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type managedRuntimeFixture struct {
	*authorizedProviderFixture
	running bool
	calls   int
	reject  bool
}

func (p *managedRuntimeFixture) providerRuntime() map[string]bool {
	return map[string]bool{"enabled": p.running, "installed": true}
}
func (p *managedRuntimeFixture) setProviderRuntime(ctx context.Context, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.reject {
		return errors.New("Rejected fixture transition")
	}
	p.calls++
	p.running = enabled
	return nil
}

func TestProviderManagementContractsAndPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	fixture := &managedRuntimeFixture{authorizedProviderFixture: alternateServiceFixture(t, c)}
	c.providers[len(c.providers)-1] = fixture
	read := func() []providerManagement {
		t.Helper()
		w := admin.request("GET", "/api/v1/metadata/provider-management", nil)
		var out []providerManagement
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		return out
	}
	items := read()
	if len(items) != 3 {
		t.Fatal(items)
	}
	var tmdb, other, music providerManagement
	for _, item := range items {
		switch item.ID {
		case "tmdb":
			tmdb = item
		case "musicbrainz":
			music = item
		case "fixture-provider":
			other = item
		}
	}
	if !tmdb.Packages || !tmdb.Activation || !tmdb.RuntimeControl || tmdb.ConfigurationOwner != tmdbConfigurationOwner {
		t.Fatal(tmdb)
	}
	if music.Packages || music.Activation || music.RuntimeControl || music.ConfigurationOwner != "" {
		t.Fatal("invented controls", music)
	}
	if other.Packages || !other.Activation || !other.RuntimeControl || other.ConfigurationOwner != "fixture.settings" {
		t.Fatal("contract ignored", other)
	}
	const route = "/api/v1/metadata/providers/fixture-provider/runtime"
	for _, body := range []map[string]any{{}, {"enabled": nil}, {"enabled": "true"}} {
		if w := admin.request("PUT", route, body); w.Code != 400 {
			t.Fatal("invalid transition", w.Code)
		}
	}
	if fixture.calls != 0 {
		t.Fatal("invalid request reached adapter")
	}
	if w := admin.request("PUT", route, map[string]bool{"enabled": true}); w.Code != 200 || !fixture.running {
		t.Fatal(w.Code)
	}
	fixture.reject = true
	if w := admin.request("PUT", route, map[string]bool{"enabled": false}); w.Code != 400 || !fixture.running {
		t.Fatal("failure changed state", w.Code)
	}
	for _, id := range []string{"absent", "musicbrainz"} {
		if w := admin.request("PUT", "/api/v1/metadata/providers/"+id+"/runtime", map[string]bool{"enabled": true}); w.Code != 404 {
			t.Fatal(id, w.Code)
		}
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	if w := reader.request("GET", "/api/v1/metadata/provider-management", nil); w.Code != 403 {
		t.Fatal("management exposed", w.Code)
	}
	if w := reader.request("PUT", route, map[string]bool{"enabled": false}); w.Code != 403 || fixture.calls != 1 {
		t.Fatal("unauthorized dispatch", w.Code)
	}
	// Generic and compatibility controls operate on the same real TMDB module.
	if w := admin.request("PUT", "/api/v1/metadata/providers/tmdb/runtime", map[string]bool{"enabled": false}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if c.provider("tmdb").(*tmdbProvider).extensionStatus()["enabled"] {
		t.Fatal("generic control missed TMDB")
	}
	if w := admin.request("PUT", "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": true}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !c.provider("tmdb").(*tmdbProvider).extensionStatus()["enabled"] {
		t.Fatal("compatibility diverged")
	}
}
