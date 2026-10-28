package main

import (
	"context"
	"encoding/json"
	"fmt"
	d "mosaune/extensionkit/capabilities/detailexperience/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func experienceExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, e := os.ReadFile("extension-kit/examples/detailexperience/extension.json")
	if e != nil {
		t.Fatal(e)
	}
	var m packageformat.ExecutableManifest
	if e = protocol.DecodeOne(raw, &m); e != nil {
		t.Fatal(e)
	}
	m.Module.Version = v
	code, e := os.ReadFile("extension-kit/examples/detailexperience/module.wasm")
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
func TestDetailExperienceLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	q := d.Query{Kind: "films", Work: true, Files: []d.File{{ID: "file", Name: "A title"}}}
	check := func(c *catalog, owner string) *detailExperienceView {
		t.Helper()
		v, e := c.projectDetailExperience(ctx, q)
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
	v := check(c, "mosaune.media.experience")
	if len(v.Actions) == 0 || v.Cover != "portrait" || len(v.Sections) != 1 {
		t.Fatal(v)
	}
	raw, pack := experienceExample(t, "1.0.0")
	installRecognitionExample(t, c, raw, pack)
	recognitionChange(t, c, pack.Manifest.Module.ID, "enable")
	check(c, "mosaune.media.experience")
	active, done, _, _, e := c.beginRecognition(ctx, "detail-experience")
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
	if len(v.Actions) != 0 || v.Cover != "square" || v.Suggestions.Enabled || v.Sections[0].Title != "Lecture directe" {
		t.Fatal("external module not used", v)
	}
	raw, next := experienceExample(t, "1.1.0")
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
	recognitionChange(t, reopened, "detail-experience", "select")
	check(reopened, "mosaune.media.experience")
	reopened.extensionSafeMode = true
	check(reopened, "")
}

func removeExperienceForLegacyFixture(t *testing.T, c *catalog) {
	t.Helper()
	recognitionChange(t, c, "detail-experience", "remove")
	if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='detail-experience'"); e != nil {
		t.Fatal(e)
	}
}
func TestDetailExperienceMigrationAndPolicy(t *testing.T) {
	c, dir := testCatalog(t)
	removeExperienceForLegacyFixture(t, c)
	if _, e := c.db.Exec("PRAGMA user_version=25"); e != nil {
		t.Fatal(e)
	}
	recognitionChange(t, c, "detail-header", "disable")
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
		if v.Kind == "detail-header" && v.Enabled {
			t.Fatal("reenabled old choice")
		}
		if v.Kind == "detail-experience" && (!v.Enabled || !v.Selected) {
			t.Fatal("new module missing")
		}
	}
	_, pack := experienceExample(t, "1.0.0")
	pack.Manifest.Module.Services = []protocol.ServiceRequirement{{Name: "bad.service.v1", Operations: []string{"read"}}}
	if _, e = validateRecognitionImport(pack); e == nil {
		t.Fatal("unexpected services")
	}
}
func TestDetailExperiencePermissionsAndPreservation(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "Song.mp3", "fake")
	lib, e := c.scan(ctx, library{Name: "Visible", Kind: "music", Path: dir}, true)
	if e != nil {
		t.Fatal(e)
	}
	items, e := c.items(ctx)
	if e != nil || len(items) != 1 {
		t.Fatal(items, e)
	}
	id := items[0].ID
	md := Metadata{Title: "Song", Artist: "Singer", Album: "Album", Track: 1, Provider: "fixture", ProviderID: "album"}
	if e = c.saveMetadata(ctx, items[0], md); e != nil {
		t.Fatal(e)
	}
	work, e := c.rememberWork(ctx, "music", Metadata{Title: "Album", Provider: "fixture", ProviderID: "album", Contributors: []Contributor{{Name: "Singer", ID: "artist"}}, Tracks: []CatalogTrack{{Title: "Song", Number: 1}, {Title: "Missing", Number: 2}}}, true)
	if e != nil {
		t.Fatal(e)
	}
	body := map[string]any{"kind": "music", "ids": []string{id}, "workId": work}
	request := func(a *accountClient, b any, code int) string {
		t.Helper()
		r := a.request("POST", "/api/v1/detail-experience", b)
		if r.Code != code {
			t.Fatal(r.Code, r.Body.String())
		}
		return r.Body.String()
	}
	before := request(admin, body, 200)
	if !strings.Contains(before, "mosaune.media.experience") || !strings.Contains(before, "Missing") || strings.Contains(before, dir) {
		t.Fatal(before)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{lib.ID: {View: true}})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	request(reader, body, 200)
	request(reader, map[string]any{"kind": "music", "ids": []string{id}, "trackId": "hidden"}, 404)
	request(reader, map[string]any{"kind": "music", "ids": []string{id}, "title": "forged"}, 400)
	request(reader, map[string]any{"kind": "films", "ids": []string{id}, "workId": work}, 404)
	admin.create("Hidden", accountTestPassword, map[string]libraryGrant{})
	hidden := &accountClient{t: t, h: h}
	hidden.enter("login", map[string]string{"name": "Hidden", "password": accountTestPassword})
	request(hidden, body, 404)
	recognitionChange(t, c, "detail-experience", "disable")
	if after := request(reader, body, 200); !strings.Contains(after, "\"view\":null") || !strings.Contains(after, "conservés") {
		t.Fatal(after)
	}
	recognitionChange(t, c, "detail-experience", "enable")
	after := request(reader, body, 200)
	if !strings.Contains(after, "mosaune.media.experience") {
		t.Fatal(after)
	}
	list, _ := c.items(ctx)
	if len(list) != 1 || list[0].Metadata.ProviderID != "album" {
		t.Fatal("data lost")
	}
	if raw, e := os.ReadFile(filepath.Join(dir, "Song.mp3")); e != nil || string(raw) != "fake" {
		t.Fatal("source changed")
	}
}
func TestDetailExperienceLargeEnvelope(t *testing.T) {
	c, _ := testCatalog(t)
	q := d.Query{Kind: "music", Files: []d.File{}, Tracks: []d.Track{}}
	for i := 0; i < 1000; i++ {
		name := fmt.Sprintf("Track %d", i)
		q.Files = append(q.Files, d.File{ID: fmt.Sprint(i), Title: name, Name: name, Number: i + 1})
		q.Tracks = append(q.Tracks, d.Track{Title: name, Number: i + 1})
	}
	raw, _ := json.Marshal(q)
	if len(raw) <= 64<<10 {
		t.Fatal("fixture must exceed default envelope")
	}
	v, e := c.projectDetailExperience(testContext(context.Background()), q)
	if e != nil || v == nil || len(v.Sections[0].Rows) != 1000 {
		t.Fatal(v, e)
	}
}
