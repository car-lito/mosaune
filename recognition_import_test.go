package main

import (
	"bytes"
	"context"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func recognitionExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	mraw, err := os.ReadFile("extension-kit/examples/recognition/extension.json")
	if err != nil {
		t.Fatal(err)
	}
	var m packageformat.ExecutableManifest
	if err = protocol.DecodeOne(mraw, &m); err != nil {
		t.Fatal(err)
	}
	m.Module.Version = v
	code, err := os.ReadFile("extension-kit/examples/recognition/module.wasm")
	if err != nil {
		t.Fatal("build contributor example first", err)
	}
	raw, err := packageformat.CreateExecutable(m, code)
	if err != nil {
		t.Fatal(err)
	}
	p, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	return raw, p
}
func installRecognitionExample(t *testing.T, c *catalog, raw []byte, p packageformat.Executable) {
	t.Helper()
	_, revision, err := c.recognitionPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = c.importRecognition(context.Background(), raw, p.Checksum, revision); err != nil {
		t.Fatal(err)
	}
}

func TestExternalRecognitionChoiceUpdateRestoreRemoval(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	raw, p := recognitionExample(t, "1.0.0")
	id := p.Manifest.Module.ID
	installRecognitionExample(t, c, raw, p)
	assertTitle := func(c *catalog, prefix string) {
		t.Helper()
		call, r, done, e := c.openRecognizer(ctx, "films")
		if e != nil {
			t.Fatal(e)
		}
		defer done()
		items, e := r.recognize(call, []string{"Iron.Man.2008.mkv"})
		if e != nil || !strings.HasPrefix(items[0].Title, prefix) {
			t.Fatal(items, e)
		}
	}
	assertTitle(c, "Iron Man") // Installing and enabling do not take over.
	recognitionChange(t, c, id, "enable")
	assertTitle(c, "Iron Man")
	old, _, done, err := c.openRecognizer(ctx, "films")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	recognitionChange(t, c, id, "select")
	if old.Err() == nil {
		t.Fatal("selection did not cancel old scan")
	}
	assertTitle(c, "Exemple — Iron Man")
	// The real scan consumes the imported program, not an embedded substitute.
	files := t.TempDir()
	putFile(t, files, "Local.2020.mkv", "fixture")
	if _, err = c.scan(ctx, library{Kind: "films", Path: files}, true); err != nil {
		t.Fatal(err)
	}
	before := getItems(t, c)
	if len(before) != 1 || before[0].Metadata == nil || !strings.HasPrefix(before[0].Metadata.Title, "Exemple — ") {
		t.Fatal("scan bypassed external module", before)
	}
	active, _, finish, err := c.openRecognizer(ctx, "films")
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	nextRaw, nextPack := recognitionExample(t, "1.1.0")
	if _, err = c.db.Exec(`CREATE TRIGGER reject_recognition_update BEFORE UPDATE ON recognition_packages BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	_, revision, _ := c.recognitionPackages(ctx)
	if err = c.importRecognition(ctx, nextRaw, nextPack.Checksum, revision); err == nil {
		t.Fatal("failed update reported success")
	}
	_, after, _ := c.recognitionPackages(ctx)
	if after != revision || active.Err() != nil {
		t.Fatal("failed update changed registry/session")
	}
	if _, err = c.db.Exec(`DROP TRIGGER reject_recognition_update`); err != nil {
		t.Fatal(err)
	}
	installRecognitionExample(t, c, nextRaw, nextPack)
	if active.Err() == nil {
		t.Fatal("update did not cancel previous executable")
	}
	list, _, err := c.recognitionPackages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range list {
		if v.ID == id && (!v.Selected || !v.Enabled || v.PreviousVersion != "1.0.0") {
			t.Fatal(v)
		}
	}
	recognitionChange(t, c, id, "restore")
	s, _ := (registry.SQLStore{DB: c.db}).Load(ctx)
	e, _ := registryEntryFor(s, id)
	if e.Descriptor.Version != "1.0.0" || !e.Enabled {
		t.Fatal(e)
	}
	// Explicit choice persists across restart.
	reopened, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	assertTitle(reopened, "Exemple — Iron Man")
	recognitionChange(t, reopened, id, "remove")
	if _, _, _, err = reopened.openRecognizer(ctx, "films"); err == nil {
		t.Fatal("removed selection silently fell back")
	}
	if len(getItems(t, reopened)) != len(before) {
		t.Fatal("catalogue changed on removal")
	}
	var n int
	if err = reopened.db.QueryRow(`SELECT count(*) FROM recognition_packages WHERE id=?`, id).Scan(&n); err != nil || n != 0 {
		t.Fatal("archive/history retained", err)
	}
	recognitionChange(t, reopened, "films", "select")
	assertTitle(reopened, "Iron Man")
}

func TestExternalRecognitionFailuresAreAtomic(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	raw, p := recognitionExample(t, "1.0.0")
	_, revision, _ := c.recognitionPackages(ctx)
	if err := c.importRecognition(ctx, raw, "wrong", revision); err == nil {
		t.Fatal("wrong preview accepted")
	}
	recognitionChange(t, c, "music", "disable")
	if err := c.importRecognition(ctx, raw, p.Checksum, revision); !errors.Is(err, registry.ErrConflict) {
		t.Fatal("stale preview", err)
	}
	c.extensionSafeMode = true
	_, revision, _ = c.recognitionPackages(ctx)
	if err := c.importRecognition(ctx, raw, p.Checksum, revision); err == nil {
		t.Fatal("safe mode import")
	}
	c.extensionSafeMode = false
	if _, err := c.db.Exec(`CREATE TRIGGER reject_recognition_insert BEFORE INSERT ON recognition_packages BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if err := c.importRecognition(ctx, raw, p.Checksum, revision); err == nil {
		t.Fatal("SQL failure accepted")
	}
	_, after, _ := c.recognitionPackages(ctx)
	if after != revision {
		t.Fatal("registry changed on failed write")
	}
	if _, err := c.db.Exec(`DROP TRIGGER reject_recognition_insert`); err != nil {
		t.Fatal(err)
	}
	bad := p.Manifest
	bad.Module.ID = "mosaune.impersonation"
	if _, err := validateRecognitionImport(packageformat.Executable{Manifest: bad}); err == nil {
		t.Fatal("reserved identity accepted")
	}
	bad = p.Manifest
	bad.Module.Services = []protocol.ServiceRequirement{{Name: "tmdb.api.v1", Operations: []string{"get"}}}
	if _, err := validateRecognitionImport(packageformat.Executable{Manifest: bad}); err == nil {
		t.Fatal("unsupported service accepted")
	}
	bad = p.Manifest
	bad.Module.Dependencies = []string{"other.service.v1"}
	if _, err := validateRecognitionImport(packageformat.Executable{Manifest: bad}); err == nil {
		t.Fatal("unsupported dependency accepted")
	}
	// A syntactically valid Wasm header is not an executable recognizer.
	invalidRaw, err := packageformat.CreateExecutable(p.Manifest, []byte{0, 97, 115, 109, 1, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	invalidPack, _ := packageformat.ParseExecutable(invalidRaw, version)
	if err = c.importRecognition(ctx, invalidRaw, invalidPack.Checksum, revision); err == nil {
		t.Fatal("unusable code installed")
	}
	_, after, _ = c.recognitionPackages(ctx)
	if after != revision {
		t.Fatal("invalid code modified registry")
	}
}

func TestExternalRecognitionImportHTTPPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	raw, p := recognitionExample(t, "1.0.0")
	upload := func(a *accountClient, action string, revision string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/recognition-packages/"+action, bytes.NewReader(raw))
		r.Header.Set("X-Mosaune-Token", a.token)
		r.Header.Set("X-Mosaune-Package-Checksum", p.Checksum)
		r.Header.Set("X-Mosaune-Registry-Revision", revision)
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := upload(admin, "preview", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := upload(admin, "import", ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
	_, revision, _ := c.recognitionPackages(context.Background())
	if w := upload(admin, "import", strconv.FormatInt(revision, 10)); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	user := &accountClient{t: t, h: h}
	user.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, action := range []string{"preview", "import"} {
		if w := upload(user, action, strconv.FormatInt(revision, 10)); w.Code != 403 {
			t.Fatal("viewer package access", w.Code)
		}
	}
}

func TestRecognitionMigration18PreservesAbsentAndDisabled(t *testing.T) {
	c, dir := testCatalog(t)
	removeAudioTagsForLegacyFixture(t, c)
	recognitionChange(t, c, "films", "remove")
	recognitionChange(t, c, "series", "disable")
	_, err := c.db.Exec(`DROP TABLE recognition_selection; ALTER TABLE recognition_packages RENAME TO recognition_new;
CREATE TABLE recognition_packages(kind TEXT PRIMARY KEY,id TEXT NOT NULL UNIQUE,code BLOB NOT NULL);
INSERT INTO recognition_packages(kind,id,code) SELECT kind,id,code FROM recognition_new; DROP TABLE recognition_new; PRAGMA user_version=18;`)
	if err != nil {
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
		if !v.Selected || v.Kind == "films" && v.Installed || v.Kind == "series" && v.Enabled {
			t.Fatal(v)
		}
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal(backups)
	}
}
