package main

import (
	"context"
	d "mosaune/extensionkit/capabilities/detailsections/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sectionsExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, e := os.ReadFile("extension-kit/examples/detailsections/extension.json")
	if e != nil {
		t.Fatal(e)
	}
	var m packageformat.ExecutableManifest
	if e = protocol.DecodeOne(raw, &m); e != nil {
		t.Fatal(e)
	}
	m.Module.Version = v
	code, e := os.ReadFile("extension-kit/examples/detailsections/module.wasm")
	if e != nil {
		t.Fatal(e)
	}
	raw, e = packageformat.CreateExecutable(m, code)
	if e != nil {
		t.Fatal(e)
	}
	pack, e := packageformat.ParseExecutable(raw, version)
	if e != nil {
		t.Fatal(e)
	}
	return raw, pack
}
func TestDetailSectionsLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	q := d.Query{Kind: "films", Overview: "A story", People: []d.Person{{Name: "Person"}}}
	check := func(c *catalog, owner string) *detailSectionsView {
		t.Helper()
		v, e := c.projectDetailSections(ctx, q)
		if owner == "" {
			if e == nil || v != nil {
				t.Fatal("disabled module returned layout")
			}
			return nil
		}
		if e != nil || v.Owner != owner {
			t.Fatal(v, e)
		}
		return v
	}
	v := check(c, "mosaune.media.information")
	if len(v.Blocks) != 3 || v.Blocks[0].Text != "A story" {
		t.Fatal(v)
	}
	raw, pack := sectionsExample(t, "1.0.0")
	installRecognitionExample(t, c, raw, pack)
	recognitionChange(t, c, pack.Manifest.Module.ID, "enable")
	check(c, "mosaune.media.information")
	active, done, _, _, e := c.beginRecognition(ctx, "detail-information")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	recognitionChange(t, c, pack.Manifest.Module.ID, "select")
	if active.Err() == nil {
		t.Fatal("old projection not cancelled")
	}
	if unlock, e := c.recognitionCommit(active); e == nil {
		unlock()
		t.Fatal("late projection accepted")
	}
	v = check(c, pack.Manifest.Module.ID)
	if len(v.Blocks) != 1 || v.Blocks[0].Title != "L’essentiel" {
		t.Fatal("external module not used", v)
	}
	raw, next := sectionsExample(t, "1.1.0")
	installRecognitionExample(t, c, raw, next)
	check(c, next.Manifest.Module.ID)
	recognitionChange(t, c, next.Manifest.Module.ID, "restore")
	check(c, next.Manifest.Module.ID)
	recognitionChange(t, c, next.Manifest.Module.ID, "disable")
	check(c, "")
	recognitionChange(t, c, next.Manifest.Module.ID, "enable")
	recognitionChange(t, c, next.Manifest.Module.ID, "remove")
	check(c, "")
	reopened, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.close()
	check(reopened, "")
	recognitionChange(t, reopened, "detail-information", "select")
	check(reopened, "mosaune.media.information")
	reopened.extensionSafeMode = true
	check(reopened, "")
}
func TestDetailSectionsPermissionAndPreservation(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := testContext(context.Background())
	files := t.TempDir()
	putFile(t, files, "Example.S01E02.mkv", "fake")
	lib, e := c.scan(ctx, library{Name: "Visible", Kind: "series", Path: files}, true)
	if e != nil {
		t.Fatal(e)
	}
	items, e := c.items(ctx)
	if e != nil || len(items) != 1 {
		t.Fatal(items, e)
	}
	id := items[0].ID
	work, e := c.rememberWork(ctx, "series", Metadata{Provider: "fixture", ProviderID: "series", Title: "Show", Overview: "A synthetic summary", Contributors: []Contributor{{ID: "person", Name: "Someone", Image: assetURL("tmdb", "/face.jpg")}}, Trailers: []Trailer{{Name: "Watch", URL: "https://www.youtube.com/watch?v=fixture", Language: "fr"}, {Name: "Bad", URL: "javascript:alert(1)"}}, Seasons: []SeasonInfo{{Number: 1, Name: "Première", Episodes: 12, Image: assetURL("tmdb", "/cover.jpg")}}}, true)
	if e != nil {
		t.Fatal(e)
	}
	md := *items[0].Metadata
	md.Provider = "fixture"
	md.ProviderID = "series"
	if e = c.saveMetadata(ctx, items[0], md); e != nil {
		t.Fatal(e)
	}
	body := map[string]any{"kind": "series", "ids": []string{id}, "workId": work}
	request := func(a *accountClient, body any, code int) string {
		t.Helper()
		r := a.request("POST", "/api/v1/detail-sections", body)
		if r.Code != code {
			t.Fatal(r.Code, r.Body.String())
		}
		return r.Body.String()
	}
	before := request(admin, body, 200)
	if !strings.Contains(before, "mosaune.media.information") || !strings.Contains(before, "A synthetic summary") || strings.Contains(before, "javascript:") || strings.Contains(before, files) {
		t.Fatal(before)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{lib.ID: {View: true, Play: true}})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	request(reader, body, 200)
	request(reader, map[string]any{"kind": "films", "workId": work, "ids": []string{id}}, 404)
	admin.create("Hidden", accountTestPassword, map[string]libraryGrant{})
	hidden := &accountClient{t: t, h: h}
	hidden.enter("login", map[string]string{"name": "Hidden", "password": accountTestPassword})
	request(hidden, body, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id, id}}, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id}, "workId": "missing"}, 404)
	recognitionChange(t, c, "detail-information", "disable")
	if after := request(reader, body, 200); !strings.Contains(after, "\"view\":null") || !strings.Contains(after, "conservés") {
		t.Fatal(after)
	}
	recognitionChange(t, c, "detail-information", "enable")
	if after := request(reader, body, 200); !strings.Contains(after, "mosaune.media.information") {
		t.Fatal(after)
	}
	list, _ := c.items(ctx)
	if len(list) != 1 || list[0].Metadata.ProviderID != "series" {
		t.Fatal("metadata lost")
	}
	if raw, e := os.ReadFile(filepath.Join(files, "Example.S01E02.mkv")); e != nil || string(raw) != "fake" {
		t.Fatal("source modified", e)
	}
}
func TestDetailSectionsMigrationAndPolicy(t *testing.T) {
	c, dir := testCatalog(t)
	removeHeaderForLegacyFixture(t, c)
	recognitionChange(t, c, "detail-information", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-information'; PRAGMA user_version=23;"); e != nil {
		t.Fatal(e)
	}
	recognitionChange(t, c, "catalogue-series", "disable")
	c.close()
	next, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer next.close()
	list, _, e := next.recognitionPackages(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range list {
		if v.Kind == "catalogue-series" && v.Enabled {
			t.Fatal("migration reenabled catalogue")
		}
		if v.Kind == "detail-information" && (!v.Enabled || !v.Selected) {
			t.Fatal("new presentation missing")
		}
	}
	_, pack := sectionsExample(t, "1.0.0")
	pack.Manifest.Module.Services = []protocol.ServiceRequirement{{Name: "bad.service.v1", Operations: []string{"read"}}}
	if _, e = validateRecognitionImport(pack); e == nil {
		t.Fatal("service accepted")
	}
}

func removeSectionsForLegacyFixture(t *testing.T, c *catalog) {
	removeHeaderForLegacyFixture(t, c)
	t.Helper()
	recognitionChange(t, c, "detail-information", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-information'"); e != nil {
		t.Fatal(e)
	}
}
