package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestProviderCompositionSelection(t *testing.T) {
	events := []string{}
	factories := []metadataProviderFactory{runtimeFactory("other", &events), runtimeFactory("second", &events)}
	for value, want := range map[string][]string{"other": {"other"}, " second , other ": {"second", "other"}, "none": {}, " none ": {}} {
		selected, err := selectMetadataProviders(value, factories)
		if err != nil {
			t.Fatal(err)
		}
		got := []string{}
		for _, f := range selected {
			got = append(got, f.ID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal(value, got, want)
		}
	}
	for _, value := range []string{"", "unknown", "other,other", "other,", "none,other", strings.Repeat("x", 4097)} {
		if _, err := selectMetadataProviders(value, factories); err == nil {
			t.Fatal("invalid selection accepted", value)
		}
	}
	if len(events) != 0 {
		t.Fatal("selection instantiated an adapter")
	}
}

func TestProviderCompositionRuntimeSelectionPreservesData(t *testing.T) {
	dir := t.TempDir()
	all := suppliedMetadataProviders()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c != nil {
			c.close()
		}
	})
	p := c.provider("tmdb").(*tmdbProvider)
	if err = c.setProviderCredential(context.Background(), "tmdb", "fixture-preserved-secret-123456"); err != nil {
		t.Fatal(err)
	}
	if err = p.setProviderEnabled(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	initial, err := p.installedPackage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	none, err := selectMetadataProviders("none", all)
	if err != nil {
		t.Fatal(err)
	}
	c, err = openCatalogWithComposition(dir, none, mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.providers) != 0 {
		t.Fatal("excluded adapters created")
	}
	storedSecret, err := tmdbConfiguration(c.db).Secret(context.Background(), "apiToken")
	if err != nil || storedSecret != "fixture-preserved-secret-123456" {
		t.Fatal("runtime filter erased credential")
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	p = c.provider("tmdb").(*tmdbProvider)
	after, err := p.installedPackage(context.Background())
	if err != nil || after.Checksum != initial.Checksum || providerEnabled(p) || c.credential(context.Background(), "tmdb") != "fixture-preserved-secret-123456" {
		t.Fatal("runtime filter changed package or activation", err)
	}
}

// Exercise the real authenticated router, not just catalogue construction.
func TestProviderCompositionBareServerAndLaterFamily(t *testing.T) {
	dir := t.TempDir()
	factories, err := selectMetadataProviders("none", suppliedMetadataProviders())
	if err != nil {
		t.Fatal(err)
	}
	c, err := openCatalogWithComposition(dir, factories, mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c != nil {
			c.close()
		}
	})
	mods, _ := enabledModules("video,music")
	h := newHandler(fstest.MapFS{"index.html": {Data: []byte("shell")}}, mods, c)
	admin := &accountClient{t: t, h: h}
	if w := admin.request("GET", "/api/v1/metadata/providers", nil); w.Code != 401 {
		t.Fatal("bare server bypassed authentication", w.Code)
	}
	admin.enter("setup", map[string]string{"name": "Owner", "password": accountTestPassword, "code": c.auth.code})
	for _, path := range []string{"/healthz", "/api/v1/status", "/api/v1/library", "/api/v1/metadata/providers", "/api/v1/media-bundles", "/api/v1/extension-repositories", "/api/v1/acquisition/settings", "/api/v1/acquisition/requests"} {
		if w := admin.request("GET", path, nil); w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w := admin.request("GET", "/api/v1/metadata/providers", nil); strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal("providers silently created", w.Body.String())
	}
	if w := admin.request("GET", "/api/v1/provider-packages/tmdb", nil); w.Code != 404 {
		t.Fatal("absent provider route", w.Code)
	}
	assertMediaComposition(t, c, mediaBootstrapSelection{})
	assertNoInitialTMDB(t, c.db)
	// Use the registry revision returned by the read-only planner, since it is
	// deliberately independent of an assumed initial revision number.
	plan := bundlePreview(t, c, "music", "prepare")
	preview := admin.request("POST", "/api/v1/media-bundles/music/preview", map[string]any{"operation": "prepare", "revision": plan.Revision})
	if preview.Code != 200 {
		t.Fatal(preview.Code, preview.Body.String())
	}
	if err = json.Unmarshal(preview.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	apply := admin.request("POST", "/api/v1/media-bundles/music/apply", map[string]any{"operation": "prepare", "revision": plan.Revision, "checksum": plan.Checksum})
	if apply.Code != 200 {
		t.Fatal(apply.Code, apply.Body.String())
	}
	assertMediaComposition(t, c, mediaBootstrapSelection{Families: []string{"music"}})
	mediaDir := t.TempDir()
	if err = os.WriteFile(filepath.Join(mediaDir, "sample.mp3"), testMP3(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = c.scan(testContext(context.Background()), library{Path: mediaDir, Kind: "music"}, true); err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	if len(items) != 1 || items[0].Metadata == nil || items[0].Metadata.Title != "Titre de test" {
		t.Fatal("local music depends on an online provider", items)
	}
	var grants int
	if err = c.db.QueryRow("SELECT count(*) FROM extension_grants WHERE extension_id='org.mosaune.screens.music'").Scan(&grants); err != nil || grants != 1 {
		t.Fatal("owner missing screen permission", grants, err)
	}
	c.close()
	c, err = openCatalogWithComposition(dir, factories, mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	assertMediaComposition(t, c, mediaBootstrapSelection{Families: []string{"music"}})
	assertNoInitialTMDB(t, c.db)
	h = newHandler(fstest.MapFS{"index.html": {Data: []byte("shell")}}, mods, c)
	admin = &accountClient{t: t, h: h}
	admin.enter("login", map[string]string{"name": "Owner", "password": accountTestPassword})
	if w := admin.request("GET", "/api/v1/metadata/providers", nil); strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal("reopen added providers")
	}
}
