package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The entire acquisition flow uses loopback fixtures, never a user's services.
type fakeArr struct {
	mu           sync.Mutex
	id           string
	object       arrObject
	writes       []string
	queue        []arrObject
	episodes     []arrObject
	episodeFiles []arrObject
	failAdd      bool
	failQueue    bool
	lookup       arrObject
	commands     []arrObject
}

func (f *fakeArr) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("X-Api-Key") != "fixture-secret" {
		http.Error(w, "bad key", 401)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v3/")
	resource := "movie"
	identifier := "tmdbId"
	external := 42
	if f.id == "sonarr" {
		resource = "series"
		identifier = "tvdbId"
		external = 84
	}
	if r.Method != "GET" {
		f.writes = append(f.writes, r.Method+" "+path)
	}
	var output any = map[string]any{}
	switch {
	case path == "system/status":
		output = map[string]any{"appName": f.id}
	case path == "qualityprofile":
		output = []any{map[string]any{"id": 1, "name": "1080p"}, map[string]any{"id": 7, "name": "Existing"}}
	case path == "rootfolder":
		output = []any{map[string]any{"id": 1, "path": "/new"}}
	case path == resource && r.Method == "GET":
		output = []arrObject{}
		if f.object != nil {
			output = []arrObject{f.object}
		}
	case path == resource+"/lookup":
		output = []arrObject{f.lookup}
	case path == resource && r.Method == "POST":
		var v arrObject
		_ = json.NewDecoder(r.Body).Decode(&v)
		v["id"] = float64(12)
		v[identifier] = float64(external)
		f.object = v
		if f.failAdd {
			http.Error(w, "fixture-secret should never escape", 500)
			return
		}
		output = v
	case path == resource+"/12" && r.Method == "PUT":
		_ = json.NewDecoder(r.Body).Decode(&f.object)
		output = f.object
	case path == "episode":
		output = f.episodes
	case path == "episodefile":
		output = f.episodeFiles
	case path == "episode/monitor":
		var v arrObject
		_ = json.NewDecoder(r.Body).Decode(&v)
		for _, id := range v["episodeIds"].([]any) {
			for _, ep := range f.episodes {
				if ep["id"] == id {
					ep["monitored"] = v["monitored"]
				}
			}
		}
	case path == "command":
		var cmd arrObject
		_ = json.NewDecoder(r.Body).Decode(&cmd)
		f.commands = append(f.commands, cmd)
	case path == "queue":
		if f.failQueue {
			http.Error(w, "fixture-secret", 503)
			return
		}
		output = map[string]any{"records": f.queue, "totalRecords": len(f.queue)}
	default:
		http.Error(w, "unexpected "+r.Method+" "+path, 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(output)
}
func setupAcquisition(t *testing.T, kind string) (*catalog, http.Handler, string, arrConfig, string, *fakeArr) {
	t.Helper()
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	l, e := c.scan(ctx, library{Kind: kind, Path: dir}, true)
	if e != nil {
		t.Fatal(e)
	}
	id := "radarr"
	if kind == "series" {
		id = "sonarr"
	}
	f := &fakeArr{id: id, lookup: arrObject{"tmdbId": float64(42), "tvdbId": float64(84), "title": "Test", "seasons": []any{map[string]any{"seasonNumber": float64(0), "monitored": false}, map[string]any{"seasonNumber": float64(1), "monitored": true}, map[string]any{"seasonNumber": float64(2), "monitored": true}}}, queue: []arrObject{}, episodes: []arrObject{}}
	server := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(server.Close)
	cfg := arrConfig{ID: id, URL: server.URL, Key: "fixture-secret", Enabled: true, Profile: 1, Root: "/new", Library: l.ID}
	h, token := handlerFor(t, c, "video,music")
	resp := call(t, h, "PUT", "/api/v1/acquisition/settings", token, cfg)
	if resp.Code != 200 {
		t.Fatal(resp.Code, resp.Body.String())
	}
	if len(f.writes) != 0 {
		t.Fatal("settings must be read only remotely", f.writes)
	}
	work, e := c.rememberWork(ctx, kind, Metadata{Provider: "tmdb", ProviderID: "42", Title: "Test"}, true)
	if e != nil {
		t.Fatal(e)
	}
	if kind == "series" {
		tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tv/42/external_ids" {
				t.Errorf("unexpected TMDB path: %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"tvdb_id":84}`))
		}))
		t.Cleanup(tmdb.Close)
		p := newTMDBProvider(c.db)
		p.apiBase = tmdb.URL
		c.providers = []MetadataProvider{p}
		if err := c.setProviderCredential(context.Background(), "tmdb", "fixture-tmdb-token-123456"); err != nil {
			t.Fatal(err)
		}
	}
	return c, h, token, cfg, work, f
}
func confirmAcquisition(t *testing.T, h http.Handler, token, work string, season int) *httptest.ResponseRecorder {
	t.Helper()
	in := requestInput{WorkID: work, Season: season}
	r := call(t, h, "POST", "/api/v1/acquisition/preview", token, in)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	var p requestPreview
	_ = json.Unmarshal(r.Body.Bytes(), &p)
	in.Confirmation = p.Confirmation
	return call(t, h, "POST", "/api/v1/acquisition/requests", token, in)
}
func TestAcquisitionMoviePersistenceAndUncertainDelivery(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "ambiguous"}[ambiguous], func(t *testing.T) {
			c, h, token, cfg, work, f := setupAcquisition(t, "films")
			f.failAdd = ambiguous
			r := confirmAcquisition(t, h, token, work, -1)
			if r.Code != 201 {
				t.Fatal(r.Code, r.Body.String())
			}
			if strings.Contains(r.Body.String(), cfg.Key) {
				t.Fatal("secret leaked")
			}
			all, e := c.requests(testContext(context.Background()))
			if e != nil || len(all) != 1 {
				t.Fatal(all, e)
			}
			if all[0].Uncertain != ambiguous {
				t.Fatal(all[0])
			}
			if len(f.writes) != 1 || f.writes[0] != "POST movie" {
				t.Fatal(f.writes)
			}
			// Duplicate clicks and refreshes must not replay the remote add or search.
			r = call(t, h, "POST", "/api/v1/acquisition/requests", token, requestInput{WorkID: work, Season: -1})
			if r.Code != 200 {
				t.Fatal(r.Code)
			}
			c.syncRequests(testContext(context.Background()), videoKind, true)
			if len(f.writes) != 1 {
				t.Fatal("replayed", f.writes)
			}
			all, _ = c.requests(testContext(context.Background()))
			if ambiguous && all[0].State != "uncertain" {
				t.Fatal("ambiguous search lost", all[0])
			}
			var dbPath string
			_ = c.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&dbPath)
			c.close()
			reopened, err := openCatalog(filepath.Dir(dbPath))
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.close()
			again, err := reopened.requests(testContext(context.Background()))
			if err != nil || len(again) != 1 || again[0].ID != all[0].ID {
				t.Fatal(again, err)
			}
		})
	}
}
func TestAcquisitionExistingMovieAndChangedConfirmation(t *testing.T) {
	_, h, token, _, work, f := setupAcquisition(t, "films")
	f.object = arrObject{"id": float64(12), "tmdbId": float64(42), "path": "/keep/film", "qualityProfileId": float64(7), "monitored": false, "tags": []any{float64(9)}}
	input := requestInput{WorkID: work, Season: -1}
	r := call(t, h, "POST", "/api/v1/acquisition/preview", token, input)
	var p requestPreview
	_ = json.Unmarshal(r.Body.Bytes(), &p)
	if !p.Existing || p.Root != "/keep/film" || p.Profile != "Existing" {
		t.Fatal(p)
	}
	input.Confirmation = p.Confirmation
	f.object["path"] = "/changed/film"
	r = call(t, h, "POST", "/api/v1/acquisition/requests", token, input)
	if r.Code != 409 || len(f.writes) != 0 {
		t.Fatal(r.Code, f.writes)
	}
	r = confirmAcquisition(t, h, token, work, -1)
	if r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	if word(f.object["path"]) != "/changed/film" || number(f.object["qualityProfileId"]) != 7 || !truth(f.object["monitored"]) || len(f.object["tags"].([]any)) != 1 {
		t.Fatal(f.object)
	}
	if len(f.commands) != 1 || word(f.commands[0]["name"]) != "MoviesSearch" {
		t.Fatal(f.commands)
	}
}
func TestAcquisitionSeasonScopeAndAvailability(t *testing.T) {
	c, h, token, cfg, work, f := setupAcquisition(t, "series")
	ctx := testContext(context.Background())
	f.object = arrObject{"id": float64(12), "tvdbId": float64(84), "path": "/keep/series", "qualityProfileId": float64(7), "monitorNewItems": "all", "monitored": true, "seasons": []any{map[string]any{"seasonNumber": float64(1), "monitored": false}, map[string]any{"seasonNumber": float64(2), "monitored": false}}}
	f.episodes = []arrObject{{"id": float64(1), "seasonNumber": float64(1), "episodeNumber": float64(1), "hasFile": false, "monitored": false}, {"id": float64(2), "seasonNumber": float64(1), "episodeNumber": float64(2), "hasFile": false, "monitored": false}, {"id": float64(3), "seasonNumber": float64(2), "episodeNumber": float64(1), "hasFile": false, "monitored": false}}
	r := confirmAcquisition(t, h, token, work, 1)
	if r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	seasons := f.object["seasons"].([]any)
	if !truth(seasons[0].(map[string]any)["monitored"]) || truth(seasons[1].(map[string]any)["monitored"]) {
		t.Fatal(seasons)
	}
	if !truth(f.episodes[0]["monitored"]) || truth(f.episodes[2]["monitored"]) || word(f.object["monitorNewItems"]) != "all" {
		t.Fatal(f.object, f.episodes)
	}
	if len(f.commands) != 1 || word(f.commands[0]["name"]) != "SeasonSearch" || number(f.commands[0]["seasonNumber"]) != 1 {
		t.Fatal(f.commands)
	}
	all, _ := c.requests(ctx)
	v := all[0]
	seasons[0].(map[string]any)["monitored"] = false
	if err := c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "unmonitored" {
		t.Fatal(v, err)
	}
	seasons[0].(map[string]any)["monitored"] = true
	f.queue = []arrObject{{"seriesId": float64(12), "episodeId": float64(1), "size": float64(100), "sizeleft": float64(25)}, {"seriesId": float64(12), "episodeId": float64(3), "size": float64(100), "sizeleft": float64(100)}}
	if err := c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "downloading" || v.Progress != 75 {
		t.Fatal(v, err)
	}
	f.queue = nil
	f.episodes[0]["hasFile"] = true
	c.requestScans = map[string]time.Time{cfg.Library: time.Now()}
	if err := c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "imported" || v.Local != 0 {
		t.Fatal(v, err)
	}
	// A file only counts after it is readable and identified in the mapped library.
	l, _ := c.requestLibrary(ctx, cfg)
	putFile(t, l.Path, "Test.S01E01.mkv", "video fixture")
	_, err := c.scan(ctx, l, false)
	if err != nil {
		t.Fatal(err)
	}
	m := getItems(t, c)[0]
	err = c.saveMetadata(ctx, m, Metadata{Provider: "tmdb", ProviderID: "42", Title: "Épisode", Season: 1, Episode: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "partial" || v.Local != 1 || v.Total != 2 {
		t.Fatal(v, err)
	}
	f.episodes[1]["hasFile"] = true
	putFile(t, l.Path, "Test.S01E02.mkv", "second fixture")
	_, _ = c.scan(ctx, l, false)
	for _, m := range getItems(t, c) {
		if strings.Contains(m.Filename, "E02") {
			_ = c.saveMetadata(ctx, m, Metadata{Provider: "tmdb", ProviderID: "42", Title: "Épisode 2", Season: 1, Episode: 2})
		}
	}
	if err = c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "available" || v.Local != 2 {
		t.Fatal(v, err)
	}
	if err = os.Remove(filepath.Join(l.Path, "Test.S01E02.mkv")); err != nil {
		t.Fatal(err)
	}
	if err = c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "partial" {
		t.Fatal("unreadable file counted", v, err)
	}
}
func TestAcquisitionNewSeasonMonitoringAndGuards(t *testing.T) {
	c, h, token, cfg, work, f := setupAcquisition(t, "series")
	r := confirmAcquisition(t, h, token, work, 2)
	if r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	seasons := f.object["seasons"].([]any)
	for _, s := range seasons {
		v := s.(map[string]any)
		if truth(v["monitored"]) != (number(v["seasonNumber"]) == 2) {
			t.Fatal(v)
		}
	}
	if word(f.object["monitorNewItems"]) != "none" || word(f.object["addOptions"].(map[string]any)["monitor"]) != "skip" {
		t.Fatal(f.object)
	}
	r = call(t, h, "GET", "/api/v1/acquisition/settings", "", nil)
	if strings.Contains(r.Body.String(), cfg.Key) || !strings.Contains(r.Body.String(), `"configured":true`) {
		t.Fatal(r.Body.String())
	}
	if r = call(t, h, "POST", "/api/v1/acquisition/preview", "", requestInput{WorkID: work, Season: 1}); r.Code != 403 {
		t.Fatal(r.Code)
	}
	disabled, disabledToken := handlerFor(t, c, "music")
	if r = call(t, disabled, "POST", "/api/v1/acquisition/requests", disabledToken, requestInput{WorkID: work, Season: 1}); r.Code != 404 {
		t.Fatal(r.Code)
	}
	before := len(f.writes)
	if r = call(t, h, "POST", "/api/v1/acquisition/sonarr/pause", token, nil); r.Code != 204 {
		t.Fatal(r.Code)
	}
	c.syncRequests(testContext(context.Background()), videoKind, true)
	if len(f.writes) != before {
		t.Fatal(f.writes)
	}
	all, _ := c.requests(testContext(context.Background()))
	if all[0].State != "paused" {
		t.Fatal(all)
	}
	cfg.URL = "http://new-host:8989"
	cfg.Key = ""
	if _, err := c.configInput(testContext(context.Background()), cfg); err == nil {
		t.Fatal("old secret reused for new host")
	}
}
func TestAcquisitionURLAndRedirectSafety(t *testing.T) {
	for _, raw := range []string{"file:///x", "http://user:secret@localhost", "http://localhost?apikey=x", "http://localhost/#fragment", "https://"} {
		if _, err := cleanArrURL(raw); err == nil {
			t.Fatal(raw)
		}
	}
	if got, err := cleanArrURL("http://nas:7878/radarr/api/v3/"); err != nil || got != "http://nas:7878/radarr" {
		t.Fatal(got, err)
	}
	touched := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { touched = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	err := (arrClient{arrConfig{URL: source.URL, Key: "fixture-secret"}}).call(testContext(context.Background()), "GET", "system/status", nil, nil)
	if err == nil || touched || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal(err, touched)
	}
}
func TestAcquisitionSchemaFiveUpgrade(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	id, err := c.rememberWork(ctx, "films", Metadata{Provider: "tmdb", ProviderID: "42", Title: "Conservé"}, true)
	if err != nil {
		t.Fatal(err)
	}
	dropAccountsForMigration(t, c)
	removeRecognitionForLegacyFixture(t, c)
	_, err = c.db.Exec(`DROP TABLE acquisition_requests; DROP TABLE acquisition_settings; PRAGMA user_version=5;`)
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openMigrationCatalog(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	w, err := next.work(ctx, id)
	if err != nil || w.Metadata.Title != "Conservé" {
		t.Fatal(w, err)
	}
	var version int
	_ = next.db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != catalogSchemaVersion {
		t.Fatal(version)
	}
}

func TestAcquisitionArrivalScanAndOfflineRecovery(t *testing.T) {
	c, h, token, cfg, work, f := setupAcquisition(t, "films")
	if r := confirmAcquisition(t, h, token, work, -1); r.Code != 201 {
		t.Fatal(r.Code)
	}
	ctx := testContext(context.Background())
	l, _ := c.requestLibrary(ctx, cfg)
	putFile(t, l.Path, "Test.2020.mkv", "new arrival")
	f.object["hasFile"] = true
	c.syncRequests(ctx, videoKind, true)
	items := getItems(t, c)
	if len(items) != 1 {
		t.Fatal("arrival scan did not run", items)
	}
	all, _ := c.requests(ctx)
	if all[0].State != "imported" {
		t.Fatal("unidentified must not be ready", all)
	}
	if err := c.saveMetadata(ctx, items[0], Metadata{Title: "Test", Provider: "tmdb", ProviderID: "42"}); err != nil {
		t.Fatal(err)
	}
	c.syncRequests(ctx, videoKind, true)
	all, _ = c.requests(ctx)
	if all[0].State != "available" {
		t.Fatal(all)
	}
	f.failQueue = true
	c.syncRequests(ctx, videoKind, true)
	all, _ = c.requests(ctx)
	if all[0].State != "unreachable" || strings.Contains(all[0].Message, cfg.Key) {
		t.Fatal(all)
	}
	f.failQueue = false
	c.syncRequests(ctx, videoKind, true)
	all, _ = c.requests(ctx)
	if all[0].State != "available" || len(f.writes) != 1 {
		t.Fatal(all, f.writes)
	}
	// Requested works are kept when the transient discovery cache is pruned.
	for i := 0; i < 1002; i++ {
		_, err := c.rememberWork(ctx, "films", Metadata{Provider: "tmdb", ProviderID: fmt.Sprint(100 + i), Title: "Other"}, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.work(ctx, work); err != nil {
		t.Fatal("requested work pruned", err)
	}
}

func TestAcquisitionMissingExternalIdentity(t *testing.T) {
	c, h, token, _, work, f := setupAcquisition(t, "series")
	p := c.provider("tmdb").(*tmdbProvider)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"tvdb_id":null}`)) }))
	defer server.Close()
	p.apiBase = server.URL
	r := call(t, h, "POST", "/api/v1/acquisition/preview", token, requestInput{WorkID: work, Season: 1})
	if r.Code != 400 || !strings.Contains(r.Body.String(), "TVDB") || len(f.writes) != 0 {
		t.Fatal(r.Code, r.Body.String(), f.writes)
	}
}
