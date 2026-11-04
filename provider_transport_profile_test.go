package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"testing"
)

func TestTransportPackageSettingsAcrossVersions(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	next, e := p.installedPackage(ctx)
	if e != nil || next.Manifest.Transport == nil {
		t.Fatal("missing signed profile", e)
	}
	old, e := packageformat.Parse(providerDiscoveryExample, trustedProviderKey(), version)
	if e != nil {
		t.Fatal(e)
	}
	// Reopen a v36 installation with the factory v0.6 package switched off.
	removeTMDBPackageFixture(t, p)
	if e = p.installPackage(ctx, old, "", false); e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec("PRAGMA user_version=36"); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	migrated, e := p.installedPackage(ctx)
	if e != nil || migrated.Checksum != next.Checksum || p.extensionStatus()["enabled"] {
		t.Fatal("migration changed pause or failed", e)
	}
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	bind, ok := c.configurationFor(tmdbConfigurationOwner)
	if !ok {
		t.Fatal("settings missing")
	}
	patch := configuration.Patch{Values: map[string]json.RawMessage{"language": json.RawMessage(`"nl-NL"`), "region": json.RawMessage(`"NL"`)}}
	if e = bind.write(ctx, patch); e != nil {
		t.Fatal(e)
	}
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.themoviedb.org" || r.URL.Query().Get("region") != "NL" || r.URL.Query().Get("language") != "nl-NL" {
			t.Errorf("wrong request %v", r.URL.Redacted())
		}
		return visualResponse(200, `{"results":[],"total_pages":1,"page":1}`), nil
	})
	if _, e = p.Shelf(ctx, "fixture", "upcoming", "films"); e != nil || calls != 1 {
		t.Fatal("configured shelf", calls, e)
	}
	if e = p.installPackage(ctx, old, next.Checksum, true); e != nil {
		t.Fatal("rollback", e)
	}
	bind, _ = c.configurationFor(tmdbConfigurationOwner)
	state, e := bind.read(ctx)
	if e != nil || len(state.Schema.Fields) != 1 || state.Values["language"] != nil {
		t.Fatal("old settings leaked newer fields", e)
	}
	if v, e := p.configuredProfile(ctx); e != nil || v.Region != "BE" {
		t.Fatal("legacy configuration", e)
	}
	if e = p.installPackage(ctx, next, old.Checksum, false); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	v, e := c.provider("tmdb").(*tmdbProvider).configuredProfile(ctx)
	if e != nil || v.Region != "NL" || v.Language != "nl-NL" {
		t.Fatal("lost settings across rollback, update or restart", e)
	}
}

func TestTransportPackageCannotBroadenNetworkConsent(t *testing.T) {
	for _, which := range []string{"api-origin", "image-secret", "port", "interval", "format", "limit", "insecure", "settings", "legacy"} {
		p, e := packageformat.Parse(providerTransportExample, trustedProviderKey(), version)
		if e != nil {
			t.Fatal(e)
		}
		m := p.Manifest
		s := &m.Transport.Sources[0]
		switch which {
		case "api-origin":
			s.BaseURL = "https://unapproved.example"
		case "image-secret":
			m.Transport.Sources[1].Credential = "apiToken"
		case "port":
			s.BaseURL = "https://api.themoviedb.org:8443"
		case "interval":
			s.IntervalMS = 0
		case "format":
			s.Format = "image"
		case "limit":
			s.MaxBytes = 9 << 20
		case "insecure":
			s.BaseURL = "http://api.themoviedb.org"
		case "settings":
			m.Transport.RegionSetting = "apiToken"
		case "legacy":
			m.MinCore = "0.0.56"
		}
		if m.Validate(version) == nil {
			t.Error("unsafe profile accepted:", which)
		}
	}
}

func TestTransportUsesInstalledProfile(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	pack, e := p.installedPackage(ctx)
	if e != nil {
		t.Fatal(e)
	}
	removeTMDBPackageFixture(t, p)
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	src := &pack.Manifest.Transport.Sources[0]
	src.BaseURL = "https://api.themoviedb.org/test-prefix"
	src.MaxBytes = 8
	raw, e := packageformat.Create(pack.Manifest, pack.Code, key)
	if e != nil {
		t.Fatal(e)
	}
	pack, e = packageformat.Parse(raw, pub, version)
	if e != nil {
		t.Fatal(e)
	}
	p.extension.key = pub
	if e = p.installPackage(ctx, pack, "", false); e != nil {
		t.Fatal(e)
	}
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	called := false
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		if r.URL.Path != "/test-prefix/search/movie" {
			t.Error("did not use installed source")
		}
		return visualResponse(200, `{"results":[]}`), nil
	})
	var out json.RawMessage
	if e = p.get(ctx, "fixture", "/search/movie", nil, &out); e == nil || !called {
		t.Fatal("package byte limit ignored", e)
	}
}
