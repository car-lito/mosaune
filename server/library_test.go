package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, identityKey{}, &identity{account: account{ID: "test-user", Name: "TestAdmin", Admin: true}})
}
func testCatalog(t *testing.T) (*catalog, string) {
	t.Helper()
	dir := t.TempDir()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e := c.createFirstAccount(context.Background(), account{ID: "test-user", Name: "TestAdmin", Admin: true, Theme: "light"}, "unused-test-hash"); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { c.close() })
	return c, dir
}
func putFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func getItems(t *testing.T, c *catalog) []media {
	t.Helper()
	items, err := c.items(testContext(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return items
}
func call(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, "http://127.0.0.1"+path, bytes.NewReader(data))
	r.Header.Set("X-Mosaune-Token", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func handlerFor(t *testing.T, c *catalog, modules string) (http.Handler, string) {
	t.Helper()
	mods, _ := enabledModules(modules)
	rawHandler := newHandler(fstest.MapFS{"index.html": {Data: []byte("ui")}}, mods, c)
	a, e := c.account(context.Background(), "test-user")
	if e != nil {
		t.Fatal(e)
	}
	session := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1", nil)
	if e = c.issueSession(session, req, a); e != nil {
		t.Fatal(e)
	}
	cookie := session.Result().Cookies()[0]
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.AddCookie(cookie); rawHandler.ServeHTTP(w, r) })
	w := call(t, h, "GET", "/api/v1/status", "", nil)
	var status struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	return h, status.Token
}

func TestPersistentLibraryAndProgress(t *testing.T) {
	c, dbDir := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "film.mp4", "0123456789")
	putFile(t, dir, "notes.txt", "ignored")
	ctx := testContext(context.Background())
	l, err := c.scan(ctx, library{Path: dir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	if len(items) != 1 || l.Count != 1 {
		t.Fatalf("items: %+v", items)
	}
	m := items[0]
	h, token := handlerFor(t, c, "video,music")
	w := call(t, h, "PUT", "/api/v1/media/"+m.ID+"/progress", token, map[string]any{"position": 42, "duration": 100, "completed": false, "revision": m.Revision})
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	// The catalog and progress survive closing all database handles and reopening.
	if err = c.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openCatalog(dbDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	c = reopened
	items = getItems(t, c)
	if items[0].Position != 42 || items[0].ID != m.ID {
		t.Fatal(items)
	}
	putFile(t, dir, "second.mp4", "new")
	if _, err = c.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	if len(getItems(t, c)) != 2 {
		t.Fatal("addition missing")
	}
	if err = os.Remove(filepath.Join(dir, "film.mp4")); err != nil {
		t.Fatal(err)
	}
	if _, err = c.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	if len(getItems(t, c)) != 1 {
		t.Fatal("missing file remained available")
	}
	// Replaced content must not inherit the old resume position.
	putFile(t, dir, "film.mp4", "a replacement movie with different size")
	if _, err = c.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	for _, item := range getItems(t, c) {
		if item.ID == m.ID && (item.Position != 0 || item.Revision == m.Revision) {
			t.Fatal("stale progress survived replacement")
		}
	}
	offline := l
	offline.Path = filepath.Join(dir, "disconnected")
	if _, err = c.scan(ctx, offline, false); err == nil {
		t.Fatal("offline scan accepted")
	}
	if len(getItems(t, c)) != 2 {
		t.Fatal("offline scan lost catalog")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = c.scan(cancelled, l, false); err == nil {
		t.Fatal("cancelled scan accepted")
	}
	original, err := os.ReadFile(filepath.Join(dir, "second.mp4"))
	if err != nil || string(original) != "new" {
		t.Fatal("media modified")
	}
}

func TestMediaRangeAndAccessControls(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "music.mp3", "0123456789")
	_, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m := getItems(t, c)[0]
	h, token := handlerFor(t, c, "music")
	r := httptest.NewRequest("GET", "http://127.0.0.1"+m.URL, nil)
	r.Header.Set("Range", "bytes=2-5")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 206 || w.Body.String() != "2345" || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatal(w.Code, w.Body.String(), w.Header())
	}
	for _, tc := range []struct{ host, origin, site, token string }{
		{"attacker.example", "", "", ""}, {"127.0.0.1", "https://attacker.example", "", ""}, {"127.0.0.1", "", "cross-site", ""},
	} {
		r := httptest.NewRequest("GET", "http://"+tc.host+"/api/v1/library", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.site)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("unsafe request accepted", tc)
		}
	}
	if w = call(t, h, "POST", "/api/v1/libraries", "", map[string]string{"path": dir, "kind": "music"}); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call(t, h, "POST", "/api/v1/libraries", token, map[string]string{"path": dir, "kind": "films"}); w.Code != 400 {
		t.Fatal("disabled module accepted")
	}
	if w = call(t, h, "GET", "/api/v1/media/unknown/stream", "", nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if _, err = c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true); err == nil {
		t.Fatal("overlapping root accepted")
	}
	disabled, _ := handlerFor(t, c, "")
	if w = call(t, disabled, "GET", m.URL, "", nil); w.Code != 404 {
		t.Fatal("disabled module streamed")
	}
}

func TestProgressValidation(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "film.mp4", "test")
	_, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m := getItems(t, c)[0]
	h, token := handlerFor(t, c, "video")
	for _, p := range []map[string]any{{"position": -1, "duration": 10}, {"position": 11, "duration": 10}, {"position": 0, "duration": 0}, {"position": 0, "duration": 99999999}} {
		p["revision"] = m.Revision
		w := call(t, h, "PUT", "/api/v1/media/"+m.ID+"/progress", token, p)
		if w.Code != 400 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := call(t, h, "PUT", "/api/v1/media/"+m.ID+"/progress", token, map[string]any{"position": 1, "duration": 10, "revision": "old"})
	if w.Code != 409 {
		t.Fatal("stale revision accepted")
	}
}

func TestSymlinkEscape(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	outside := t.TempDir()
	putFile(t, dir, "song.mp3", "inside")
	putFile(t, outside, "secret.mp3", "outside")
	_, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m := getItems(t, c)[0]
	link := filepath.Join(dir, "escape.mp3")
	if err = os.Symlink(filepath.Join(outside, "secret.mp3"), link); err != nil {
		t.Skipf("symlink unavailable on this host: %v", err)
	}
	if err = os.Remove(filepath.Join(dir, "song.mp3")); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(link, filepath.Join(dir, "song.mp3")); err != nil {
		t.Fatal(err)
	}
	f, _, err := c.openMedia(testContext(context.Background()), m.ID)
	if err == nil {
		f.Close()
		t.Fatal("escaped library root")
	}
}

func TestStoredPathCannotEscapeRoot(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "song.mp3", "inside")
	_, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	m := getItems(t, c)[0]
	// Even a corrupted catalog must not turn a media ID into an arbitrary file reader.
	_, err = c.db.Exec(`UPDATE media SET relative_path=? WHERE id=?`, "../outside.mp3", m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if f, _, err := c.openMedia(testContext(context.Background()), m.ID); err == nil {
		f.Close()
		t.Fatal("path escaped root")
	}
}

func dropAccountsForMigration(t *testing.T, c *catalog) {
	t.Helper()
	if _, e := c.db.Exec("DROP TABLE extension_preferences;DROP TABLE extension_grants;DROP TABLE extensions;DROP TABLE invitations;DROP TABLE sessions;DROP TABLE user_progress;DROP TABLE user_works;DROP TABLE accounts;ALTER TABLE playlists DROP COLUMN owner;"); e != nil {
		t.Fatal(e)
	}
}
func openMigrationCatalog(t *testing.T, dir string) (*catalog, error) {
	t.Helper()
	c, e := openCatalog(dir)
	if e == nil {
		e = c.createFirstAccount(context.Background(), account{ID: "test-user", Name: "TestAdmin", Admin: true}, "unused-test-hash")
	}
	return c, e
}
