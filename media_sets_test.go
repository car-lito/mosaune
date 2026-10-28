package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/registry"
	"reflect"
	"testing"
)

func setPreview(t *testing.T, c *catalog, id, op string) mediaSetPlan {
	t.Helper()
	p, e := c.mediaSetPlans(context.Background(), id, op, nil, "", false)
	if e != nil || len(p) != 1 {
		t.Fatal(p, e)
	}
	return p[0]
}
func setApply(c *catalog, p mediaSetPlan) error {
	_, e := c.mediaSetPlans(context.Background(), p.ID, p.Operation, &p.Revision, p.Checksum, true)
	return e
}
func TestMediaSetsLifecycleAndPreservedData(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	files := t.TempDir()
	putFile(t, files, "Example.2020.mkv", "fixture")
	if _, e := c.scan(ctx, library{Path: files, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	before := getItems(t, c)
	p := setPreview(t, c, "films", "prepare")
	if len(p.Steps) != 0 || len(p.Kept) != 5 {
		t.Fatal(p)
	}
	active, done, _, _, e := c.beginRecognition(ctx, "films")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	shared, sharedDone, _, _, e := c.beginRecognition(ctx, "detail-header")
	if e != nil {
		t.Fatal(e)
	}
	defer sharedDone()
	p = setPreview(t, c, "films", "remove")
	if len(p.Steps) != 2 || len(p.Kept) != 3 {
		t.Fatal(p)
	}
	bad := p
	bad.Checksum = "forged"
	if setApply(c, bad) == nil {
		t.Fatal("forged preview accepted")
	}
	if e = setApply(c, p); e != nil {
		t.Fatal(e)
	}
	if active.Err() == nil || shared.Err() != nil {
		t.Fatal("incorrect cancellation")
	}
	if !reflect.DeepEqual(before, getItems(t, c)) {
		t.Fatal("media data changed")
	}
	if e = setApply(c, p); !errors.Is(e, registry.ErrConflict) {
		t.Fatal("stale preview accepted", e)
	}
	p = setPreview(t, c, "films", "prepare")
	if len(p.Steps) != 4 {
		t.Fatal(p)
	}
	if e = setApply(c, p); e != nil {
		t.Fatal(e)
	}
	if unlock, e := c.recognitionCommit(active); e == nil {
		unlock()
		t.Fatal("old result revived")
	}
	p = setPreview(t, c, "films", "prepare")
	if len(p.Steps) != 0 {
		t.Fatal(p)
	}
	p = setPreview(t, c, "music", "disable")
	if len(p.Steps) != 3 || len(p.Kept) != 3 {
		t.Fatal(p)
	}
	if e = setApply(c, p); e != nil {
		t.Fatal(e)
	}
	reopened, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.close()
	p = setPreview(t, reopened, "music", "prepare")
	if len(p.Steps) != 3 {
		t.Fatal("disabled state lost", p)
	}
	if len(setPreview(t, reopened, "series", "prepare").Steps) != 0 {
		t.Fatal("series affected")
	}
	if !reflect.DeepEqual(before, getItems(t, reopened)) {
		t.Fatal("collection lost")
	}
}
func TestMediaSetsAtomicFailureAndStalePreview(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	before, rev, e := c.recognitionPackages(ctx)
	if e != nil {
		t.Fatal(e)
	}
	call, done, _, _, e := c.beginRecognition(ctx, "films")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	// Fail after the first module was removed inside the transaction.
	if _, e = c.db.Exec(`CREATE TRIGGER refuse_set_remove BEFORE DELETE ON recognition_packages WHEN OLD.kind='catalogue-films' BEGIN SELECT RAISE(ABORT,'fixture failure'); END;`); e != nil {
		t.Fatal(e)
	}
	p := setPreview(t, c, "films", "remove")
	if setApply(c, p) == nil {
		t.Fatal("injected failure accepted")
	}
	after, next, e := c.recognitionPackages(ctx)
	if e != nil || rev != next || !reflect.DeepEqual(before, after) || call.Err() != nil {
		t.Fatal("partial operation or premature cancellation", e)
	}
	if _, e = c.db.Exec("DROP TRIGGER refuse_set_remove"); e != nil {
		t.Fatal(e)
	}
	if e = setApply(c, p); e != nil {
		t.Fatal(e)
	}
	p = setPreview(t, c, "films", "prepare")
	if _, e = c.db.Exec(`CREATE TRIGGER refuse_set_install BEFORE INSERT ON recognition_packages WHEN NEW.kind='catalogue-films' BEGIN SELECT RAISE(ABORT,'fixture failure'); END;`); e != nil {
		t.Fatal(e)
	}
	before, rev, _ = c.recognitionPackages(ctx)
	if setApply(c, p) == nil {
		t.Fatal("partial install accepted")
	}
	after, next, _ = c.recognitionPackages(ctx)
	if rev != next || !reflect.DeepEqual(before, after) {
		t.Fatal("install not atomic")
	}
	if _, e = c.db.Exec("DROP TRIGGER refuse_set_install"); e != nil {
		t.Fatal(e)
	}
	recognitionChange(t, c, "series", "disable")
	if !errors.Is(setApply(c, p), registry.ErrConflict) {
		t.Fatal("concurrent state change ignored")
	}
}
func TestMediaSetsExternalChoicesAndSafeMode(t *testing.T) {
	c, _ := testCatalog(t)
	raw, pack := experienceExample(t, "1.0.0")
	installRecognitionExample(t, c, raw, pack)
	id := pack.Manifest.Module.ID
	recognitionChange(t, c, id, "enable")
	recognitionChange(t, c, id, "select")
	recognitionChange(t, c, id, "disable")
	p := setPreview(t, c, "music", "prepare")
	if len(p.Steps) != 1 || p.Steps[0].ID != id || p.Steps[0].Operation != "enable" || !p.Steps[0].Shared {
		t.Fatal("replacement lost", p)
	}
	if e := setApply(c, p); e != nil {
		t.Fatal(e)
	}
	p = setPreview(t, c, "music", "remove")
	if e := setApply(c, p); e != nil {
		t.Fatal(e)
	}
	items, _, _ := c.recognitionPackages(context.Background())
	found := false
	for _, v := range items {
		if v.ID == id {
			found = v.Enabled && v.Selected
		}
	}
	if !found {
		t.Fatal("shared replacement removed")
	}
	recognitionChange(t, c, id, "remove")
	p = setPreview(t, c, "films", "prepare")
	if len(p.Blockers) == 0 || setApply(c, p) == nil {
		t.Fatal("missing alternative silently replaced")
	}
	recognitionChange(t, c, "detail-experience", "select")
	c.extensionSafeMode = true
	p = setPreview(t, c, "music", "prepare")
	if len(p.Blockers) == 0 || setApply(c, p) == nil {
		t.Fatal("safe mode bypassed")
	}
	p = setPreview(t, c, "films", "disable")
	if e := setApply(c, p); e != nil {
		t.Fatal(e)
	}
}
func TestMediaSetsHTTPPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	if w := admin.request("GET", "/api/v1/media-sets", nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("POST", "/api/v1/media-sets/films/preview", map[string]any{"operation": "remove"}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	p := setPreview(t, c, "films", "remove")
	in := map[string]any{"operation": p.Operation, "revision": p.Revision}
	w := admin.request("POST", "/api/v1/media-sets/films/preview", in)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e := json.Unmarshal(w.Body.Bytes(), &p); e != nil {
		t.Fatal(e)
	}
	in["checksum"] = p.Checksum
	if w = admin.request("POST", "/api/v1/media-sets/films/apply", in); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = admin.request("POST", "/api/v1/media-sets/films/apply", in); w.Code != 409 {
		t.Fatal(w.Code)
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	user := &accountClient{t: t, h: h}
	user.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, path := range []string{"/api/v1/media-sets", "/api/v1/media-sets/films/preview", "/api/v1/media-sets/films/apply"} {
		method := "POST"
		if path == "/api/v1/media-sets" {
			method = "GET"
		}
		if w = user.request(method, path, in); w.Code != 403 {
			t.Fatal("viewer accessed sets", path, w.Code)
		}
	}
}
