package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const accountTestPassword = "Une phrase de test 2026!"

func TestAccountsDownloadDestinationGrant(t *testing.T) {
	c, _, _, cfg, work, f := setupAcquisition(t, "films")
	mods, _ := enabledModules("video,music")
	h := newHandler(fstest.MapFS{"index.html": {Data: []byte("ui")}}, mods, c)
	a, e := c.account(context.Background(), "test-user")
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	if e = c.issueSession(w, httptest.NewRequest("GET", "http://127.0.0.1", nil), a); e != nil {
		t.Fatal(e)
	}
	var session struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	admin := &accountClient{t: t, h: h, cookie: w.Result().Cookies()[0], token: session.Token}
	admin.create("Requester", accountTestPassword, map[string]libraryGrant{cfg.Library: {View: true, Request: true}})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Requester", "password": accountTestPassword})
	result := reader.request("POST", "/api/v1/acquisition/preview", map[string]any{"workId": work, "season": -1})
	if result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	var preview struct{ Confirmation string }
	_ = json.Unmarshal(result.Body.Bytes(), &preview)
	result = reader.request("POST", "/api/v1/acquisition/requests", map[string]any{"workId": work, "season": -1, "confirmation": preview.Confirmation})
	if result.Code != 201 {
		t.Fatal(result.Code, result.Body.String())
	}
	f.mu.Lock()
	writes := len(f.writes)
	f.mu.Unlock()
	if writes == 0 {
		t.Fatal("authorized request never reached fixture")
	}
	result = reader.request("GET", "/api/v1/acquisition/requests", nil)
	if result.Code != 200 || !strings.Contains(result.Body.String(), work) || strings.Contains(result.Body.String(), cfg.URL) {
		t.Fatal("request projection", result.Body.String())
	}
	if result = reader.request("PUT", "/api/v1/acquisition/settings", cfg); result.Code != 403 {
		t.Fatal("request granted configuration rights")
	}
}

type accountClient struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
	token  string
}

func (a *accountClient) request(method, path string, body any) *httptest.ResponseRecorder {
	a.t.Helper()
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, "http://127.0.0.1"+path, bytes.NewReader(b))
	r.Header.Set("X-Mosaune-Token", a.token)
	if a.cookie != nil {
		r.AddCookie(a.cookie)
	}
	w := httptest.NewRecorder()
	a.h.ServeHTTP(w, r)
	return w
}
func (a *accountClient) enter(action string, body any) {
	a.t.Helper()
	w := a.request("GET", "/api/v1/auth/state", nil)
	var state struct{ Token string }
	if json.Unmarshal(w.Body.Bytes(), &state) != nil {
		a.t.Fatal("state")
	}
	a.token = state.Token
	w = a.request("POST", "/api/v1/auth/"+action, body)
	if w.Code != 200 {
		a.t.Fatalf("enter %s: %d %s", action, w.Code, w.Body.String())
	}
	var result struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	a.token = result.Token
	a.cookie = w.Result().Cookies()[0]
}
func accountFixture(t *testing.T) (*catalog, http.Handler, *accountClient) {
	t.Helper()
	c, e := openCatalog(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { c.close() })
	mods, _ := enabledModules("video,music")
	h := newHandler(fstest.MapFS{"index.html": {Data: []byte("ui")}}, mods, c)
	admin := &accountClient{t: t, h: h}
	return c, h, admin
}
func (a *accountClient) create(name, password string, grants map[string]libraryGrant) (account, string) {
	a.t.Helper()
	w := a.request("POST", "/api/v1/accounts", map[string]any{"account": account{Name: name, Kinds: []string{"films", "series", "music"}, Libraries: grants, Theme: "light"}, "password": password})
	if w.Code != 200 {
		a.t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		User       account
		Invitation string
	}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	return result.User, result.Invitation
}
func TestAccountsBootstrapInvitationAndSessions(t *testing.T) {
	c, h, admin := accountFixture(t)
	for _, path := range []string{"/api/v1/status", "/api/v1/library", "/api/v1/metadata/providers", "/api/v1/media/guess/stream", "/api/v1/posters/media/guess", "/api/v1/artwork/guess"} {
		if w := admin.request("GET", path, nil); w.Code != 401 {
			t.Fatalf("anonymous %s: %d", path, w.Code)
		}
	}
	w := admin.request("GET", "/api/v1/auth/state", nil)
	var state struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	admin.token = state.Token
	if strings.Contains(w.Body.String(), c.auth.code) {
		t.Fatal("bootstrap secret exposed")
	}
	if w = admin.request("POST", "/api/v1/auth/setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": "wrong"}); w.Code != 403 {
		t.Fatal(w.Code)
	}
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	if !admin.cookie.HttpOnly || admin.cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	var hash string
	_ = c.db.QueryRow(`SELECT password FROM accounts WHERE name='Admin'`).Scan(&hash)
	if strings.Contains(hash, accountTestPassword) || !strings.HasPrefix(hash, "$argon2id$") || !verifyPassword(hash, accountTestPassword) {
		t.Fatal("password storage")
	}
	var stored string
	_ = c.db.QueryRow(`SELECT id FROM sessions LIMIT 1`).Scan(&stored)
	if stored == admin.cookie.Value {
		t.Fatal("raw session stored")
	}
	invited, code := admin.create("Alice", "", map[string]libraryGrant{})
	if !invited.Pending || code == "" {
		t.Fatal("missing invitation")
	}
	if w = admin.request("GET", "/api/v1/accounts", nil); strings.Contains(w.Body.String(), code) || strings.Contains(w.Body.String(), hash) {
		t.Fatal("secret in account list")
	}
	alice := &accountClient{t: t, h: h}
	alice.enter("activate", map[string]string{"code": code, "password": accountTestPassword})
	anonymous := &accountClient{t: t, h: h}
	w = anonymous.request("GET", "/api/v1/auth/state", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	anonymous.token = state.Token
	if w = anonymous.request("POST", "/api/v1/auth/activate", map[string]string{"code": code, "password": accountTestPassword}); w.Code != 403 {
		t.Fatal("invitation replay", w.Code)
	}
	if w = alice.request("GET", "/api/v1/accounts", nil); w.Code != 403 {
		t.Fatal("nonadmin accounts", w.Code)
	}
	if w = alice.request("PUT", "/api/v1/account/preferences", map[string]string{"theme": "dark"}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	oldToken := alice.token
	alice.token = admin.token
	if w = alice.request("PUT", "/api/v1/account/preferences", map[string]string{"theme": "light"}); w.Code != 403 {
		t.Fatal("cross-session CSRF", w.Code)
	}
	alice.token = oldToken
	invited.Pending = false
	if w = admin.request("PUT", "/api/v1/accounts/"+invited.ID, map[string]any{"account": invited, "password": "Un autre long mot de passe"}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = alice.request("GET", "/api/v1/library", nil); w.Code != 401 {
		t.Fatal("reset did not revoke", w.Code)
	}
	alice = &accountClient{t: t, h: h}
	alice.enter("login", map[string]string{"name": "Alice", "password": "Un autre long mot de passe"})
	if w = alice.request("POST", "/api/v1/auth/logout", nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if w = alice.request("GET", "/api/v1/status", nil); w.Code != 401 {
		t.Fatal("logout did not revoke")
	}
	direct, directInvite := admin.create("Bob", accountTestPassword, nil)
	if direct.Pending || directInvite != "" {
		t.Fatal("explicit password should activate")
	}
	bob := &accountClient{t: t, h: h}
	bob.enter("login", map[string]string{"name": "Bob", "password": accountTestPassword})
	if w = bob.request("PUT", "/api/v1/account/password", map[string]string{"current": accountTestPassword, "password": "Encore une phrase assez longue"}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = bob.request("GET", "/api/v1/status", nil); w.Code != 401 {
		t.Fatal("password change session survived")
	}
}
func TestAccountsPrivateDataAndDirectAccess(t *testing.T) {
	c, h, admin := accountFixture(t)
	ctx := context.Background()
	dir := t.TempDir()
	putFile(t, dir, "film.mp4", "fixture movie")
	visible, e := c.scan(ctx, library{Name: "Visible", Path: dir, Kind: "films"}, true)
	if e != nil {
		t.Fatal(e)
	}
	hiddenDir := t.TempDir()
	putFile(t, hiddenDir, "secret.mp4", "hidden movie")
	hidden, e := c.scan(ctx, library{Name: "Hidden", Path: hiddenDir, Kind: "films"}, true)
	if e != nil {
		t.Fatal(e)
	}
	items, _ := c.items(ctx)
	var m, secret media
	for _, i := range items {
		if i.LibraryID == visible.ID {
			m = i
		} else {
			secret = i
		}
	}
	musicDir := t.TempDir()
	putFile(t, musicDir, "song.mp3", "music")
	music, e := c.scan(ctx, library{Name: "Music", Path: musicDir, Kind: "music"}, true)
	if e != nil {
		t.Fatal(e)
	}
	all, _ := c.items(ctx)
	var track media
	for _, i := range all {
		if i.LibraryID == music.ID {
			track = i
		}
	}
	id, e := c.rememberWork(ctx, "films", Metadata{Title: "Private favorite", Provider: "fixture", ProviderID: "42"}, true)
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.db.Exec(`UPDATE media SET position=42,duration=100,completed=0,last_played=12 WHERE id=?`, m.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.db.Exec(`UPDATE works SET favorite=1,watchlist=1 WHERE id=?`, id)
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.db.Exec(`INSERT INTO playlists(id,name,revision,tracks) VALUES('old','Ancienne playlist',1,'[]')`)
	if e != nil {
		t.Fatal(e)
	}
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	w := admin.request("GET", "/api/v1/library", nil)
	if !strings.Contains(w.Body.String(), `"position":42`) {
		t.Fatal("legacy progress missing", w.Body.String())
	}
	w = admin.request("GET", "/api/v1/discovery/library", nil)
	if !strings.Contains(w.Body.String(), `"favorite":true`) {
		t.Fatal("legacy favorite missing", w.Body.String())
	}
	aliceAccount, _ := admin.create("Alice", accountTestPassword, map[string]libraryGrant{visible.ID: {View: true, Play: true}, music.ID: {View: true, Play: true}})
	alice := &accountClient{t: t, h: h}
	alice.enter("login", map[string]string{"name": "Alice", "password": accountTestPassword})
	w = alice.request("GET", "/api/v1/library", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), hidden.ID) || strings.Contains(w.Body.String(), filepath.ToSlash(dir)) || strings.Contains(w.Body.String(), `"position":42`) {
		t.Fatal("library isolation", w.Body.String())
	}
	for _, path := range []string{"/api/v1/media/" + secret.ID + "/stream", "/api/v1/posters/media/" + secret.ID, "/api/v1/music/playlists/old/cover", "/api/v1/acquisition/settings", "/api/v1/metadata/queue", "/api/v1/library/scan"} {
		if w = alice.request("GET", path, nil); w.Code != 403 {
			t.Fatalf("denied %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w = alice.request("GET", "/api/v1/media/"+m.ID+"/stream", nil); w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("stream", w.Code, w.Header())
	}
	if w = alice.request("PUT", "/api/v1/media/"+m.ID+"/identify", map[string]string{}); w.Code != 403 {
		t.Fatal("edit denied", w.Code)
	}
	if w = alice.request("POST", "/api/v1/acquisition/preview", map[string]any{"workId": id, "season": -1}); w.Code != 403 {
		t.Fatal("request denied", w.Code, w.Body.String())
	}
	if w = alice.request("PUT", "/api/v1/media/"+m.ID+"/progress", map[string]any{"position": 7, "duration": 100, "revision": m.Revision}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = admin.request("GET", "/api/v1/library", nil); !strings.Contains(w.Body.String(), `"position":42`) {
		t.Fatal("admin progress overwritten")
	}
	w = alice.request("GET", "/api/v1/music/playlists", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), "Ancienne") {
		t.Fatal("private playlists", w.Body.String())
	}
	if w = alice.request("PUT", "/api/v1/music/playlists/old", map[string]any{"name": "stolen", "revision": 1, "trackIds": []string{}}); w.Code != 403 {
		t.Fatal("playlist IDOR", w.Code)
	}
	if w = alice.request("PATCH", "/api/v1/discovery/works/"+id, map[string]bool{"favorite": false}); w.Code != 200 || strings.Contains(w.Body.String(), `"watchlist":true`) {
		t.Fatal("personal list", w.Code, w.Body.String())
	}
	if w = admin.request("GET", "/api/v1/discovery/library", nil); !strings.Contains(w.Body.String(), `"favorite":true`) {
		t.Fatal("favorite overwritten")
	}
	if w = alice.request("POST", "/api/v1/music/playlists", map[string]any{"name": "Alice music", "trackIds": []string{track.ID}}); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	aliceAccount.Libraries[visible.ID] = libraryGrant{View: true}
	if w = admin.request("PUT", "/api/v1/accounts/"+aliceAccount.ID, map[string]any{"account": aliceAccount}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = alice.request("GET", "/api/v1/media/"+m.ID+"/stream", nil); w.Code != 401 {
		t.Fatal("permission change session not revoked")
	}
	alice = &accountClient{t: t, h: h}
	alice.enter("login", map[string]string{"name": "Alice", "password": accountTestPassword})
	if w = alice.request("GET", "/api/v1/media/"+m.ID+"/stream", nil); w.Code != 403 {
		t.Fatal("view is not play")
	}
	// A newly added library never inherits a grant.
	if canLibrary(context.WithValue(ctx, identityKey{}, &identity{account: aliceAccount}), "new-library", "films", "view") {
		t.Fatal("grant default")
	}
}
func TestAccountsRateLimitOriginAndExpiredInvite(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	_, code := admin.create("Pending", "", nil)
	_, _ = c.db.Exec(`UPDATE invitations SET expires=?`, time.Now().Add(-time.Minute).Unix())
	anonymous := &accountClient{t: t, h: h}
	w := anonymous.request("GET", "/api/v1/auth/state", nil)
	var state struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	anonymous.token = state.Token
	if w = anonymous.request("POST", "/api/v1/auth/activate", map[string]string{"code": code, "password": accountTestPassword}); w.Code != 403 {
		t.Fatal("expired invite")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/auth/login", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("X-Mosaune-Token", state.Token)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("origin")
	}
	c.auth.mu.Lock()
	c.auth.attempts = 10
	c.auth.window = time.Now()
	c.auth.mu.Unlock()
	if w = anonymous.request("POST", "/api/v1/auth/login", map[string]string{"name": "Admin", "password": accountTestPassword}); w.Code != 429 {
		t.Fatal("rate limit", w.Code)
	}
}

func TestAccountsSessionRestartAndAdminRecovery(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	var file string
	if e := c.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&file); e != nil {
		t.Fatal(e)
	}
	c.close()
	next, e := openCatalog(filepath.Dir(file))
	if e != nil {
		t.Fatal(e)
	}
	defer next.close()
	mods, _ := enabledModules("video,music")
	h := newHandler(fstest.MapFS{"index.html": {Data: []byte("ui")}}, mods, next)
	admin.h = h
	if w := admin.request("GET", "/api/v1/library", nil); w.Code != 200 {
		t.Fatal("session lost on restart", w.Code)
	}
	next.auth.recovery = "Admin"
	next.auth.code = "local-recovery-fixture-code"
	recovered := &accountClient{t: t, h: h}
	recovered.enter("recover", map[string]string{"name": "Admin", "code": next.auth.code, "password": "Une nouvelle phrase administrateur"})
	if w := admin.request("GET", "/api/v1/library", nil); w.Code != 401 {
		t.Fatal("recovery did not revoke old session")
	}
	if next.auth.code != "" || next.auth.recovery != "" {
		t.Fatal("recovery code reusable")
	}
}
