package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	discovery "mosaune/extensionkit/discovery/v1"
	f "mosaune/extensionkit/packageformat"
	"path/filepath"
	"testing"
)

func TestInstalledDiscoveryUpdateRollbackRemoval(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	// Install the previous immutable catalogue package to exercise the upgrade path.
	removeTMDBPackageFixture(t, p)
	old, err := f.Parse(providerCatalogueExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, old, "", false); err != nil {
		t.Fatal(err)
	}
	assertSurface := func(p *tmdbProvider, search, shelves int) {
		t.Helper()
		s, e := providerDiscoverySurface(ctx, p)
		if e != nil || s == nil || len(s.Search) != search || len(s.Shelves) != shelves {
			t.Fatalf("surface: %#v %v", s, e)
		}
	}
	assertSurface(p, 2, 3)
	// Use a trusted fixture signer so we can restrict the signed surface without
	// changing the Wasm engine. Empty means none, never legacy defaults.
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	resign := func(pack f.Package) f.Package {
		t.Helper()
		raw, e := f.Create(pack.Manifest, pack.Code, key)
		if e != nil {
			t.Fatal(e)
		}
		v, e := f.Parse(raw, pub, version)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	old = resign(old)
	// Fixture replacement establishes trust before exercising only public routes.
	if _, err = c.db.Exec("UPDATE provider_packages SET archive=?,checksum=? WHERE id='tmdb'", old.Archive, old.Checksum); err != nil {
		t.Fatal(err)
	}
	p.extension.key = pub
	next, err := f.Parse(providerDiscoveryExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	next.Manifest.Discovery = &discovery.Surface{Version: 1}
	next = resign(next)
	if w := providerUpload(admin, "install", next.Archive, next.Checksum, old.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertSurface(p, 0, 0)
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	if w := admin.request("POST", "/api/v1/discovery/browse", map[string]any{"provider": "tmdb", "kind": "films", "query": "Film", "page": 1}); w.Code != 400 {
		t.Fatal("search still exposed", w.Code, w.Body.String())
	}
	if w := admin.request("POST", "/api/v1/shelves", map[string]string{"provider": "tmdb", "kind": "films", "shelf": "upcoming"}); w.Code != 400 {
		t.Fatal("shelf still exposed", w.Code, w.Body.String())
	}
	if _, err = p.Shelf(ctx, "fixture", "upcoming", "films"); err == nil {
		t.Fatal("direct adapter bypassed declaration")
	}
	var file string
	if err = c.db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&file); err != nil {
		t.Fatal(err)
	}
	reopened, err := openCatalog(filepath.Dir(file))
	if err != nil {
		t.Fatal(err)
	}
	rp := reopened.provider("tmdb").(*tmdbProvider)
	rp.extension.key = pub
	assertSurface(rp, 0, 0)
	reopened.close()
	if w := admin.request("POST", "/api/v1/provider-packages/tmdb/rollback", map[string]string{"checksum": next.Checksum, "previousChecksum": old.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertSurface(p, 2, 3)
	s, _ := providerDiscoverySurface(ctx, p)
	s.Search[0].Label = "mutated"
	s.Shelves[0].ID = "mutated"
	s, _ = providerDiscoverySurface(ctx, p)
	if s.Search[0].Label == "mutated" || s.Shelves[0].ID == "mutated" {
		t.Fatal("cache escaped")
	}
	// Another valid update supplies its own labels, not the bundled defaults.
	next, err = f.Parse(providerDiscoveryExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	next.Manifest.Discovery.Search[0].Label = "Cinéma du paquet"
	next.Manifest.Discovery.Shelves = next.Manifest.Discovery.Shelves[:1]
	next = resign(next)
	if w := providerUpload(admin, "install", next.Archive, next.Checksum, old.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertSurface(p, 2, 1)
	s, _ = providerDiscoverySurface(ctx, p)
	if s.Search[0].Label != "Cinéma du paquet" {
		t.Fatal("static label instead of installed declaration")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := providerDiscoverySurface(cancelled, p); e == nil {
		t.Fatal("cancelled read accepted")
	}
	if w := admin.request("DELETE", "/api/v1/provider-packages/tmdb", map[string]string{"checksum": next.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertSurface(p, 0, 0)
}
