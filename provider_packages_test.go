package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	f "mosaune/extensionkit/packageformat"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func providerUpload(a *accountClient, action string, raw []byte, checksum, previous string) *httptest.ResponseRecorder {
	a.t.Helper()
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/provider-packages/"+action, bytes.NewReader(raw))
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
func TestProviderPackageInstallUpdateFailureRollbackAndRemoval(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := c.provider("tmdb").(*tmdbProvider)
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	removeTMDBPackageFixture(t, p)
	p.extension.key = pub
	resign := func(example []byte, changedVersion string) f.Package {
		t.Helper()
		pack, e := f.Parse(example, trustedProviderKey(), version)
		if e != nil {
			t.Fatal(e)
		}
		if changedVersion != "" {
			pack.Manifest.Descriptor.Version = changedVersion
		}
		raw, e := f.Create(pack.Manifest, pack.Code, key)
		if e != nil {
			t.Fatal(e)
		}
		pack, e = f.Parse(raw, pub, version)
		if e != nil {
			t.Fatal(e)
		}
		return pack
	}
	first, next := resign(providerExample, ""), resign(providerUpdateExample, "")
	ctx := context.Background()
	if err := c.setProviderCredential(ctx, "tmdb", "fixture-kept-token-123456789"); err != nil {
		t.Fatal(err)
	}
	_, err := c.db.Exec(`INSERT INTO libraries VALUES('keep','Keep','fixture-path','films');INSERT INTO media(id,library_id,relative_path,name,size,modified,revision,position) VALUES('keep','keep','keep.mkv','Keep',1,1,'rev',42);INSERT INTO metadata VALUES('keep','rev','{"manual":true,"title":"Keep"}');`)
	if err != nil {
		t.Fatal(err)
	}
	if w := providerUpload(admin, "preview", first.Archive, "", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := providerUpload(admin, "install", first.Archive, first.Checksum, "stale"); w.Code != 409 {
		t.Fatal("stale install", w.Code)
	}
	if w := providerUpload(admin, "install", first.Archive, first.Checksum, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if p.extensionStatus()["enabled"] {
		t.Fatal("first install auto-enabled")
	}
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"id":42,"title":"From installed module"}]}`)), Header: make(http.Header)}, nil
	})}
	result, err := p.Search(ctx, "fixture-key", MetadataQuery{Kind: "films", Title: "Fixture"})
	if err != nil || len(result) != 1 || result[0].ID != "42" {
		t.Fatal("installed module bridge", result, err)
	}
	if _, err = c.db.Exec(`CREATE TEMP TRIGGER fail_package_update BEFORE UPDATE ON provider_packages BEGIN SELECT RAISE(ABORT,'fixture write failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if w := providerUpload(admin, "install", next.Archive, next.Checksum, first.Checksum); w.Code != 409 {
		t.Fatal("write failure ignored", w.Code)
	}
	unchanged, e := p.packageState(ctx)
	if e != nil || unchanged.Package.Checksum != first.Checksum || !unchanged.Runtime["enabled"] || unchanged.PreviousChecksum != "" {
		t.Fatal("failed transaction changed state", unchanged, e)
	}
	if _, err = c.db.Exec(`DROP TRIGGER fail_package_update`); err != nil {
		t.Fatal(err)
	}
	if w := providerUpload(admin, "install", next.Archive, next.Checksum, first.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	state, err := p.packageState(ctx)
	if err != nil || state.Package.Checksum != next.Checksum || state.PreviousVersion != "0.4.0" || !state.Runtime["enabled"] {
		t.Fatal(state, err)
	}
	// Signed and compatible, but its real health response has the wrong version.
	broken := resign(providerUpdateExample, "0.4.2")
	if w := providerUpload(admin, "install", broken.Archive, broken.Checksum, next.Checksum); w.Code != 409 {
		t.Fatal("broken update accepted", w.Code, w.Body.String())
	}
	state, _ = p.packageState(ctx)
	if state.Package.Checksum != next.Checksum || !state.Runtime["enabled"] || state.PreviousChecksum != first.Checksum {
		t.Fatal("failed update changed state")
	}
	c.extensionSafeMode = true
	if w := providerUpload(admin, "install", broken.Archive, broken.Checksum, next.Checksum); w.Code != 409 {
		t.Fatal("safe mode", w.Code)
	}
	if p.extensionStatus()["effective"] {
		t.Fatal("safe mode ignored")
	}
	c.extensionSafeMode = false
	if w := admin.request("POST", "/api/v1/provider-packages/tmdb/rollback", map[string]string{"checksum": next.Checksum, "previousChecksum": first.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	state, _ = p.packageState(ctx)
	if state.Package.Checksum != first.Checksum || state.PreviousChecksum != next.Checksum || !state.Runtime["enabled"] {
		t.Fatal("rollback", state)
	}
	// Reopening loads the archive stored in SQLite, not the bundled program.
	var databaseFile string
	if err = c.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&databaseFile); err != nil {
		t.Fatal(err)
	}
	reopened, e := openCatalog(filepath.Dir(databaseFile))
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.close()
	rp := reopened.provider("tmdb").(*tmdbProvider)
	rp.extension.key = pub
	loaded, e := rp.packageState(ctx)
	if e != nil || loaded.Package.Checksum != first.Checksum || !loaded.Runtime["enabled"] {
		t.Fatal(loaded, e)
	}
	engine, release, e := rp.extensionEngine(ctx)
	if e != nil || engine == nil {
		t.Fatal("restart module", e)
	}
	release()
	rp.closeExtension()
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	for _, path := range []string{"/api/v1/provider-packages/tmdb", "/api/v1/provider-packages/example"} {
		if w := reader.request("GET", path, nil); w.Code != 403 {
			t.Fatal("permission", w.Code)
		}
	}
	if w := providerUpload(reader, "install", next.Archive, next.Checksum, first.Checksum); w.Code != 403 {
		t.Fatal("install permission", w.Code)
	}
	if w := admin.request("DELETE", "/api/v1/provider-packages/tmdb", map[string]string{"checksum": next.Checksum}); w.Code != 409 {
		t.Fatal("stale removal", w.Code)
	}
	if w := admin.request("DELETE", "/api/v1/provider-packages/tmdb", map[string]string{"checksum": first.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	state, e = p.packageState(ctx)
	if e != nil || state.Package != nil || state.Runtime["enabled"] {
		t.Fatal(state, e)
	}
	var position float64
	var metadata string
	_ = c.db.QueryRow(`SELECT position FROM media WHERE id='keep'`).Scan(&position)
	_ = c.db.QueryRow(`SELECT data FROM metadata WHERE media_id='keep'`).Scan(&metadata)
	if position != 42 || !strings.Contains(metadata, `"manual":true`) || c.credential(ctx, "tmdb") != "fixture-kept-token-123456789" {
		t.Fatal("shared data deleted")
	}
	// Unknown signatures are rejected before a provider engine is created.
	unknown := append([]byte{}, next.Archive...)
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	p.extension.key = otherPub
	if w := providerUpload(admin, "preview", unknown, "", ""); w.Code != 400 {
		t.Fatal("unknown signer accepted", w.Code)
	}
}
func TestProviderPackageMigrationBackup(t *testing.T) {
	dir := t.TempDir()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	removeRecognitionForLegacyFixture(t, c)
	_, e = c.db.Exec(`DROP TABLE provider_packages;PRAGMA user_version=11`)
	if e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	var v int
	_ = c.db.QueryRow(`PRAGMA user_version`).Scan(&v)
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if v != catalogSchemaVersion || len(backups) != 1 {
		t.Fatal(v, backups)
	}
	w := httptest.NewRecorder()
	sendJSON(w, 200, providerPackageState{})
	var data map[string]any
	if json.Unmarshal(w.Body.Bytes(), &data) != nil {
		t.Fatal("state encoding")
	}
}
