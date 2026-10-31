package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func testBundleArchive(t *testing.T, kind, v string) bundleArchive {
	t.Helper()
	b, e := exampleBundleArchive(kind, v)
	if e != nil {
		t.Fatal(e)
	}
	p, e := parseBundleArchive(b)
	if e != nil {
		t.Fatal(kind, e)
	}
	return p
}
func importTestBundle(t *testing.T, c *catalog, p bundleArchive) bundleArchivePlan {
	t.Helper()
	plan, e := c.processBundleArchive(context.Background(), p, "", false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.processBundleArchive(context.Background(), p, plan.Checksum, true); e != nil {
		t.Fatal(e)
	}
	return plan
}
func TestBundleArchivesRoundtripAndUpdate(t *testing.T) {
	for _, kind := range recognitionKinds {
		t.Run(kind, func(t *testing.T) {
			c, dir := testCatalog(t)
			ctx := context.Background()
			p := testBundleArchive(t, kind, "1.0.0")
			plan := importTestBundle(t, c, p)
			for _, step := range plan.Steps {
				if step.Shared && step.Operation != "keep" {
					t.Fatal("shared factory duplicated", step)
				}
			}
			state := bundlePreview(t, c, kind, "prepare")
			if state.Archive == nil || state.ScreenID != p.Screen.Manifest.ID || state.ScreenState != "disabled" {
				t.Fatal(state)
			}
			if e := bundleApply(c, state); e != nil {
				t.Fatal(e)
			}
			if bundlePreview(t, c, kind, "prepare").State != "ready" {
				t.Fatal("activation incomplete")
			}
			c.close()
			var e error
			c, e = openCatalog(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer c.close()
			update := testBundleArchive(t, kind, "1.1.0")
			plan = importTestBundle(t, c, update)
			if plan.PreviousVersion != "1.0.0" || bundlePreview(t, c, kind, "prepare").State != "ready" {
				t.Fatal("update lost active state")
			}
			var old string
			if e = c.db.QueryRowContext(ctx, "SELECT previous_version FROM recognition_packages WHERE id=?", p.Modules[kind].Manifest.Module.ID).Scan(&old); e != nil || old != "1.0.0" {
				t.Fatal("component rollback missing", old, e)
			}
			if _, e = c.processBundleArchive(ctx, p, "", false); e == nil {
				t.Fatal("downgrade accepted")
			}
			if e = bundleApply(c, bundlePreview(t, c, kind, "remove")); e != nil {
				t.Fatal(e)
			}
			if bundlePreview(t, c, kind, "prepare").Archive != nil {
				t.Fatal("archive record survived removal")
			}
			importTestBundle(t, c, update)
			if e = bundleApply(c, bundlePreview(t, c, kind, "prepare")); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestBundleArchivesAtomicFailureAndStalePlan(t *testing.T) {
	c, _, a := accountFixture(t)
	ctx := context.Background()
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	a.create("Reader", accountTestPassword, nil)
	p := testBundleArchive(t, "films", "1.0.0")
	plan, e := c.processBundleArchive(ctx, p, "", false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec("DELETE FROM extension_grants WHERE extension_id='org.mosaune.screens.films'"); e != nil {
		t.Fatal(e)
	}
	if _, e = c.processBundleArchive(ctx, p, plan.Checksum, true); e == nil {
		t.Fatal("stale access accepted")
	}
	plan, e = c.processBundleArchive(ctx, p, "", false)
	if e != nil {
		t.Fatal(e)
	}
	before, rev, _ := c.recognitionPackages(ctx)
	call, done, _, _, e := c.beginRecognition(testContext(ctx), "films")
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	if _, e = c.db.Exec(`CREATE TEMP TRIGGER fail_family_screen BEFORE INSERT ON extensions BEGIN SELECT RAISE(ABORT,'fixture'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e = c.processBundleArchive(ctx, p, plan.Checksum, true); e == nil {
		t.Fatal("failed write accepted")
	}
	after, next, _ := c.recognitionPackages(ctx)
	if rev != next || !reflect.DeepEqual(before, after) || call.Err() != nil {
		t.Fatal("partial commit or premature cancellation")
	}
	if bundlePreview(t, c, "films", "prepare").State != "ready" {
		t.Fatal("screen altered despite rollback")
	}
	if _, e = c.db.Exec("DROP TRIGGER fail_family_screen"); e != nil {
		t.Fatal(e)
	}
	importTestBundle(t, c, p)
	if call.Err() == nil {
		t.Fatal("old selection still running")
	}
	c.extensionSafeMode = true
	if _, e = c.processBundleArchive(ctx, p, "", false); e == nil {
		t.Fatal("safe mode bypass")
	}
}
func TestBundleArchivesKeepAlternativesAndPausedState(t *testing.T) {
	c, _, a := accountFixture(t)
	ctx := context.Background()
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	p := testBundleArchive(t, "films", "1.0.0")
	importTestBundle(t, c, p)
	if e := bundleApply(c, bundlePreview(t, c, "films", "prepare")); e != nil {
		t.Fatal(e)
	}
	// Explicitly choose the original recognizer as an alternative to the imported one.
	// A supplied default may be adopted by a family, so use a distinct external recognizer instead.
	replacement := p.Modules["films"]
	replacement.Manifest.Module.ID = "org.example.alternative.films"
	raw, e := packageformat.CreateExecutable(replacement.Manifest, replacement.Code)
	if e != nil {
		t.Fatal(e)
	}
	replacement, e = packageformat.ParseExecutable(raw, version)
	if e != nil {
		t.Fatal(e)
	}
	_, rev, _ := c.recognitionPackages(ctx)
	if e = c.importRecognition(ctx, raw, replacement.Checksum, rev); e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"enable", "select"} {
		_, rev, _ = c.recognitionPackages(ctx)
		if e = c.changeRecognition(ctx, replacement.Manifest.Module.ID, op, rev); e != nil {
			t.Fatal(e)
		}
	}
	// Pause the screen without forgetting it; import must preserve the pause and grants.
	screen, e := c.extension(ctx, p.Screen.Manifest.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.setUIScreenState(ctx, screen, false); e != nil {
		t.Fatal(e)
	}
	update := testBundleArchive(t, "films", "1.1.0")
	plan := importTestBundle(t, c, update)
	for _, step := range plan.Steps {
		if step.Slot == "films" && step.Operation != "keep" {
			t.Fatal("replacement overwritten")
		}
	}
	state := bundlePreview(t, c, "films", "prepare")
	if state.ScreenState != "disabled" {
		t.Fatal("paused screen reactivated")
	}
	var grants int
	c.db.QueryRow("SELECT count(*) FROM extension_grants WHERE extension_id=?", p.Screen.Manifest.ID).Scan(&grants)
	if grants != 0 {
		t.Fatal("new external screen auto-granted")
	}
	// Even a removed alternative remains an explicit choice.
	_, rev, _ = c.recognitionPackages(ctx)
	if e = c.changeRecognition(ctx, replacement.Manifest.Module.ID, "remove", rev); e != nil {
		t.Fatal(e)
	}
	plan, e = c.processBundleArchive(ctx, update, "", false)
	if e != nil {
		t.Fatal(e)
	}
	for _, step := range plan.Steps {
		if step.Slot == "films" && step.Operation != "keep" {
			t.Fatal("missing alternative substituted")
		}
	}
}
func mutateBundleArchive(t *testing.T, raw []byte, f func(map[string][]byte)) []byte {
	t.Helper()
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal(e)
	}
	files := map[string][]byte{}
	for _, entry := range z.File {
		r, _ := entry.Open()
		files[entry.Name], _ = io.ReadAll(r)
		r.Close()
	}
	f(files)
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for name, b := range files {
		entry, _ := w.Create(name)
		entry.Write(b)
	}
	w.Close()
	return out.Bytes()
}
func TestBundleArchivesRejectMalformedAndChangedIdentity(t *testing.T) {
	raw, _ := exampleBundleArchive("films", "1.0.0")
	cases := []func(map[string][]byte){func(f map[string][]byte) { f["../escape"] = []byte("x") }, func(f map[string][]byte) { delete(f, "screen.zip") }, func(f map[string][]byte) { f["films.zip"][0] ^= 1 }, func(f map[string][]byte) { f["family.json"] = append(f["family.json"], []byte(` {}`)...) }}
	for i, change := range cases {
		if _, e := parseBundleArchive(mutateBundleArchive(t, raw, change)); e == nil {
			t.Fatal("accepted malformed archive", i)
		}
	}
	c, _ := testCatalog(t)
	p := testBundleArchive(t, "films", "1.0.0")
	importTestBundle(t, c, p)
	next := testBundleArchive(t, "films", "1.1.0")
	program := next.Modules["films"]
	program.Manifest.Module.ID = "org.example.changed.identity"
	next.Modules["films"] = program
	if _, e := c.processBundleArchive(context.Background(), next, "", false); e == nil {
		t.Fatal("identity changed")
	}
}
func TestBundleArchivesAdminHTTP(t *testing.T) {
	c, h, a := accountFixture(t)
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	raw, _ := exampleBundleArchive("films", "1.0.0")
	post := func(client *accountClient, action, checksum string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/media-bundle-archives/"+action, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/zip")
		req.Header.Set("X-Mosaune-Token", client.token)
		req.Header.Set("X-Mosaune-Plan-Checksum", checksum)
		req.AddCookie(client.cookie)
		r := httptest.NewRecorder()
		h.ServeHTTP(r, req)
		return r
	}
	r := post(a, "preview", "")
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	var plan bundleArchivePlan
	if e := json.Unmarshal(r.Body.Bytes(), &plan); e != nil {
		t.Fatal(e)
	}
	if r = post(a, "import", ""); r.Code != 409 {
		t.Fatal("missing preview", r.Code)
	}
	if r = post(a, "import", plan.Checksum); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	a.create("Viewer", accountTestPassword, nil)
	v := &accountClient{t: t, h: h}
	v.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, action := range []string{"preview", "import"} {
		if r = post(v, action, plan.Checksum); r.Code != 403 {
			t.Fatal("non-admin import", r.Code)
		}
	}
	r = a.request("GET", "/api/v1/media-bundle-archives/example/music", nil)
	if r.Code != 200 || !strings.Contains(r.Header().Get("Content-Type"), "zip") {
		t.Fatal(r.Code)
	}
	if _, e := parseBundleArchive(r.Body.Bytes()); e != nil {
		t.Fatal(e)
	}
}

func TestBundleArchivesExplicitReturnToDefaults(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	p := testBundleArchive(t, "films", "1.0.0")
	importTestBundle(t, c, p)
	_, rev, _ := c.recognitionPackages(ctx)
	if e := c.changeRecognition(ctx, "films", "select", rev); e != nil {
		t.Fatal(e)
	}
	screen, e := c.extension(ctx, "org.mosaune.screens.films")
	if e != nil {
		t.Fatal(e)
	}
	if e = c.setUIScreenState(ctx, screen, true); e != nil {
		t.Fatal(e)
	}
	next := testBundleArchive(t, "films", "1.1.0")
	plan := importTestBundle(t, c, next)
	for _, step := range plan.Steps {
		if (step.Slot == "films" || step.Slot == "screen") && step.Operation != "keep" {
			t.Fatal("explicit return to defaults overwritten", step)
		}
	}
}
