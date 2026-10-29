package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIScreensValidation(t *testing.T) {
	for _, p := range append(screenPackages(), screenExamplePackage()) {
		if _, e := parseExtensionPackage(screenArchive(p)); e != nil {
			t.Fatal(e)
		}
	}
	var original extensionManifest
	for _, p := range screenPackages() {
		if p.Manifest.Pages[0].Screen == "films" {
			original = p.Manifest
		}
	}
	cases := map[string]func(*extensionManifest){
		"host settings":      func(m *extensionManifest) { m.Pages[0].Screen = "settings" },
		"duplicate claim":    func(m *extensionManifest) { p := m.Pages[0]; p.ID = "other"; m.Pages = append(m.Pages, p) },
		"wrong family":       func(m *extensionManifest) { m.Pages[0].Kinds = []string{"music"} },
		"hidden claim":       func(m *extensionManifest) { m.Pages[0].Placement = "hidden" },
		"unsupported source": func(m *extensionManifest) { m.Pages[0].Blocks[0].Source = "files" },
		"remote URL":         func(m *extensionManifest) { m.Pages[0].Blocks[3].Provider = "http://localhost/private" },
		"unsupported feed":   func(m *extensionManifest) { m.Pages[0].Blocks[3].Shelf = "secrets" },
		"duplicate remote":   func(m *extensionManifest) { m.Pages[0].Blocks = append(m.Pages[0].Blocks, m.Pages[0].Blocks[3]) },
		"old core":           func(m *extensionManifest) { m.MinCore = "0.0.41" },
		"mixed fields":       func(m *extensionManifest) { m.Pages[0].Blocks[0].Type = "text" },
	}
	raw, _ := json.Marshal(original)
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var m extensionManifest
			_ = json.Unmarshal(raw, &m)
			change(&m)
			if e := validateExtension(m); e == nil {
				t.Fatal("invalid screen accepted")
			}
		})
	}
}
func TestUIScreensReplacementPermissionsAndRemoval(t *testing.T) {
	c, _, a := accountFixture(t)
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "Un film (2024).mkv"), []byte("media fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	if r := a.request("POST", "/api/v1/libraries", map[string]string{"name": "Films conservés", "kind": "films", "path": dir}); r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	guest, _ := a.create("Reader", accountTestPassword, nil)
	g := &accountClient{t: t, h: a.h}
	g.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	var supplied extensionPackage
	for _, p := range screenPackages() {
		if p.Manifest.Pages[0].Screen == "films" {
			supplied = p
		}
	}
	route := func(p extensionPackage) string {
		return "/api/v1/ui/pages/" + p.Manifest.ID + "/" + p.Manifest.Pages[0].ID
	}
	state := func(p extensionPackage, on bool, want int) {
		t.Helper()
		r := a.request("PUT", "/api/v1/extensions/"+p.Manifest.ID+"/state", map[string]any{"checksum": p.Checksum, "enabled": on})
		if r.Code != want {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	nav := func(client *accountClient) []uiPage {
		t.Helper()
		r := client.request("GET", "/api/v1/ui/navigation", nil)
		if r.Code != 200 {
			t.Fatal(r.Code)
		}
		var pages []uiPage
		if e := json.Unmarshal(r.Body.Bytes(), &pages); e != nil {
			t.Fatal(e)
		}
		return pages
	}
	owner := func(client *accountClient, screen string) string {
		t.Helper()
		found := ""
		for _, p := range nav(client) {
			if p.Screen == screen {
				if found != "" {
					t.Fatal("two owners", screen)
				}
				found = p.Owner
			}
			if p.ID == screen && p.Owner == "" {
				t.Fatal("core fallback", screen)
			}
		}
		return found
	}
	if owner(a, "films") != supplied.Manifest.ID || owner(g, "films") != supplied.Manifest.ID {
		t.Fatal("supplied screens not granted")
	}
	if r := g.request("GET", route(supplied), nil); r.Code != 200 {
		t.Fatal("new account missing screen", r.Code)
	}
	// Ordinary external manifest: the host knows no replacement ID and no alternate layout.
	m := supplied.Manifest
	m.ID = "atelier.cinema"
	m.Name = "Cinéma compact"
	m.Publisher = "Atelier"
	m.Pages[0].Label = "Mon cinéma"
	m.Pages[0].Blocks = []uiBlock{{Type: "feed", View: "films", Source: "recent", Display: "grid", Title: "Tous mes films"}}
	raw, _ := json.Marshal(m)
	archive := extensionZIP(t, raw, "extension.json")
	replacement, e := parseExtensionPackage(archive)
	if e != nil {
		t.Fatal(e)
	}
	if r := extensionUpload(a, "install", archive, replacement.Checksum, ""); r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	state(replacement, true, 409)
	if owner(a, "films") != supplied.Manifest.ID {
		t.Fatal("conflict changed selection")
	}
	state(supplied, false, 204)
	if owner(a, "films") != "" {
		t.Fatal("disabled screen retained")
	}
	if owner(a, "music") == "" {
		t.Fatal("unrelated screen lost")
	}
	if r := a.request("GET", route(supplied), nil); r.Code != 404 {
		t.Fatal("disabled direct route", r.Code)
	}
	state(replacement, true, 204)
	if owner(a, "films") != m.ID {
		t.Fatal("replacement unused")
	}
	if owner(g, "films") != "" {
		t.Fatal("ungranted replacement leaked")
	}
	if r := a.request("PUT", "/api/v1/extensions/"+m.ID+"/access", map[string]any{"users": []string{guest.ID}, "checksum": replacement.Checksum}); r.Code != 204 {
		t.Fatal(r.Code)
	}
	if owner(g, "films") != m.ID {
		t.Fatal("grant ignored")
	}
	// Account restrictions remain authoritative even with an extension grant.
	guest.Kinds = []string{"music"}
	raw, _ = json.Marshal(guest)
	if _, e = c.db.Exec("UPDATE accounts SET data=? WHERE id=?", string(raw), guest.ID); e != nil {
		t.Fatal(e)
	}
	if owner(g, "films") != "" {
		t.Fatal("kind restriction bypassed")
	}
	if r := g.request("GET", route(replacement), nil); r.Code != 404 {
		t.Fatal("restricted direct route", r.Code)
	}
	c.extensionSafeMode = true
	if owner(a, "films") != "" {
		t.Fatal("safe mode screen")
	}
	c.extensionSafeMode = false
	// Data belongs to the host, not the screen package. Check all shared stores.
	counts := func() map[string]int {
		out := map[string]int{}
		for _, table := range []string{"libraries", "media", "metadata", "accounts"} {
			var n int
			if e := c.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); e != nil {
				t.Fatal(e)
			}
			out[table] = n
		}
		return out
	}
	before := counts()
	if r := a.request("DELETE", "/api/v1/extensions/"+m.ID, map[string]string{"checksum": replacement.Checksum}); r.Code != 204 {
		t.Fatal(r.Code)
	}
	if owner(a, "films") != "" {
		t.Fatal("implicit fallback after deletion")
	}
	for table, n := range counts() {
		if before[table] != n {
			t.Fatal("shared store changed", table)
		}
	}
	if e = migrateUIScreens(c.db); e != nil {
		t.Fatal(e)
	}
	if owner(a, "films") != "" {
		t.Fatal("migration reactivated screen")
	}
	state(supplied, true, 204)
	if owner(a, "films") != supplied.Manifest.ID {
		t.Fatal("explicit restore failed")
	}
	if _, e = c.repositoryPackage(context.Background(), bundledRepository(), supplied.Manifest.ID, "1.0.0"); e != nil {
		t.Fatal("bundled reinstall unavailable", e)
	}
	// Removing every screen leaves the recovery/settings navigation, not a dead shell.
	for _, p := range screenPackages() {
		state(p, false, 204)
	}
	for _, p := range nav(a) {
		if p.Screen != "" {
			t.Fatal("screen left")
		}
	}
	raw, _ = json.Marshal(nav(a))
	if !strings.Contains(string(raw), "extensions") || !strings.Contains(string(raw), "account") {
		t.Fatal("recovery unavailable")
	}
}

func TestUIScreensPersistWithoutReseeding(t *testing.T) {
	dir := t.TempDir()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	packages := screenPackages()
	removed, disabled := packages[0].Manifest.ID, packages[1].Manifest.ID
	if _, e = c.db.Exec("DELETE FROM extensions WHERE id=?", removed); e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec("UPDATE extensions SET enabled=0 WHERE id=?", disabled); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	var n int
	if e = c.db.QueryRow("SELECT COUNT(*) FROM extensions WHERE id=?", removed).Scan(&n); e != nil || n != 0 {
		t.Fatal("removed package restored", e, n)
	}
	if e = c.db.QueryRow("SELECT enabled FROM extensions WHERE id=?", disabled).Scan(&n); e != nil || n != 0 {
		t.Fatal("disabled package enabled", e, n)
	}
}
