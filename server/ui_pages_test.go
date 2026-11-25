package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUIPagesValidation(t *testing.T) {
	if _, e := parseExtensionPackage(uiExampleArchive()); e != nil {
		t.Fatal(e)
	}
	cases := map[string]func(*extensionManifest){
		"duplicate":           func(m *extensionManifest) { m.Pages = append(m.Pages, m.Pages[0]) },
		"traversal":           func(m *extensionManifest) { m.Pages[0].ID = "../home" },
		"owner impersonation": func(m *extensionManifest) { m.Pages[0].Owner = "mosaune.core" },
		"javascript":          func(m *extensionManifest) { m.Pages[0].Blocks[1].Target = "javascript:alert(1)" },
		"external URL":        func(m *extensionManifest) { m.Pages[0].Blocks[1].Target = "https://example.com" },
		"unknown route":       func(m *extensionManifest) { m.Pages[0].Blocks[1].Target = "page:missing" },
		"raw markup":          func(m *extensionManifest) { m.Pages[0].Blocks[0].Type = "html" },
		"unknown service":     func(m *extensionManifest) { m.Pages[0].Blocks[3].View = "secrets" },
		"script access":       func(m *extensionManifest) { m.Permissions = []string{"execute"} },
		"invalid icon":        func(m *extensionManifest) { m.Pages[0].Icon = "https://example.com/pixel" },
		"mix theme":           func(m *extensionManifest) { m.Theme.Light = map[string]string{"ink": "#000000"} },
		"too many pages":      func(m *extensionManifest) { m.Pages = make([]uiPage, 17) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var m extensionManifest
			_ = json.Unmarshal(exampleUIPages, &m)
			change(&m)
			raw, _ := json.Marshal(m)
			if _, e := parseExtensionPackage(extensionZIP(t, raw, "extension.json")); e == nil {
				t.Fatal("invalid package accepted")
			}
		})
	}
}
func TestUIPagesLifecyclePermissionsAndRepository(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	guest, _ := admin.create("Guest", accountTestPassword, nil)
	guest.Kinds = []string{"films"}
	raw, _ := json.Marshal(guest)
	if _, e := c.db.Exec("UPDATE accounts SET data=? WHERE id=?", string(raw), guest.ID); e != nil {
		t.Fatal(e)
	}
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Guest", "password": accountTestPassword})
	var m extensionManifest
	_ = json.Unmarshal(exampleUIPages, &m)
	m.Pages = append(m.Pages, uiPage{ID: "private", Label: "Administration du salon", Placement: "settings", Admin: true}, uiPage{ID: "music-only", Label: "Musique privée", Placement: "main", Kinds: []string{"music"}})
	m.Pages[0].Blocks = append(m.Pages[0].Blocks, uiBlock{Type: "link", Title: "Secret", Target: "page:private"}, uiBlock{Type: "link", Title: "Serveur", Target: "settings:server"})
	raw, _ = json.Marshal(m)
	data := extensionZIP(t, raw, "extension.json")
	p, e := parseExtensionPackage(data)
	if e != nil {
		t.Fatal(e)
	}
	if w := extensionUpload(reader, "install", data, p.Checksum, ""); w.Code != 403 {
		t.Fatal("guest install", w.Code)
	}
	if w := extensionUpload(admin, "install", data, p.Checksum, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	route := "/api/v1/ui/pages/" + m.ID + "/salon"
	base := "/api/v1/extensions/" + m.ID
	check := func(a *accountClient, path string, want int) string {
		t.Helper()
		w := a.request("GET", path, nil)
		if w.Code != want {
			t.Fatal(path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	state := func(enabled bool) {
		t.Helper()
		w := admin.request("PUT", base+"/state", map[string]any{"enabled": enabled, "checksum": p.Checksum})
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	check(admin, route, 404)
	state(true)
	check(admin, route, 200)
	check(reader, route, 404)
	grant := func(users []string) {
		t.Helper()
		w := admin.request("PUT", base+"/access", map[string]any{"users": users, "checksum": p.Checksum})
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	grant([]string{guest.ID})
	body := check(reader, route, 200)
	if strings.Contains(body, `"type":"collection"`) || strings.Contains(body, "Secret") || strings.Contains(body, "Serveur") {
		t.Fatal("restricted blocks leaked", body)
	}
	nav := check(reader, "/api/v1/ui/navigation", 200)
	if !strings.Contains(nav, "atelier.salon") || strings.Contains(nav, "music-only") || strings.Contains(nav, "Administration du salon") || strings.Contains(nav, "Le carnet") {
		t.Fatal("navigation scope", nav)
	}
	check(reader, "/api/v1/ui/pages/atelier.salon/private", 404)
	if w := admin.request("PUT", "/api/v1/account/extension-theme", map[string]string{"id": m.ID}); w.Code != 409 {
		t.Fatal("page selected as theme", w.Code)
	}
	state(false)
	check(reader, route, 404)
	if strings.Contains(check(admin, "/api/v1/ui/navigation", 200), "atelier.salon") {
		t.Fatal("disabled menu visible")
	}
	state(true)
	grant([]string{})
	check(reader, route, 404)
	grant([]string{guest.ID})
	c.extensionSafeMode = true
	check(admin, route, 404)
	c.extensionSafeMode = false
	check(reader, route, 200)
	// A music-only host must not advertise film collections, even to administrators.
	mods, _ := enabledModules("music")
	onlyMusic := &accountClient{t: t, h: newHandler(fstest.MapFS{"index.html": {Data: []byte("ui")}}, mods, c), cookie: admin.cookie, token: admin.token}
	if strings.Contains(check(onlyMusic, "/api/v1/ui/navigation", 200), `"id":"films"`) {
		t.Fatal("disabled family advertised")
	}
	// Updates deactivate pages until explicitly enabled, and stale installs conflict.
	m.Version = "1.1.0"
	raw, _ = json.Marshal(m)
	updated := extensionZIP(t, raw, "extension.json")
	next, _ := parseExtensionPackage(updated)
	if w := extensionUpload(admin, "install", updated, next.Checksum, "wrong"); w.Code != 409 {
		t.Fatal("stale update accepted")
	}
	if w := extensionUpload(admin, "install", updated, next.Checksum, p.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	p = next
	check(reader, route, 404)
	state(true)
	if w := admin.request("DELETE", base, map[string]string{"checksum": p.Checksum}); w.Code != 204 {
		t.Fatal(w.Code)
	}
	check(admin, route, 404)
	// The same package is installable through the bundled/public repository path.
	repo := bundledRepository()
	archive, e := c.repositoryPackage(context.Background(), repo, "atelier.salon", "1.0.0")
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := parseExtensionPackage(archive)
	if e != nil || parsed.Manifest.API != 3 {
		t.Fatal(e)
	}
	for i := range repo.Index.Packages {
		if repo.Index.Packages[i].ID == "atelier.salon" {
			repo.Index.Packages[i].Kind = "theme"
		}
	}
	if _, e = c.repositoryPackage(context.Background(), repo, "atelier.salon", "1.0.0"); e == nil {
		t.Fatal("pages mislabeled as theme accepted")
	}
}
