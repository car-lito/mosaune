package main

import (
	"context"
	"encoding/json"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func catalogueExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, err := os.ReadFile("extension-kit/examples/catalogue/extension.json")
	if err != nil {
		t.Fatal(err)
	}
	var m packageformat.ExecutableManifest
	if err = protocol.DecodeOne(raw, &m); err != nil {
		t.Fatal(err)
	}
	m.Module.Version = v
	code, err := os.ReadFile("extension-kit/examples/catalogue/module.wasm")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = packageformat.CreateExecutable(m, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	return raw, pack
}

func removeCatalogueForLegacyFixture(t *testing.T, c *catalog) {
	t.Helper()
	for _, kind := range catalogueKinds {
		recognitionChange(t, c, "catalogue-"+kind, "remove")
	}
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind LIKE 'catalogue-%'"); e != nil {
		t.Fatal(e)
	}
}
func sampleCatalogue() []media {
	return []media{{ID: "a", Kind: "music", Name: "One", Filename: "secret/folder/a.mp3", Metadata: &Metadata{Title: "One", AlbumArtist: "Artist", Artist: "Guest", Album: "Album", Disc: 1, Track: 10}}, {ID: "b", Kind: "music", Name: "Two", Filename: "b.flac", Metadata: &Metadata{Title: "Two", Artist: "Artist", Album: "Album", Disc: 1, Track: 2}}, {ID: "e10", Kind: "series", Name: "Ten", Metadata: &Metadata{Series: "Show", Season: 1, Episode: 10}}, {ID: "e2", Kind: "series", Name: "Two", Metadata: &Metadata{Series: "Show", Season: 1, Episode: 2}}}
}
func TestCatalogueModulesAndLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	items := sampleCatalogue()
	original := sampleCatalogue()
	check := func(c *catalog, wantOwner string, wantWarnings int) {
		t.Helper()
		warnings := c.projectCatalogue(ctx, items)
		if len(warnings) != wantWarnings {
			t.Fatal(warnings)
		}
		if wantOwner == "" {
			if items[0].Catalogue != nil || items[1].Catalogue != nil {
				t.Fatal("disabled policy returned groups")
			}
		} else if items[0].Catalogue == nil || items[0].Catalogue.Owner != wantOwner {
			t.Fatal("incorrect selection", items[0].Catalogue)
		}
		if items[2].Catalogue == nil || items[2].Catalogue.Key != items[3].Catalogue.Key {
			t.Fatal("unrelated series affected")
		}
	}
	check(c, "mosaune.music.catalogue", 0)
	if items[0].Catalogue.Key != items[1].Catalogue.Key || items[0].Catalogue.Order[1] != 10 {
		t.Fatal("album grouping")
	}
	if catalogueFacts(items[0]).Filename != "a.mp3" {
		t.Fatal("directory disclosed")
	}
	raw, pack := catalogueExample(t, "1.0.0")
	id := pack.Manifest.Module.ID
	installRecognitionExample(t, c, raw, pack)
	recognitionChange(t, c, id, "enable")
	check(c, "mosaune.music.catalogue", 0)
	old, done, _, _, err := c.beginRecognition(ctx, "catalogue-music")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	recognitionChange(t, c, id, "select")
	if old.Err() == nil {
		t.Fatal("selection did not cancel in-flight projection")
	}
	if unlock, e := c.recognitionCommit(old); e == nil {
		unlock()
		t.Fatal("late result accepted")
	}
	check(c, id, 0)
	if items[0].Catalogue.Key != "" || items[0].Catalogue.Subtitle != "Exemple contributeur" {
		t.Fatal("external program did not run")
	}
	newer, p2 := catalogueExample(t, "1.1.0")
	installRecognitionExample(t, c, newer, p2)
	check(c, id, 0)
	recognitionChange(t, c, id, "restore")
	check(c, id, 0)
	recognitionChange(t, c, id, "disable")
	check(c, "", 1)
	recognitionChange(t, c, id, "enable")
	recognitionChange(t, c, id, "remove")
	check(c, "", 1)
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	check(next, "", 1)
	recognitionChange(t, next, "catalogue-music", "select")
	check(next, "mosaune.music.catalogue", 0)
	if _, err = next.db.Exec("UPDATE recognition_packages SET code=x'00' WHERE kind='catalogue-music'"); err != nil {
		t.Fatal(err)
	}
	check(next, "", 1)
	for i := range items {
		items[i].Catalogue = nil
	}
	if !reflect.DeepEqual(items, original) {
		t.Fatal("projection changed source metadata")
	}
	next.extensionSafeMode = true
	if warnings := next.projectCatalogue(ctx, items); len(warnings) != 2 || items[2].Catalogue != nil {
		t.Fatal("safe mode executed projection")
	}
}
func TestCatalogueRealLibraryPermissionsAndRestart(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := context.Background()
	files := t.TempDir()
	if err := os.WriteFile(filepath.Join(files, "a.mp3"), testMP3(), 0600); err != nil {
		t.Fatal(err)
	}
	scan := admin.request("POST", "/api/v1/libraries", map[string]string{"name": "Music", "kind": "music", "path": files})
	if scan.Code != 201 {
		t.Fatal(scan.Body.String())
	}
	var lib library
	json.Unmarshal(scan.Body.Bytes(), &lib)
	get := func(client *accountClient) []media {
		t.Helper()
		r := client.request("GET", "/api/v1/library", nil)
		if r.Code != 200 {
			t.Fatal(r.Code, r.Body.String())
		}
		var data struct{ Items []media }
		if err := json.Unmarshal(r.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data.Items
	}
	list := get(admin)
	if len(list) != 1 || list[0].Catalogue == nil || list[0].Catalogue.Title != "Album" {
		t.Fatal("real library missing projection", list)
	}
	admin.create("Hidden", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Hidden", "password": accountTestPassword})
	if len(get(reader)) != 0 {
		t.Fatal("private library leaked")
	}
	recognitionChange(t, c, "catalogue-music", "disable")
	if got := get(admin); len(got) != 1 || got[0].Catalogue != nil || got[0].Metadata.Album != list[0].Metadata.Album {
		t.Fatal("disabled catalogue lost facts")
	}
	recognitionChange(t, c, "catalogue-music", "enable")
	if get(admin)[0].Catalogue == nil {
		t.Fatal("needs rescan")
	}
	// Scan has its independent recognizer and is not disabled with the catalogue view.
	recognitionChange(t, c, "catalogue-music", "remove")
	if _, err := c.scan(testContext(ctx), lib, false); err != nil {
		t.Fatal("projection removal stopped scan", err)
	}
}
func TestCatalogueMigration21AndPolicy(t *testing.T) {
	c, dir := testCatalog(t)
	recognitionChange(t, c, "music", "disable")
	removeCatalogueForLegacyFixture(t, c)
	if _, err := c.db.Exec("PRAGMA user_version=21"); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	list, _, err := next.recognitionPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range list {
		if strings.HasPrefix(v.Kind, "catalogue-") && (!v.Installed || !v.Enabled || !v.Selected) {
			t.Fatal(v)
		}
		if v.Kind == "music" && v.Enabled {
			t.Fatal("migration changed recognizer choice")
		}
	}
	recognitionChange(t, next, "catalogue-films", "remove")
	again, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer again.close()
	list, _, _ = again.recognitionPackages(context.Background())
	for _, v := range list {
		if v.Kind == "catalogue-films" && v.Installed {
			t.Fatal("removed module reseeded")
		}
	}
	_, pack := catalogueExample(t, "1.0.0")
	for _, mutate := range []func(*packageformat.Executable){func(p *packageformat.Executable) {
		p.Manifest.Module.Services = []protocol.ServiceRequirement{{Name: "media.file.read.v1", Operations: []string{"range"}}}
	}, func(p *packageformat.Executable) { p.Manifest.Contributions[0].Target = "unknown" }, func(p *packageformat.Executable) { p.Manifest.Module.ID = "mosaune.evil" }} {
		_, copy := catalogueExample(t, "1.0.0")
		mutate(&copy)
		if _, err := validateRecognitionImport(copy); err == nil {
			t.Fatal("invalid permissions accepted")
		}
	}
	raw := mustAudioArchive(t, pack)
	installRecognitionExample(t, next, raw, pack)
	_, changed := recognitionExample(t, "1.1.0")
	changed.Manifest.Module.ID = pack.Manifest.Module.ID
	if _, err := next.previewRecognition(context.Background(), changed); err == nil {
		t.Fatal("role changed")
	}
}
func TestCatalogueThousandFilmsBoundedProjection(t *testing.T) {
	c, _ := testCatalog(t)
	items := make([]media, 1000)
	for i := range items {
		items[i] = media{ID: strings.Repeat("x", 20), Kind: "films", Name: "Film", Metadata: &Metadata{Title: "Film", Year: 2008}}
	}
	start := time.Now()
	if warnings := c.projectCatalogue(context.Background(), items); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	for _, v := range items {
		if v.Catalogue == nil || v.Catalogue.Subtitle != "2008" {
			t.Fatal("incomplete projection")
		}
	}
	t.Logf("1000 films first: %s", time.Since(start))
	start = time.Now()
	if warnings := c.projectCatalogue(context.Background(), items); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	t.Logf("1000 films cached: %s", time.Since(start))
	items[0].Metadata.Title = "Changed title"
	if warnings := c.projectCatalogue(context.Background(), items); len(warnings) != 0 || items[0].Catalogue.Title != "Changed title" {
		t.Fatal("stale metadata cache", warnings)
	}
	// A cached programme must still honor disable and safe-mode decisions.
	recognitionChange(t, c, "catalogue-films", "disable")
	if warnings := c.projectCatalogue(context.Background(), items); len(warnings) != 1 || items[0].Catalogue != nil {
		t.Fatal("cache bypassed disabled module")
	}
	recognitionChange(t, c, "catalogue-films", "enable")
	// Long escaped metadata uses smaller batches, not an oversized runtime message.
	items = items[:8]
	for i := range items {
		items[i].Metadata.Title = strings.Repeat("<", 4096)
		items[i].Metadata.Artist = strings.Repeat("<", 4096)
		items[i].Metadata.Album = strings.Repeat("<", 4096)
	}
	if warnings := c.projectCatalogue(context.Background(), items); len(warnings) != 0 {
		t.Fatal(warnings)
	}
}
