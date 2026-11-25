package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func extensionZIP(t *testing.T, raw []byte, name string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, e := z.Create(name)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Write(raw); e != nil {
		t.Fatal(e)
	}
	if e = z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func extensionUpload(a *accountClient, path string, data []byte, checksum, previous string) *httptest.ResponseRecorder {
	a.t.Helper()
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/extensions/"+path, bytes.NewReader(data))
	r.Header.Set("X-Mosaune-Token", a.token)
	r.Header.Set("X-Mosaune-Package-Checksum", checksum)
	r.Header.Set("X-Mosaune-Previous-Checksum", previous)
	if a.cookie != nil {
		r.AddCookie(a.cookie)
	}
	w := httptest.NewRecorder()
	a.h.ServeHTTP(w, r)
	return w
}
func TestExtensionPackageValidation(t *testing.T) {
	var m extensionManifest
	if e := json.Unmarshal(exampleTheme, &m); e != nil {
		t.Fatal(e)
	}
	if e := validateExtension(m); e != nil {
		t.Fatal("example", e)
	}
	data := extensionZIP(t, exampleTheme, "extension.json")
	p, e := parseExtensionPackage(data)
	if e != nil || p.Checksum == "" {
		t.Fatal(e)
	}
	cases := []struct {
		name   string
		change func(*extensionManifest)
	}{
		{"script permission", func(m *extensionManifest) { m.Permissions = []string{"execute"} }},
		{"unknown dependency", func(m *extensionManifest) { m.Requires = []string{"movies.catalog.v1"} }},
		{"future host", func(m *extensionManifest) { m.MinCore = "999.0.0" }},
		{"future api", func(m *extensionManifest) { m.API = 2 }},
		{"reserved id", func(m *extensionManifest) { m.ID = "mosaune.official" }},
		{"css injection", func(m *extensionManifest) { m.Theme.Light["ink"] = "#ffffff;background:url(https://example.com)" }},
		{"unreadable", func(m *extensionManifest) { m.Theme.Dark["ink"] = m.Theme.Dark["canvas"] }},
		{"extra token", func(m *extensionManifest) { m.Theme.Dark["display"] = "none" }},
		{"executable family", func(m *extensionManifest) { m.Family = "provider" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m extensionManifest
			_ = json.Unmarshal(exampleTheme, &m)
			tc.change(&m)
			raw, _ := json.Marshal(m)
			if _, e := parseExtensionPackage(extensionZIP(t, raw, "extension.json")); e == nil {
				t.Fatal("accepted invalid package")
			}
		})
	}
	for _, name := range []string{"../extension.json", "dir/extension.json", "extension.js", "/extension.json"} {
		if _, e := parseExtensionPackage(extensionZIP(t, exampleTheme, name)); e == nil {
			t.Fatal("accepted", name)
		}
	}
	if _, e := parseExtensionPackage(extensionZIP(t, bytes.Repeat([]byte(" "), extensionLimit+1), "extension.json")); e == nil {
		t.Fatal("accepted decompression excess")
	}
	if _, e := parseExtensionPackage(bytes.Repeat([]byte("x"), extensionLimit+1)); e == nil {
		t.Fatal("accepted upload excess")
	}
	if _, e := parseExtensionPackage(extensionZIP(t, append(append([]byte{}, exampleTheme...), []byte("{}")...), "extension.json")); e == nil {
		t.Fatal("accepted trailing JSON")
	}
	if _, e := parseExtensionPackage(extensionZIP(t, bytes.Replace(exampleTheme, []byte(`"schema": 1`), []byte(`"script": "evil", "schema": 1`), 1), "extension.json")); e == nil {
		t.Fatal("accepted unknown field")
	}
}

func TestExtensionsLifecycleAndPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	guest, _ := admin.create("Guest", accountTestPassword, nil)
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Guest", "password": accountTestPassword})
	data := extensionZIP(t, exampleTheme, "extension.json")
	p, _ := parseExtensionPackage(data)
	if w := extensionUpload(reader, "install", data, p.Checksum, ""); w.Code != 403 {
		t.Fatal("non-admin install", w.Code)
	}
	originalToken := admin.token
	admin.token = "bad"
	if w := extensionUpload(admin, "install", data, p.Checksum, ""); w.Code != 403 {
		t.Fatal("csrf", w.Code)
	}
	admin.token = originalToken
	if w := extensionUpload(admin, "preview", data, "", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := extensionUpload(admin, "install", data, p.Checksum, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	id := p.Manifest.ID
	path := "/api/v1/extensions/" + id
	if w := reader.request("GET", "/api/v1/extensions", nil); strings.Contains(w.Body.String(), id) {
		t.Fatal("ungranted package listed")
	}
	if w := reader.request("PUT", "/api/v1/account/extension-theme", map[string]string{"id": id}); w.Code != 403 {
		t.Fatal("ungranted selection", w.Code)
	}
	if w := admin.request("PUT", "/api/v1/account/extension-theme", map[string]string{"id": id}); w.Code != 409 {
		t.Fatal("disabled selection", w.Code)
	}
	if w := admin.request("PUT", path+"/state", map[string]any{"enabled": true, "checksum": p.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("PUT", path+"/access", map[string]any{"users": []string{guest.ID}, "checksum": p.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := reader.request("PUT", "/api/v1/account/extension-theme", map[string]string{"id": id}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	appearance := func(a *accountClient) extensionAppearance {
		t.Helper()
		w := a.request("GET", "/api/v1/extensions/appearance", nil)
		var state extensionAppearance
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &state) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		return state
	}
	g := appearance(reader)
	if g.Effective != id || g.Stamp == "" {
		t.Fatal(g)
	}
	if appearance(admin).Selected != "" {
		t.Fatal("preference leaked to admin")
	}
	css := reader.request("GET", "/api/v1/extensions/theme.css?stamp="+g.Stamp, nil)
	if css.Code != 200 || !strings.Contains(css.Body.String(), "#faf3e7") || !strings.Contains(css.Body.String(), `data-extension-theme="`+g.Stamp+`"`) {
		t.Fatal("css", css.Code, css.Body.String())
	}
	if w := admin.request("GET", "/api/v1/extensions/theme.css?stamp="+g.Stamp, nil); w.Body.Len() != 0 {
		t.Fatal("other account got css")
	}
	if css.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("theme cached", css.Header())
	}
	// Removing a grant stops CSS, while keeping the choice for possible restoration.
	admin.request("PUT", path+"/access", map[string]any{"users": []string{}, "checksum": p.Checksum})
	if appearance(reader).Effective != "" {
		t.Fatal("revoked grant active")
	}
	if w := reader.request("GET", "/api/v1/extensions/theme.css?stamp="+g.Stamp, nil); w.Body.Len() != 0 {
		t.Fatal("stale css after revocation")
	}
	admin.request("PUT", path+"/access", map[string]any{"users": []string{guest.ID}, "checksum": p.Checksum})
	admin.request("PUT", path+"/state", map[string]any{"enabled": false, "checksum": p.Checksum})
	if a := appearance(reader); a.Effective != "" || a.Selected != id {
		t.Fatal("disable discarded selection", a)
	}
	admin.request("PUT", path+"/state", map[string]any{"enabled": true, "checksum": p.Checksum})
	c.extensionSafeMode = true
	if a := appearance(reader); !a.SafeMode || a.Effective != "" {
		t.Fatal("safe mode", a)
	}
	if w := admin.request("PUT", path+"/state", map[string]any{"enabled": true, "checksum": p.Checksum}); w.Code != 409 {
		t.Fatal("safe-mode activation", w.Code)
	}
	c.extensionSafeMode = false
	// An update is atomic, preserves grants and preference, and starts disabled.
	var m extensionManifest
	_ = json.Unmarshal(exampleTheme, &m)
	m.Version = "1.1.0"
	m.Theme.Light["accent"] = "#703310"
	raw, _ := json.Marshal(m)
	newData := extensionZIP(t, raw, "extension.json")
	next, e := parseExtensionPackage(newData)
	if e != nil {
		t.Fatal(e)
	}
	if w := extensionUpload(admin, "install", newData, next.Checksum, ""); w.Code != 409 {
		t.Fatal("stale update accepted")
	}
	if w := extensionUpload(admin, "install", newData, next.Checksum, p.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a := appearance(reader); a.Effective != "" || a.Selected != id {
		t.Fatal("update preference", a)
	}
	if w := admin.request("DELETE", path, map[string]string{"checksum": p.Checksum}); w.Code != 409 {
		t.Fatal("old uninstall accepted")
	}
	if w := extensionUpload(admin, "install", data, p.Checksum, next.Checksum); w.Code != 409 {
		t.Fatal("downgrade accepted")
	}
	admin.request("PUT", path+"/state", map[string]any{"enabled": true, "checksum": next.Checksum})
	if appearance(reader).Effective != id {
		t.Fatal("grant lost during update")
	}
	if w := reader.request("DELETE", path, map[string]string{"checksum": next.Checksum}); w.Code != 403 {
		t.Fatal("non-admin uninstall")
	}
	// Keep unrelated data as a sentinel for cleanup scope.
	if _, e = c.db.Exec("INSERT INTO libraries VALUES('keep','Keep','unused','films')"); e != nil {
		t.Fatal(e)
	}
	if w := admin.request("DELETE", path, map[string]string{"checksum": next.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a := appearance(reader); a.Selected != "" || a.Effective != "" {
		t.Fatal("uninstall retained choice", a)
	}
	var count int
	_ = c.db.QueryRow("SELECT COUNT(*) FROM extension_grants WHERE extension_id=?", next.Manifest.ID).Scan(&count)
	if count != 0 {
		t.Fatal("orphan grants")
	}
	_ = c.db.QueryRow("SELECT COUNT(*) FROM libraries WHERE id='keep'").Scan(&count)
	if count != 1 {
		t.Fatal("uninstall deleted shared data")
	}
	if w := extensionUpload(admin, "install", data, p.Checksum, ""); w.Code != 201 {
		t.Fatal("reinstall", w.Code)
	}
	if c.canUseExtension(context.WithValue(context.Background(), identityKey{}, &identity{account: guest}), id) {
		t.Fatal("reinstall inherited removed permissions")
	}
}

func TestExtensionsMigrationAndRestart(t *testing.T) {
	dir := t.TempDir()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	removeRecognitionForLegacyFixture(t, c)
	if _, e = c.db.Exec(`DROP TABLE extension_preferences; DROP TABLE extension_grants; DROP TABLE extensions; PRAGMA user_version=9;`); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("missing migration backup", backups)
	}
	var v int
	_ = c.db.QueryRow("PRAGMA user_version").Scan(&v)
	if v != catalogSchemaVersion {
		t.Fatal(v)
	}
	p, e := parseExtensionPackage(extensionZIP(t, exampleTheme, "extension.json"))
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(p.Manifest)
	if _, e = c.db.Exec("INSERT INTO extensions VALUES(?,?,?,1)", p.Manifest.ID, string(raw), p.Checksum); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	reopened, e := c.extension(context.Background(), p.Manifest.ID)
	if e != nil || !reopened.Enabled || reopened.Checksum != p.Checksum {
		t.Fatal("lost package", e)
	}
	if info, e := os.Stat(backups[0]); e != nil || info.Size() == 0 {
		t.Fatal("empty backup")
	}
}
