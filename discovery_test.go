package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type discoveryFixture struct {
	calls int
	fail  bool
}

func (p *discoveryFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "fixture", Kinds: []string{"films", "series"}, ContractVersion: 1}
}
func (p *discoveryFixture) Search(context.Context, string, MetadataQuery) ([]MetadataCandidate, error) {
	return nil, nil
}
func (p *discoveryFixture) Details(_ context.Context, _ string, q MetadataSelection) (Metadata, error) {
	if p.fail {
		return Metadata{}, errors.New("Fournisseur temporairement indisponible.")
	}
	if q.Episode != 0 || q.Season != 0 {
		return Metadata{}, errors.New("episode detail used for series")
	}
	return Metadata{Title: "Fiche complète", Overview: "Résumé de l’œuvre entière", Cast: []string{"Actrice"}, Provider: "fixture", ProviderID: q.ID, ArtworkRef: "/poster.jpg"}, nil
}
func (p *discoveryFixture) Artwork(context.Context, string, string) ([]byte, error) {
	p.calls++
	return testPicture(), nil
}
func (p *discoveryFixture) Discover(_ context.Context, _ string, q DiscoveryQuery) (DiscoveryPage, error) {
	if p.fail {
		return DiscoveryPage{}, errors.New("Fournisseur temporairement indisponible.")
	}
	return DiscoveryPage{Page: q.Page, Pages: 2, Items: []Metadata{{Title: "Même identifiant", ProviderID: "opaque:abc", Provider: "fixture", ArtworkRef: "/poster.jpg"}, {Title: "Absent", ProviderID: "absent", Provider: "fixture"}}}, nil
}
func decodeWorks(t *testing.T, data []byte) []discoveryWork {
	t.Helper()
	var works []discoveryWork
	if err := json.Unmarshal(data, &works); err != nil {
		t.Fatal(err)
	}
	return works
}

func TestDiscoveryListsAvailabilityAndPersistence(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	fixture := &discoveryFixture{}
	c.providers = []MetadataProvider{fixture}
	mediaDir := t.TempDir()
	putFile(t, mediaDir, "Show.S01E01.mkv", "one")
	putFile(t, mediaDir, "Show.S01E02.mkv", "two")
	if _, err := c.scan(ctx, library{Path: mediaDir, Kind: "series"}, true); err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	for i, m := range items {
		md := Metadata{Title: "Titre de l’épisode", Series: "La série", Episode: i + 1, Season: 1, Overview: "Résumé de l’épisode", Provider: "fixture", ProviderID: "opaque:abc"}
		if err := c.saveMetadata(ctx, m, md); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.db.Exec(`INSERT INTO user_progress(user_id,media_id,revision,completed) SELECT 'test-user',id,revision,1 FROM media WHERE id=?`, items[0].ID); err != nil {
		t.Fatal(err)
	}
	h, token := handlerFor(t, c, "video,music")
	response := call(t, h, "GET", "/api/v1/discovery/library", "", nil)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	works := decodeWorks(t, response.Body.Bytes())
	if len(works) != 1 || len(works[0].LocalIDs) != 2 || !works[0].Watched || works[0].Metadata.Overview != "" || works[0].Metadata.Title != "La série" {
		t.Fatalf("wrong series grouping: %+v", works)
	}
	id := works[0].ID
	endpoint := "/api/v1/discovery/works/" + id
	for _, patch := range []map[string]bool{{"favorite": true}, {"watchlist": true}} {
		r := call(t, h, "PATCH", endpoint, token, patch)
		if r.Code != 200 {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	// A metadata refresh must not erase saved lists.
	r := call(t, h, "GET", endpoint, "", nil)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	var detail struct{ Work discoveryWork }
	if err := json.Unmarshal(r.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if !detail.Work.Favorite || !detail.Work.Watchlist || len(detail.Work.LocalIDs) != 2 || detail.Work.Metadata.Overview != "Résumé de l’œuvre entière" {
		t.Fatalf("detail overwritten lists: %+v", detail)
	}
	for range 2 {
		r = call(t, h, "GET", endpoint+"/poster", "", nil)
		if r.Code != 200 || r.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatal(r.Code)
		}
	}
	if fixture.calls != 1 {
		t.Fatalf("image not cached: %d", fixture.calls)
	}
	r = call(t, h, "POST", "/api/v1/discovery/browse", token, map[string]any{"provider": "fixture", "kind": "series", "seed": id, "page": 1})
	if r.Code != 200 || !strings.Contains(r.Body.String(), "Parce que vous aimez") {
		t.Fatal(r.Code, r.Body.String())
	}
	var page struct{ Items []discoveryWork }
	if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || len(page.Items[0].LocalIDs) != 0 {
		t.Fatal("seed or absent availability", page)
	}
	absentID := page.Items[0].ID
	r = call(t, h, "PATCH", "/api/v1/discovery/works/"+absentID, token, map[string]bool{"watchlist": true})
	if r.Code != 200 {
		t.Fatal(r.Code)
	}
	// Removing local availability does not remove a saved work or create a false play link.
	if _, err := c.db.Exec(`UPDATE media SET available=0`); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	next.providers = []MetadataProvider{fixture}
	h, token = handlerFor(t, next, "video,music")
	r = call(t, h, "GET", "/api/v1/discovery/library", "", nil)
	works = decodeWorks(t, r.Body.Bytes())
	if len(works) != 2 {
		t.Fatal("lost lists on restart", works)
	}
	for _, work := range works {
		if !work.Watchlist || len(work.LocalIDs) != 0 || work.Watched {
			t.Fatal("incorrect saved availability", work)
		}
	}
	r = call(t, h, "PATCH", endpoint, token, map[string]bool{"favorite": false})
	if r.Code != 200 {
		t.Fatal(r.Code)
	}
	saved, err := next.work(ctx, id)
	if err != nil || saved.Favorite || !saved.Watchlist {
		t.Fatal("independent fields lost", saved, err)
	}
	// Same opaque provider ID in two media kinds represents two different works.
	movie, err := next.rememberWork(ctx, "films", Metadata{Title: "Film", Provider: "fixture", ProviderID: "opaque:abc"}, false)
	if err != nil || movie == id {
		t.Fatal("cross-kind collision", err)
	}
}

func TestDiscoveryGuardsAndProviderFailure(t *testing.T) {
	c, _ := testCatalog(t)
	p := &discoveryFixture{}
	c.providers = []MetadataProvider{p}
	h, token := handlerFor(t, c, "video")
	valid := map[string]any{"provider": "fixture", "kind": "films", "query": "test", "page": 1}
	if r := call(t, h, "POST", "/api/v1/discovery/browse", "", valid); r.Code != 403 {
		t.Fatal(r.Code)
	}
	for _, bad := range []map[string]any{{"provider": "fixture", "kind": "music", "query": "test", "page": 1}, {"provider": "fixture", "kind": "films", "query": "test", "page": 51}, {"provider": "fixture", "kind": "films", "query": "a", "page": 1}, {"provider": "fixture", "kind": "films", "seed": "unknown", "page": 1}, {"provider": "fixture", "kind": "films", "query": "test", "page": 1, "url": "https://example.com"}} {
		if r := call(t, h, "POST", "/api/v1/discovery/browse", token, bad); r.Code != 400 {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	r := call(t, h, "POST", "/api/v1/discovery/browse", token, valid)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	var page struct{ Items []discoveryWork }
	if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	id := page.Items[0].ID
	for _, patch := range []any{map[string]any{}, map[string]string{"favorite": "true"}, map[string]string{"title": "spoof"}} {
		if r := call(t, h, "PATCH", "/api/v1/discovery/works/"+id, token, patch); r.Code != 400 {
			t.Fatal(r.Code)
		}
	}
	p.fail = true
	if r = call(t, h, "POST", "/api/v1/discovery/browse", token, valid); r.Code != 502 {
		t.Fatal(r.Code)
	}
	if r = call(t, h, "GET", "/api/v1/discovery/works/"+id, "", nil); r.Code != 200 || !strings.Contains(r.Body.String(), "temporairement indisponible") {
		t.Fatal("missing cached fallback", r.Code, r.Body.String())
	}
	h, token = handlerFor(t, c, "music")
	for _, path := range []string{"/api/v1/discovery/works/" + id, "/api/v1/discovery/works/" + id + "/poster"} {
		if r = call(t, h, "GET", path, "", nil); r.Code != 404 {
			t.Fatal(r.Code, path)
		}
	}
	if r = call(t, h, "POST", "/api/v1/discovery/browse", token, valid); r.Code != 400 {
		t.Fatal(r.Code)
	}
	if r = call(t, h, "PATCH", "/api/v1/discovery/works/"+id, token, map[string]bool{"favorite": true}); r.Code != 404 {
		t.Fatal(r.Code)
	}
}

func TestTMDBDiscoveryPaginationAndCache(t *testing.T) {
	c, _ := testCatalog(t)
	p := newTMDBProvider(c.db)
	requests := 0
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("language") != "fr-FR" || r.Header.Get("Authorization") != "Bearer fixture-secret" {
			t.Fatal("request parameters")
		}
		if requests == 1 && (r.URL.Path != "/3/search/movie" || r.URL.Query().Get("query") != "Iron Man") {
			t.Fatal(r.URL.Path)
		}
		if requests == 2 && r.URL.Path != "/3/tv/42/recommendations" {
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"total_pages":90,"results":[{"id":7,"title":"Titre","release_date":"2020-01-01","poster_path":"/poster.jpg"}]}`)), Header: make(http.Header)}, nil
	})}
	q := DiscoveryQuery{Kind: "films", Query: "Iron Man", Page: 2}
	for range 2 {
		page, err := p.Discover(testContext(context.Background()), "fixture-secret", q)
		if err != nil || page.Pages != 50 || len(page.Items) != 1 || page.Items[0].Year != 2020 || page.Items[0].ArtworkRef != "/poster.jpg" {
			t.Fatal(page, err)
		}
	}
	if requests != 1 {
		t.Fatal("cache miss", requests)
	}
	if _, err := p.Discover(testContext(context.Background()), "fixture-secret", DiscoveryQuery{Kind: "series", Seed: "42", Page: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Discover(testContext(context.Background()), "fixture-secret", DiscoveryQuery{Kind: "series", Seed: "../42", Page: 2}); err == nil {
		t.Fatal("unsafe seed accepted")
	}
}

func TestDiscoveryMigrationPreservesV3(t *testing.T) {
	c, dir := testCatalog(t)
	dropAccountsForMigration(t, c)
	removeRecognitionForLegacyFixture(t, c)
	if _, err := c.db.Exec(`DROP TABLE album_links; DROP TABLE playlist_art; ALTER TABLE media DROP COLUMN added_at; ALTER TABLE media DROP COLUMN last_played; DROP TABLE works; PRAGMA user_version=3; INSERT INTO playlists VALUES('saved','Ma playlist',7,'[]'); INSERT INTO provider_settings VALUES('example','fixture-secret');`); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openMigrationCatalog(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	var revision int
	if err = next.db.QueryRow(`SELECT revision FROM playlists WHERE id='saved'`).Scan(&revision); err != nil || revision != 7 {
		t.Fatal(err, revision)
	}
	if next.credential(testContext(context.Background()), "example") != "fixture-secret" {
		t.Fatal("credential lost")
	}
	var v int
	if err = next.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil || v != catalogSchemaVersion {
		t.Fatal(err, v)
	}
}

func TestDiscoveryCacheBoundAndLegacyProvider(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	id, err := c.rememberWork(ctx, "films", Metadata{Title: "À conserver", Provider: "fixture", ProviderID: "keep"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.db.Exec(`INSERT INTO user_works(user_id,work_id,favorite) VALUES('test-user',?,1)`, id); err != nil {
		t.Fatal(err)
	}
	tx, err := c.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1005; i++ {
		if _, err = tx.Exec(`INSERT INTO works(id,kind,data,touched) VALUES(?,'films','{}',?)`, fmt.Sprint(i), i); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err = c.rememberWork(ctx, "films", Metadata{Title: "Nouveau", Provider: "fixture", ProviderID: "new"}, false); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = c.db.QueryRow(`SELECT COUNT(*) FROM works`).Scan(&count); err != nil || count != 1001 {
		t.Fatal("unbounded transient cache", count, err)
	}
	if kept, err := c.work(ctx, id); err != nil || !kept.Favorite {
		t.Fatal("saved work evicted", err)
	}
	c.providers = []MetadataProvider{&sampleProvider{}, &discoveryFixture{}}
	info := c.providerInfo(ctx)
	if info[0].Discovery || !info[1].Discovery {
		t.Fatal("optional capability", info)
	}
	h, token := handlerFor(t, c, "video")
	r := call(t, h, "POST", "/api/v1/discovery/browse", token, map[string]any{"provider": "sample", "kind": "films", "query": "example", "page": 1})
	if r.Code != 400 {
		t.Fatal("legacy provider coerced", r.Code)
	}
}
