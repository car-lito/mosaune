package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/registry"
	"path/filepath"
	"reflect"
	"testing"
)

func recognitionChange(t *testing.T, c *catalog, kind, op string) {
	t.Helper()
	_, revision, err := c.recognitionPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(context.Background(), kind, op, revision); err != nil {
		t.Fatal(err)
	}
}

func TestRecognitionLifecyclePersistenceAndCatalogue(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	files := t.TempDir()
	putFile(t, files, "Iron.Man.2008.mkv", "fixture")
	l, err := c.scan(ctx, library{Path: files, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	before := getItems(t, c)
	call, reader, done, err := c.openRecognizer(ctx, "films")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	music, musicReader, musicDone, err := c.openRecognizer(ctx, "music")
	if err != nil {
		t.Fatal(err)
	}
	defer musicDone()
	recognitionChange(t, c, "films", "disable")
	if !errors.Is(context.Cause(call), errRecognitionUnavailable) {
		t.Fatal("running session not cancelled")
	}
	if _, err = reader.recognize(call, []string{"Iron.Man.2008.mkv"}); err == nil {
		t.Fatal("disabled session executed")
	}
	if _, err = musicReader.recognize(music, []string{"Song.mp3"}); err != nil {
		t.Fatal("unrelated module affected", err)
	}
	if _, err = c.scan(ctx, l, false); err == nil {
		t.Fatal("disabled scan accepted")
	}
	if !reflect.DeepEqual(before, getItems(t, c)) {
		t.Fatal("disabled scan changed catalogue")
	}
	recognitionChange(t, c, "films", "enable")
	if unlock, err := c.recognitionCommit(call); err == nil {
		unlock()
		t.Fatal("old results accepted after reactivation")
	}
	recognitionChange(t, c, "films", "remove")
	var count int
	if err = c.db.QueryRow(`SELECT count(*) FROM recognition_packages WHERE kind='films'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("private executable retained", count, err)
	}
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	if _, err = next.scan(ctx, l, false); err == nil {
		t.Fatal("restart silently reinstalled module")
	}
	if !reflect.DeepEqual(before, getItems(t, next)) {
		t.Fatal("removal changed catalogue")
	}
	recognitionChange(t, next, "films", "install")
	if _, err = next.scan(ctx, l, false); err == nil {
		t.Fatal("installation silently activated module")
	}
	recognitionChange(t, next, "films", "enable")
	if _, err = next.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, getItems(t, next)) {
		t.Fatal("reinstall lost identifiers/metadata/progress")
	}
}

func TestRecognitionPackageRollbackStaleStateAndCorruption(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	call, _, done, err := c.openRecognizer(ctx, "films")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	_, revision, _ := c.recognitionPackages(ctx)
	recognitionChange(t, c, "music", "disable")
	if err = c.changeRecognition(ctx, "films", "remove", revision); !errors.Is(err, registry.ErrConflict) {
		t.Fatal("stale state accepted", err)
	}
	if call.Err() != nil {
		t.Fatal("stale mutation cancelled scan")
	}
	if _, err = c.db.Exec(`CREATE TRIGGER refuse_recognition_delete BEFORE DELETE ON recognition_packages BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	_, revision, _ = c.recognitionPackages(ctx)
	if err = c.changeRecognition(ctx, "films", "remove", revision); err == nil {
		t.Fatal("failed delete reported success")
	}
	_, after, _ := c.recognitionPackages(ctx)
	if after != revision || call.Err() != nil {
		t.Fatal("failed deletion changed registry/session")
	}
	if _, err = c.db.Exec(`DROP TRIGGER refuse_recognition_delete; UPDATE recognition_packages SET code=x'00' WHERE kind='films'`); err != nil {
		t.Fatal(err)
	}
	files := t.TempDir()
	putFile(t, files, "Film.mkv", "fixture")
	if _, err = c.scan(testContext(ctx), library{Kind: "films", Path: files}, true); err == nil {
		t.Fatal("corrupt package fell back to embedded code")
	}
}

func TestRecognitionSafeModeAndNoInstalledModules(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	c.extensionSafeMode = true
	if _, _, _, err := c.openRecognizer(ctx, "films"); err == nil {
		t.Fatal("safe mode executed module")
	}
	_, revision, _ := c.recognitionPackages(ctx)
	if err := c.changeRecognition(ctx, "films", "enable", revision); err == nil {
		t.Fatal("safe mode enabled module")
	}
	c.extensionSafeMode = false
	for _, kind := range recognitionKinds {
		recognitionChange(t, c, kind, "remove")
	}
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	list, _, err := next.recognitionPackages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if p.Installed || p.Enabled {
			t.Fatal("empty installation not preserved", p)
		}
	}
}

// Build a genuine prior-schema fixture, not a new database with only its
// version number changed. Old migrations must not see tables from the future.
func removeRecognitionForLegacyFixture(t *testing.T, c *catalog) {
	t.Helper()
	for _, kind := range recognitionKinds {
		recognitionChange(t, c, kind, "remove")
	}
	if _, err := c.db.Exec(`DROP TABLE recognition_packages`); err != nil {
		t.Fatal(err)
	}
}

func TestRecognitionMigrationSeedsOnceWithBackup(t *testing.T) {
	c, dir := testCatalog(t)
	removeRecognitionForLegacyFixture(t, c)
	if _, err := c.db.Exec(`PRAGMA user_version=17`); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	list, _, err := next.recognitionPackages(context.Background())
	if err != nil || len(list) != 3 {
		t.Fatal(list, err)
	}
	for _, p := range list {
		if !p.Installed || !p.Enabled {
			t.Fatal(p)
		}
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("migration backup", backups)
	}
}

func TestRecognitionPackageHTTPPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	path := "/api/v1/recognition-packages"
	if w := admin.request("GET", path, nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("POST", path+"/films", map[string]any{"operation": "disable"}); w.Code != 400 {
		t.Fatal("missing revision accepted", w.Code)
	}
	_, revision, _ := c.recognitionPackages(context.Background())
	if w := admin.request("POST", path+"/films", map[string]any{"operation": "disable", "revision": revision}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("POST", path+"/films", map[string]any{"operation": "enable", "revision": revision}); w.Code != 409 {
		t.Fatal("stale request accepted", w.Code)
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	user := &accountClient{t: t, h: h}
	user.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "POST"} {
		p := path
		if method == "POST" {
			p += "/music"
		}
		if w := user.request(method, p, map[string]any{"operation": "disable", "revision": revision}); w.Code != 403 {
			t.Fatal("viewer changed/accessed packages", w.Code)
		}
	}
}
