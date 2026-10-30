package main

import (
	"context"
	"encoding/json"
	"io"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type tmdbFixtureTransport func(*http.Request) (*http.Response, error)

func (f tmdbFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTMDBExtensionCatalogBridge(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	calls := 0
	p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.themoviedb.org" || r.Header.Get("Authorization") != "Bearer fixture-token" || r.URL.Query().Get("language") != appLanguage {
			t.Error("host service authorization/language")
		}
		body := `{"results":[{"id":42,"title":"Le Voyage","original_title":"The Journey","release_date":"2024-01-01","overview":"Une aventure."}]}`
		if r.URL.Path == "/3/search/tv" {
			body = `{"results":[{"id":43,"name":"La Série","original_name":"The Show","first_air_date":"2023-02-01"}]}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	if err := p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	if p.extensionStatus()["loaded"] {
		t.Fatal("engine should load lazily")
	}
	q := MetadataQuery{Kind: "films", Title: "Voyage", Year: 2024}
	result, err := p.Search(ctx, "fixture-token", q)
	if err != nil || len(result) != 1 || result[0].ID != "42" || result[0].OriginalTitle != "The Journey" {
		t.Fatal(result, err)
	}
	if !p.extensionStatus()["loaded"] {
		t.Fatal("WASM not used")
	}
	if _, err = p.Search(ctx, "fixture-token", q); err != nil || calls != 1 {
		t.Fatal("cache not reused", calls, err)
	}
	result, err = p.Search(ctx, "fixture-token", MetadataQuery{Kind: "series", Title: "Série", Year: 2023})
	if err != nil || len(result) != 1 || result[0].Title != "La Série" || result[0].Year != 2023 {
		t.Fatal(result, err)
	}
	var key, body string
	if err = c.db.QueryRow(`SELECT key,body FROM provider_cache LIMIT 1`).Scan(&key, &body); err != nil || strings.Contains(key+body, "fixture-token") {
		t.Fatal("cache", err)
	}
	before := body
	if err = p.setExtension(ctx, false); err != nil {
		t.Fatal(err)
	}
	if p.extensionStatus()["loaded"] {
		t.Fatal("engine not released")
	}
	if err = c.db.QueryRow(`SELECT body FROM provider_cache LIMIT 1`).Scan(&body); err != nil || body != before {
		t.Fatal("cache lost")
	}
	result, err = p.Search(ctx, "fixture-token", q)
	if err == nil || len(result) != 0 || calls != 2 {
		t.Fatal("disabled module fallback", result, calls, err)
	}
}

func TestTMDBServiceGrantRejectsUnrequestedAccess(t *testing.T) {
	q := metadata.Query{Kind: "films", Title: "Voyage", Year: 2024, Language: "fr-FR"}
	good := map[string]any{"path": "/search/movie", "query": map[string]string{"query": "Voyage", "year": "2024", "include_adult": "false", "language": "fr-FR"}}
	raw, _ := json.Marshal(good)
	if _, _, err := tmdbSearchGrant("get", raw, q); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		`{"path":"https://evil.invalid","query":{}}`,
		`{"path":"/search/movie/../account","query":{}}`,
		`{"path":"/search/movie","query":{"query":"Other","year":"2024","include_adult":"false","language":"fr-FR"}}`,
		`{"path":"/search/movie","query":{"query":"Voyage","year":"2024","include_adult":"false","language":"fr-FR","api_key":"injected"}}`,
		`{"path":"/search/movie","query":{},"headers":{"Authorization":"injected"}}`,
	}
	for _, body := range cases {
		if _, _, err := tmdbSearchGrant("get", json.RawMessage(body), q); err == nil {
			t.Fatal("accepted", body)
		}
	}
	if _, _, err := tmdbSearchGrant("post", raw, q); err == nil {
		t.Fatal("write accepted")
	}
}
func TestTMDBRuntimePermissionsPersistenceAndSafeMode(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "PUT"} {
		if w := reader.request(method, "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": true}); w.Code != 403 {
			t.Fatal(method, w.Code)
		}
	}
	if w := admin.request("PUT", "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": true}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var enabled bool
	if err := c.db.QueryRow(`SELECT enabled FROM provider_runtime WHERE id='tmdb'`).Scan(&enabled); err != nil || !enabled {
		t.Fatal("preference not saved", err)
	}
	c.extensionSafeMode = true
	if w := admin.request("GET", "/api/v1/metadata/tmdb-runtime", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"effective":false`) {
		t.Fatal(w.Body.String())
	}
	if w := admin.request("PUT", "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": true}); w.Code != 400 {
		t.Fatal("safe mode bypass")
	}
	if w := admin.request("PUT", "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": false}); w.Code != 200 {
		t.Fatal(w.Code)
	}
}

func TestTMDBRuntimeMigrationManualChoiceAndRestart(t *testing.T) {
	c, dir := testCatalog(t)
	mediaDir := t.TempDir()
	putFile(t, mediaDir, "Voyage.2024.mp4", "fixture")
	ctx := testContext(context.Background())
	if _, err := c.scan(ctx, library{Path: mediaDir, Kind: "films"}, true); err != nil {
		t.Fatal(err)
	}
	removeRecognitionForLegacyFixture(t, c)
	if _, err := c.db.Exec(`DROP TABLE provider_runtime; PRAGMA user_version=10`); err != nil {
		t.Fatal(err)
	}
	c.close()
	reopened, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := reopened.provider("tmdb").(*tmdbProvider)
	if !p.extensionStatus()["enabled"] {
		t.Fatal("legacy engine was not migrated")
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("backup missing")
	}
	p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"id":42,"title":"Le Voyage","release_date":"2024-01-01","overview":"Fiche choisie manuellement","videos":{"results":[]}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	item := getItems(t, reopened)[0]
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	if err = reopened.applyProvider(ctx, p, "fixture-token", item, "42", true); err != nil {
		t.Fatal(err)
	}
	if err = p.setExtension(ctx, false); err != nil {
		t.Fatal(err)
	}
	md := getItems(t, reopened)[0].Metadata
	if !md.Manual || md.ProviderID != "42" || md.Provider != "tmdb" || md.Title != "Le Voyage" {
		t.Fatal("lost selection", md)
	}
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	reopened.close()
	again, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer again.close()
	status := again.provider("tmdb").(*tmdbProvider).extensionStatus()
	if !status["enabled"] || status["loaded"] {
		t.Fatal("restart preference/lazy loading", status)
	}
	md = getItems(t, again)[0].Metadata
	if !md.Manual || md.ProviderID != "42" {
		t.Fatal("selection lost after restart", md)
	}
}
