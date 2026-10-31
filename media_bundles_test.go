package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/registry"
	"reflect"
	"testing"
)

func bundlePreview(t *testing.T, c *catalog, id, op string) mediaBundlePlan {
	t.Helper()
	p, e := c.mediaBundlePlans(context.Background(), id, op, nil, "", false)
	if e != nil || len(p) != 1 {
		t.Fatal(p, e)
	}
	return p[0]
}
func bundleApply(c *catalog, p mediaBundlePlan) error {
	_, e := c.mediaBundlePlans(context.Background(), p.ID, p.Operation, &p.Revision, p.Checksum, true)
	return e
}
func TestMediaBundlesScreenAndModulesLifecycle(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	files := t.TempDir()
	putFile(t, files, "Example.2020.mkv", "fixture")
	if _, e := c.scan(ctx, library{Path: files, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	before := getItems(t, c)
	p := bundlePreview(t, c, "films", "prepare")
	if p.State != "ready" || len(p.Steps) != 0 || p.InstalledVersion != "1.0.0" {
		t.Fatal(p)
	}
	active, done, _, _, e := c.beginRecognition(ctx, "films")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	p = bundlePreview(t, c, "films", "remove")
	if len(p.Steps) != 3 || len(p.Kept) != 3 {
		t.Fatal(p)
	}
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	if active.Err() == nil {
		t.Fatal("scan was not cancelled")
	}
	if !reflect.DeepEqual(before, getItems(t, c)) {
		t.Fatal("media changed")
	}
	p = bundlePreview(t, c, "films", "prepare")
	if p.State != "absent" || p.ScreenState != "absent" {
		t.Fatal(p)
	}
	if bundlePreview(t, c, "series", "prepare").State != "ready" {
		t.Fatal("series affected")
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = bundlePreview(t, c, "films", "prepare")
	if p.State != "absent" {
		t.Fatal("group resurrected")
	}
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	if bundlePreview(t, c, "films", "prepare").State != "ready" {
		t.Fatal("incomplete reinstall")
	}
	if !reflect.DeepEqual(before, getItems(t, c)) {
		t.Fatal("media lost on reinstall")
	}
	p = bundlePreview(t, c, "films", "disable")
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	p = bundlePreview(t, c, "films", "prepare")
	if p.State != "disabled" || p.ScreenState != "disabled" {
		t.Fatal(p)
	}
}
func TestMediaBundlesAtomicScreenFailureAndStalePreview(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	before, rev, _ := c.recognitionPackages(ctx)
	p := bundlePreview(t, c, "films", "remove")
	call, done, _, _, e := c.beginRecognition(ctx, "films")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	if _, e = c.db.Exec(`CREATE TEMP TRIGGER refuse_screen_remove BEFORE DELETE ON extensions BEGIN SELECT RAISE(ABORT,'fixture failure'); END;`); e != nil {
		t.Fatal(e)
	}
	if bundleApply(c, p) == nil {
		t.Fatal("failed screen write accepted")
	}
	after, next, _ := c.recognitionPackages(ctx)
	if rev != next || !reflect.DeepEqual(before, after) || call.Err() != nil || bundlePreview(t, c, "films", "prepare").State != "ready" {
		t.Fatal("partial write or premature cancellation")
	}
	if _, e = c.db.Exec("DROP TRIGGER refuse_screen_remove"); e != nil {
		t.Fatal(e)
	}
	// A screen edit does not advance the executable registry; its snapshot must still invalidate the preview.
	if _, e = c.db.Exec("UPDATE extensions SET enabled=0 WHERE id=?", p.Manifest.Screen); e != nil {
		t.Fatal(e)
	}
	if bundleApply(c, p) == nil {
		t.Fatal("screen change did not invalidate preview")
	}
	p = bundlePreview(t, c, "films", "remove")
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	p = bundlePreview(t, c, "films", "prepare")
	if _, e = c.db.Exec(`CREATE TEMP TRIGGER refuse_screen_install BEFORE INSERT ON extensions BEGIN SELECT RAISE(ABORT,'fixture failure'); END;`); e != nil {
		t.Fatal(e)
	}
	before, rev, _ = c.recognitionPackages(ctx)
	if bundleApply(c, p) == nil {
		t.Fatal("failed install accepted")
	}
	after, next, _ = c.recognitionPackages(ctx)
	if rev != next || !reflect.DeepEqual(before, after) {
		t.Fatal("partial install")
	}
}
func TestMediaBundlesRememberScreenReplacementAndKeepGrants(t *testing.T) {
	c, _, a := accountFixture(t)
	ctx := context.Background()
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	p := bundlePreview(t, c, "films", "disable")
	if e := bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	replacement := screenExamplePackage()
	raw := screenArchive(replacement)
	pack, e := parseExtensionPackage(raw)
	if e != nil {
		t.Fatal(e)
	}
	if r := extensionUpload(a, "install", raw, pack.Checksum, ""); r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	state := func(on bool) {
		t.Helper()
		if r := a.request("PUT", "/api/v1/extensions/"+pack.Manifest.ID+"/state", map[string]any{"enabled": on, "checksum": pack.Checksum}); r.Code != 204 {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	state(true)
	state(false)
	p = bundlePreview(t, c, "films", "prepare")
	if p.ScreenID != pack.Manifest.ID {
		t.Fatal("replacement forgotten", p.ScreenID)
	}
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	var grants int
	if e = c.db.QueryRow("SELECT count(*) FROM extension_grants WHERE extension_id=?", pack.Manifest.ID).Scan(&grants); e != nil || grants != 0 {
		t.Fatal("external screen granted automatically", e, grants)
	}
	if r := a.request("DELETE", "/api/v1/extensions/"+pack.Manifest.ID, map[string]string{"checksum": pack.Checksum}); r.Code != 204 {
		t.Fatal(r.Code)
	}
	p = bundlePreview(t, c, "films", "prepare")
	if len(p.Blockers) == 0 || bundleApply(c, p) == nil {
		t.Fatal("missing replacement silently substituted")
	}
	supplied, e := c.extension(ctx, p.Manifest.Screen)
	if e != nil {
		t.Fatal(e)
	}
	if r := a.request("PUT", "/api/v1/extensions/"+supplied.Manifest.ID+"/state", map[string]any{"enabled": true, "checksum": supplied.Checksum}); r.Code != 204 {
		t.Fatal(r.Code, r.Body.String())
	}
	if len(bundlePreview(t, c, "films", "prepare").Blockers) != 0 {
		t.Fatal("explicit supplied choice not honored")
	}
}
func TestMediaBundlesMigrationPreservesPartialState(t *testing.T) {
	c, dir := testCatalog(t)
	recognitionChange(t, c, "music", "disable")
	if _, e := c.db.Exec("DELETE FROM extensions WHERE id='org.mosaune.screens.music'; DROP TABLE media_bundles; DROP TABLE media_bundle_screens; PRAGMA user_version=29"); e != nil {
		t.Fatal(e)
	}
	c.close()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p := bundlePreview(t, c, "music", "prepare")
	if p.State != "partial" || p.ScreenState != "absent" {
		t.Fatal("migration changed existing state", p)
	}
	for _, step := range p.Steps {
		if step.Slot == "music" && step.Operation == "enable" {
			return
		}
	}
	t.Fatal("disabled recognizer re-enabled")
}
func TestMediaBundlesPermissionsSafeModeAndConflicts(t *testing.T) {
	c, h, a := accountFixture(t)
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	p := bundlePreview(t, c, "films", "disable")
	in := map[string]any{"operation": "disable", "revision": p.Revision, "checksum": p.Checksum}
	if r := a.request("POST", "/api/v1/media-bundles/films/apply", in); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if e := bundleApply(c, p); !errors.Is(e, registry.ErrConflict) {
		t.Fatal("stale revision", e)
	}
	a.create("Reader", accountTestPassword, nil)
	g := &accountClient{t: t, h: h}
	g.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	for _, route := range []string{"/api/v1/media-bundles", "/api/v1/media-bundles/films/preview", "/api/v1/media-bundles/films/apply"} {
		method := "POST"
		if route == "/api/v1/media-bundles" {
			method = "GET"
		}
		if r := g.request(method, route, in); r.Code != 403 {
			t.Fatal("non-admin", r.Code)
		}
	}
	c.extensionSafeMode = true
	p = bundlePreview(t, c, "films", "prepare")
	if len(p.Blockers) == 0 || bundleApply(c, p) == nil {
		t.Fatal("safe mode bypass")
	}
	// The endpoint returns the same manifest used by the planner, rather than a second UI list.
	r := a.request("GET", "/api/v1/media-bundles", nil)
	var out struct {
		Bundles []mediaBundlePlan `json:"bundles"`
	}
	if json.Unmarshal(r.Body.Bytes(), &out) != nil || len(out.Bundles) != 3 {
		t.Fatal(r.Code, r.Body.String())
	}
}

func TestMediaBundlesMultipleScreenClaimsAndGrantChanges(t *testing.T) {
	c, _, a := accountFixture(t)
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	a.create("Reader", accountTestPassword, nil)
	p := bundlePreview(t, c, "films", "remove")
	if err := bundleApply(c, p); err != nil {
		t.Fatal(err)
	}
	replacement := screenExamplePackage()
	for _, screen := range screenPackages() {
		if screen.Manifest.ID == "org.mosaune.screens.music" {
			replacement.Manifest.Pages = append(replacement.Manifest.Pages, screen.Manifest.Pages...)
		}
	}
	raw, _ := json.Marshal(replacement.Manifest)
	if _, err := c.db.Exec("INSERT INTO extensions VALUES(?,?,?,0)", replacement.Manifest.ID, string(raw), digest(string(raw))); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("UPDATE media_bundle_screens SET id=? WHERE kind='films'", replacement.Manifest.ID); err != nil {
		t.Fatal(err)
	}
	p = bundlePreview(t, c, "films", "prepare")
	if len(p.Blockers) == 0 || bundleApply(c, p) == nil {
		t.Fatal("multiple-page replacement bypassed music ownership", p)
	}
	// Disable the other owner: preparing can now enable the chosen replacement.
	if _, err := c.db.Exec("UPDATE extensions SET enabled=0 WHERE id='org.mosaune.screens.music'"); err != nil {
		t.Fatal(err)
	}
	p = bundlePreview(t, c, "films", "prepare")
	if len(p.Blockers) != 0 {
		t.Fatal(p)
	}
	// Account grants are part of the preview even when the registry revision is unchanged.
	result, err := c.db.Exec("DELETE FROM extension_grants WHERE extension_id='org.mosaune.screens.music'")
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		t.Fatal("fixture has no grant to revoke")
	}
	if bundleApply(c, p) == nil {
		t.Fatal("grant changes accepted with stale preview")
	}
	p = bundlePreview(t, c, "films", "prepare")
	if err := bundleApply(c, p); err != nil {
		t.Fatal(err)
	}
	p = bundlePreview(t, c, "films", "remove")
	if len(p.Blockers) == 0 || bundleApply(c, p) == nil {
		t.Fatal("shared screen removed with only one family")
	}
}
