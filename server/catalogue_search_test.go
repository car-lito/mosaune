package main

import (
	"context"
	"encoding/json"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
	"strings"
	"testing"
)

func TestCatalogueSearchFromGuestAndReplacement(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	items := sampleCatalogue()
	if warnings := c.projectCatalogue(ctx, items); len(warnings) > 0 {
		t.Fatal(warnings)
	}
	for _, m := range items {
		if m.Catalogue.Search == nil || len(m.Catalogue.Search.Entries) != 2 {
			t.Fatal("search index absent", m.Catalogue)
		}
	}
	if items[0].Catalogue.Search.Entries[0].Presentation.Label != "Album" || items[2].Catalogue.Search.Entries[1].Presentation.Label != "Épisode" {
		t.Fatal("wrong index")
	}
	b, _ := json.Marshal(items[0].Catalogue)
	if strings.Contains(string(b), "secret/folder") {
		t.Fatal("filesystem path exposed")
	}
	raw, p := catalogueExample(t, "2.0.0")
	installRecognitionExample(t, c, raw, p)
	recognitionChange(t, c, p.Manifest.Module.ID, "enable")
	recognitionChange(t, c, p.Manifest.Module.ID, "select")
	if warnings := c.projectCatalogue(ctx, items); len(warnings) > 0 {
		t.Fatal(warnings)
	}
	candidate := items[0].Catalogue.Search.Entries[0]
	if candidate.Presentation.Label != "Piste démo" || candidate.Presentation.Preview.Shape != "circle" || candidate.Terms[1] != "collection démo" {
		t.Fatal(candidate)
	}
	recognitionChange(t, c, p.Manifest.Module.ID, "disable")
	c.projectCatalogue(ctx, items)
	if items[0].Catalogue != nil {
		t.Fatal("disabled module search remains")
	}
}
func TestCatalogueSearchMigrationPreservesIntent(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	store := registry.SQLStore{DB: c.db}
	for _, kind := range catalogueKinds {
		d, _, e := modules.Lookup("catalogue-" + kind)
		if e != nil {
			t.Fatal(e)
		}
		snapshot, e := store.Load(ctx)
		if e != nil {
			t.Fatal(e)
		}
		old, ok := registryEntryFor(snapshot, d.ID)
		if !ok {
			t.Fatal("missing fixture")
		}
		old.Descriptor.Version = "0.1.0"
		old.Enabled = false
		old.Choices = nil
		if _, e = store.Commit(ctx, snapshot.Revision, registry.Change{Operation: "restore", ID: d.ID, Entry: &old}); e != nil {
			t.Fatal(e)
		}
	}
	recognitionChange(t, c, "catalogue-music", "disable")
	recognitionChange(t, c, "catalogue-series", "remove")
	if _, e := c.db.Exec("UPDATE recognition_selection SET id='org.example.missing.catalogue' WHERE kind='catalogue-films';PRAGMA user_version=32"); e != nil {
		t.Fatal(e)
	}
	c.close()
	var e error
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	snapshot, e := (registry.SQLStore{DB: c.db}).Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"films", "music"} {
		d, code, _ := modules.Lookup("catalogue-" + kind)
		v, ok := registryEntryFor(snapshot, d.ID)
		if !ok || v.Descriptor.Version != "0.2.0" || v.CodeSHA256 != recognitionHash(code) || v.Enabled != (kind == "films") {
			t.Fatal(v)
		}
	}
	id, _ := recognitionID("catalogue-series")
	if _, ok := registryEntryFor(snapshot, id); ok {
		t.Fatal("removed module resurrected")
	}
	var chosen string
	c.db.QueryRow("SELECT id FROM recognition_selection WHERE kind='catalogue-films'").Scan(&chosen)
	if chosen != "org.example.missing.catalogue" {
		t.Fatal("choice replaced", chosen)
	}
}
func TestCatalogueSearchMigrationAtomicFailure(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	store := registry.SQLStore{DB: c.db}
	snap, e := store.Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	id, _ := recognitionID("catalogue-films")
	old, _ := registryEntryFor(snap, id)
	old.Descriptor.Version = "0.1.0"
	old.Enabled = false
	old.Choices = nil
	if _, e = store.Commit(ctx, snap.Revision, registry.Change{Operation: "restore", ID: id, Entry: &old}); e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec("PRAGMA user_version=32;CREATE TEMP TRIGGER fail_search_migration BEFORE UPDATE ON recognition_packages BEGIN SELECT RAISE(ABORT,'fixture failure'); END"); e != nil {
		t.Fatal(e)
	}
	if migrateCatalogueSearch(c.db) == nil {
		t.Fatal("injected failure ignored")
	}
	after, e := store.Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	v, _ := registryEntryFor(after, id)
	if v.Descriptor.Version != "0.1.0" {
		t.Fatal("partial registry update")
	}
	var schema int
	c.db.QueryRow("PRAGMA user_version").Scan(&schema)
	if schema != 32 {
		t.Fatal(schema)
	}
}
