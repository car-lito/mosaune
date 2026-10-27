package main

import (
	"context"
	d "mosaune/extensionkit/capabilities/detailheader/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func headerExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, e := os.ReadFile("extension-kit/examples/detailheader/extension.json")
	if e != nil {
		t.Fatal(e)
	}
	var m packageformat.ExecutableManifest
	if e = protocol.DecodeOne(raw, &m); e != nil {
		t.Fatal(e)
	}
	m.Module.Version = v
	code, e := os.ReadFile("extension-kit/examples/detailheader/module.wasm")
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
func TestDetailHeaderLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	q := d.Query{Kind: "films", Title: "A title", Year: 2020, Runtime: 120}
	check := func(c *catalog, owner string) *detailHeaderView {
		t.Helper()
		v, e := c.projectDetailHeader(ctx, q)
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
	v := check(c, "mosaune.media.header")
	if v.Title != "A title" || len(v.Facts) != 2 || v.Facts[1].Text != "120 min" {
		t.Fatal(v)
	}
	raw, pack := headerExample(t, "1.0.0")
	installRecognitionExample(t, c, raw, pack)
	recognitionChange(t, c, pack.Manifest.Module.ID, "enable")
	check(c, "mosaune.media.header")
	active, done, _, _, e := c.beginRecognition(ctx, "detail-header")
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
	if len(v.Facts) != 0 || v.Eyebrow != "DANS LE CADRE" {
		t.Fatal("external module not used", v)
	}
	q.GroupTitle = "Custom collection"
	if v = check(c, pack.Manifest.Module.ID); v.Title != "Custom collection" {
		t.Fatal("catalogue title ignored", v)
	}
	q.Track = true
	if v = check(c, pack.Manifest.Module.ID); v.Title != "A title" {
		t.Fatal("track title replaced", v)
	}
	q.Track = false
	q.GroupTitle = ""
	raw, next := headerExample(t, "1.1.0")
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
	recognitionChange(t, reopened, "detail-header", "select")
	check(reopened, "mosaune.media.header")
	reopened.extensionSafeMode = true
	check(reopened, "")
}
func TestDetailHeaderPermissionAndPreservation(t *testing.T) {
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
		r := a.request("POST", "/api/v1/detail-header", body)
		if r.Code != code {
			t.Fatal(r.Code, r.Body.String())
		}
		return r.Body.String()
	}
	before := request(admin, body, 200)
	if !strings.Contains(before, "mosaune.media.header") || !strings.Contains(before, "Show") || strings.Contains(before, "javascript:") || strings.Contains(before, files) {
		t.Fatal(before)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{lib.ID: {View: true, Play: true}})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	request(reader, body, 200)
	request(reader, map[string]any{"kind": "series", "ids": []string{id}, "trackId": id}, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id}, "title": "forged"}, 400)
	request(reader, map[string]any{"kind": "films", "workId": work, "ids": []string{id}}, 404)
	admin.create("Hidden", accountTestPassword, map[string]libraryGrant{})
	hidden := &accountClient{t: t, h: h}
	hidden.enter("login", map[string]string{"name": "Hidden", "password": accountTestPassword})
	request(hidden, body, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id, id}}, 404)
	request(reader, map[string]any{"kind": "series", "ids": []string{id}, "workId": "missing"}, 404)
	recognitionChange(t, c, "detail-header", "disable")
	if after := request(reader, body, 200); !strings.Contains(after, "\"view\":null") || !strings.Contains(after, "conservés") {
		t.Fatal(after)
	}
	recognitionChange(t, c, "detail-header", "enable")
	if after := request(reader, body, 200); !strings.Contains(after, "mosaune.media.header") {
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
func TestDetailHeaderMigrationAndPolicy(t *testing.T) {
	c, dir := testCatalog(t)
	recognitionChange(t, c, "detail-header", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-header'; PRAGMA user_version=24;"); e != nil {
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
		if v.Kind == "detail-header" && (!v.Enabled || !v.Selected) {
			t.Fatal("new presentation missing")
		}
	}
	_, pack := headerExample(t, "1.0.0")
	pack.Manifest.Module.Services = []protocol.ServiceRequirement{{Name: "bad.service.v1", Operations: []string{"read"}}}
	if _, e = validateRecognitionImport(pack); e == nil {
		t.Fatal("service accepted")
	}
}

func removeHeaderForLegacyFixture(t *testing.T, c *catalog) {
	t.Helper()
	recognitionChange(t, c, "detail-header", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-header'"); e != nil {
		t.Fatal(e)
	}
}

func TestDetailHeaderTrackAndAlbum(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "track.mp3", "fake")
	if _, e := c.scan(ctx, library{Name: "Music", Kind: "music", Path: dir}, true); e != nil {
		t.Fatal(e)
	}
	items, e := c.items(ctx)
	if e != nil || len(items) != 1 {
		t.Fatal(items, e)
	}
	item := items[0]
	md := Metadata{Title: "The song", Album: "The album", Artist: "Singer", Provider: "fixture", ProviderID: "release", Year: 2022}
	if e = c.saveMetadata(ctx, item, md); e != nil {
		t.Fatal(e)
	}
	work, e := c.rememberWork(ctx, "music", Metadata{Title: "The album", Provider: "fixture", ProviderID: "release", Year: 2022, Edition: "Deluxe"}, true)
	if e != nil {
		t.Fatal(e)
	}
	check := func(body any, code int, title string) {
		t.Helper()
		r := admin.request("POST", "/api/v1/detail-header", body)
		if r.Code != code {
			t.Fatal(r.Code, r.Body.String())
		}
		if title != "" && !strings.Contains(r.Body.String(), "\"title\":\""+title+"\"") {
			t.Fatal(r.Body.String())
		}
	}
	check(map[string]any{"kind": "music", "ids": []string{item.ID}}, 200, "The album")
	check(map[string]any{"kind": "music", "workId": work, "ids": []string{item.ID}}, 200, "The album")
	check(map[string]any{"kind": "music", "workId": work, "ids": []string{item.ID}, "trackId": item.ID}, 200, "The song")
	check(map[string]any{"kind": "music", "workId": work, "ids": []string{item.ID}, "trackId": "missing"}, 404, "")
	check(map[string]any{"kind": "music", "workId": work, "ids": []string{}, "trackId": item.ID}, 404, "")
	check(map[string]any{"kind": "music", "workId": work, "ids": []string{}}, 200, "The album")
}
