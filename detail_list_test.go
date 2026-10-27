package main

import (
	"context"
	d "mosaune/extensionkit/capabilities/detaillist/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func detailExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, e := os.ReadFile("extension-kit/examples/detaillist/extension.json")
	if e != nil {
		t.Fatal(e)
	}
	var m packageformat.ExecutableManifest
	if e = protocol.DecodeOne(raw, &m); e != nil {
		t.Fatal(e)
	}
	m.Module.Version = v
	code, e := os.ReadFile("extension-kit/examples/detaillist/module.wasm")
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
func TestDetailListLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	q := d.Query{Entries: []d.Entry{{ID: "ten", Title: "Ten", Season: 1, Episode: 10}, {ID: "two", Title: "Two", Season: 1, Episode: 2}}, Seasons: []d.Season{{Number: 1, Name: "Première saison", Episodes: 10}, {Number: 2, Name: "À venir", Episodes: 6}}}
	check := func(c *catalog, owner string) *detailListView {
		t.Helper()
		v, e := c.projectDetailList(ctx, q, []string{"cover1", "cover2"})
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
	v := check(c, "mosaune.series.details")
	if len(v.Groups) != 2 || v.Rows[0].ID != "two" || v.Groups[0].Total != 10 || *v.Groups[1].RequestSeason != 2 {
		t.Fatal(v)
	}
	raw, pack := detailExample(t, "1.0.0")
	installRecognitionExample(t, c, raw, pack)
	recognitionChange(t, c, pack.Manifest.Module.ID, "enable")
	check(c, "mosaune.series.details")
	active, done, _, _, e := c.beginRecognition(ctx, "detail-series")
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
	if len(v.Groups) != 0 || v.Files != "Épisodes — vue simple" {
		t.Fatal("external module not used", v)
	}
	raw, next := detailExample(t, "1.1.0")
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
	recognitionChange(t, reopened, "detail-series", "select")
	check(reopened, "mosaune.series.details")
	reopened.extensionSafeMode = true
	check(reopened, "")
}
func TestDetailListPermissionAndPreservation(t *testing.T) {
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
	work, e := c.rememberWork(ctx, "series", Metadata{Provider: "fixture", ProviderID: "series", Title: "Show", Seasons: []SeasonInfo{{Number: 1, Name: "Première", Episodes: 12, Image: assetURL("tmdb", "/cover.jpg")}}}, true)
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
		r := a.request("POST", "/api/v1/detail-list", body)
		if r.Code != code {
			t.Fatal(r.Code, r.Body.String())
		}
		return r.Body.String()
	}
	before := request(admin, body, 200)
	if !strings.Contains(before, "mosaune.series.details") || !strings.Contains(before, "Première") || strings.Contains(before, files) {
		t.Fatal(before)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{lib.ID: {View: true, Play: true}})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	request(reader, body, 200)
	admin.create("Hidden", accountTestPassword, map[string]libraryGrant{})
	hidden := &accountClient{t: t, h: h}
	hidden.enter("login", map[string]string{"name": "Hidden", "password": accountTestPassword})
	request(hidden, body, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id, id}}, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id}, "workId": "missing"}, 404)
	recognitionChange(t, c, "detail-series", "disable")
	if after := request(reader, body, 200); !strings.Contains(after, "\"view\":null") || !strings.Contains(after, "individuellement") {
		t.Fatal(after)
	}
	recognitionChange(t, c, "detail-series", "enable")
	if after := request(reader, body, 200); !strings.Contains(after, "mosaune.series.details") {
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
func TestDetailListMigrationAndPolicy(t *testing.T) {
	c, dir := testCatalog(t)
	removeSectionsForLegacyFixture(t, c)
	recognitionChange(t, c, "detail-series", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-series'; PRAGMA user_version=22;"); e != nil {
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
		if v.Kind == "detail-series" && (!v.Enabled || !v.Selected) {
			t.Fatal("new presentation missing")
		}
	}
	_, pack := detailExample(t, "1.0.0")
	pack.Manifest.Module.Services = []protocol.ServiceRequirement{{Name: "bad.service.v1", Operations: []string{"read"}}}
	if _, e = validateRecognitionImport(pack); e == nil {
		t.Fatal("service accepted")
	}
}
