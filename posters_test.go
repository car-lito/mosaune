package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type posterFixture struct{ sampleProvider }

func (p *posterFixture) Posters(context.Context, string, MetadataSelection, string) ([]PosterOption, error) {
	return []PosterOption{{Ref: "/choice.jpg", Language: "fr", Width: 600, Height: 900}}, nil
}
func (p *posterFixture) Artwork(context.Context, string, string) ([]byte, error) {
	return testPicture(), nil
}
func uploadPoster(h http.Handler, route, token string, b []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest("PUT", "http://127.0.0.1"+route, bytes.NewReader(b))
	r.Header.Set("X-Mosaune-Token", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestPosterChoicePersistenceAndAutomaticFallback(t *testing.T) {
	ctx := testContext(context.Background())
	c, dir := testCatalog(t)
	mediaDir := t.TempDir()
	putFile(t, mediaDir, "Example.2020.mp4", "test")
	l, e := c.scan(ctx, library{Path: mediaDir, Kind: "films"}, true)
	if e != nil {
		t.Fatal(e)
	}
	m := getItems(t, c)[0]
	md := Metadata{Title: "Example", Provider: "sample", ProviderID: "42", MatchState: "matched"}
	if e = c.saveMetadata(ctx, m, md); e != nil {
		t.Fatal(e)
	}
	work, e := c.rememberWork(ctx, "films", md, true)
	if e != nil {
		t.Fatal(e)
	}
	c.providers = []MetadataProvider{&posterFixture{}}
	h, token := handlerFor(t, c, "video")
	route := "/api/v1/posters/media/" + m.ID
	r := uploadPoster(h, route, token, testPicture())
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if getItems(t, c)[0].PosterURL == "" {
		t.Fatal("local card missing selected image")
	}
	w, e := c.work(ctx, work)
	if e != nil || !w.Poster || w.PosterURL == "" {
		t.Fatal(w, e)
	}
	chosen := call(t, h, "GET", w.PosterURL, "", nil)
	if chosen.Code != 200 || chosen.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal(chosen.Code)
	}
	if _, _, e = image.DecodeConfig(bytes.NewReader(chosen.Body.Bytes())); e != nil {
		t.Fatal(e)
	}
	// A provider refresh, filesystem scan and cache cleanup must not erase the choice.
	md.Title = "Updated"
	if _, e = c.rememberWork(ctx, "films", md, true); e != nil {
		t.Fatal(e)
	}
	if _, e = c.scan(ctx, l, false); e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec(`DELETE FROM artwork;DELETE FROM provider_cache`); e != nil {
		t.Fatal(e)
	}
	c.close()
	next, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer next.close()
	h, token = handlerFor(t, next, "video")
	if r = call(t, h, "GET", w.PosterURL, "", nil); r.Code != 200 || !bytes.Equal(r.Body.Bytes(), chosen.Body.Bytes()) {
		t.Fatal("custom image lost")
	}
	if r = call(t, h, "DELETE", route, token, nil); r.Code != 204 {
		t.Fatal(r.Body.String())
	}
	if getItems(t, next)[0].PosterURL != "" {
		t.Fatal("reset did not remove override")
	}
	var count int
	_ = next.db.QueryRow(`SELECT count(*) FROM custom_artwork`).Scan(&count)
	if count != 0 {
		t.Fatal("orphaned image remained")
	}
}
func TestPosterProviderSelectionAndGuards(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	c.providers = []MetadataProvider{&posterFixture{}}
	id, e := c.rememberWork(ctx, "films", Metadata{Title: "Example", Provider: "sample", ProviderID: "42"}, true)
	if e != nil {
		t.Fatal(e)
	}
	h, token := handlerFor(t, c, "video")
	route := "/api/v1/posters/works/" + id
	r := call(t, h, "GET", route+"/options", "", nil)
	if r.Code != 200 || !strings.Contains(r.Body.String(), "/choice.jpg") {
		t.Fatal(r.Body.String())
	}
	if r = call(t, h, "POST", route, token, map[string]string{"ref": "https://localhost/secret"}); r.Code != 400 {
		t.Fatal("unlisted image accepted", r.Code)
	}
	if r = call(t, h, "POST", route, token, map[string]string{"ref": "/choice.jpg"}); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	for _, b := range [][]byte{[]byte(`<svg onload="alert(1)"/>`), bytes.Repeat([]byte{1}, (8<<20)+1)} {
		if r = uploadPoster(h, route, token, b); r.Code != 400 {
			t.Fatal("invalid image accepted", r.Code)
		}
	}
	if r = uploadPoster(h, route, "", testPicture()); r.Code != 403 {
		t.Fatal("missing token accepted")
	}
	h2, t2 := handlerFor(t, c, "music")
	if r = uploadPoster(h2, route, t2, testPicture()); r.Code != 404 {
		t.Fatal("disabled module accepted")
	}
	if r = call(t, h, "GET", "/api/v1/discovery/works/"+id+"/poster", "", nil); r.Code != 200 {
		t.Fatal("standard poster route ignored override")
	}
	// The cached-work eviction policy must protect explicitly customized works.
	for n := 0; n < 1003; n++ {
		if _, e = c.rememberWork(ctx, "films", Metadata{Title: "Transient", Provider: "sample", ProviderID: strings.Repeat("x", 1) + string(rune(n+1000))}, false); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = c.work(ctx, id); e != nil {
		t.Fatal("customized work evicted", e)
	}
}
func TestUnidentifiedPosterAndSharedImageCleanup(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "Unknown.mp4", "test")
	if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	m := getItems(t, c)[0]
	h, token := handlerFor(t, c, "video")
	route := "/api/v1/posters/media/" + m.ID
	if r := uploadPoster(h, route, token, testPicture()); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	id, e := c.rememberWork(ctx, "films", Metadata{Title: "Other", Provider: "sample", ProviderID: "2"}, true)
	if e != nil {
		t.Fatal(e)
	}
	other := "/api/v1/posters/works/" + id
	if r := uploadPoster(h, other, token, testPicture()); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	if e = c.saveMetadata(ctx, m, Metadata{Title: "Now identified", Provider: "sample", ProviderID: "3"}); e != nil {
		t.Fatal(e)
	}
	if r := call(t, h, "GET", route, "", nil); r.Code != 200 {
		t.Fatal("identification lost local correction")
	}
	if r := call(t, h, "DELETE", route, token, nil); r.Code != 204 {
		t.Fatal(r.Body.String())
	}
	if r := call(t, h, "GET", other, "", nil); r.Code != 200 {
		t.Fatal("shared image deleted")
	}
}
func TestTMDBPosterOrderingAndFiltering(t *testing.T) {
	p := newTMDBProvider(nil)
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/3/movie/42/images" || r.URL.Query().Get("language") != "" {
			t.Fatal(r.URL.String())
		}
		data := `{"posters":[{"file_path":"/en.jpg","iso_639_1":"en","width":600,"height":900,"vote_count":50},{"file_path":"/none.jpg","iso_639_1":null,"width":600,"height":900},{"file_path":"/fr.jpg","iso_639_1":"fr","width":600,"height":900},{"file_path":"https://bad.example/x.jpg","width":600,"height":900},{"file_path":"/fr.jpg","iso_639_1":"fr","width":600,"height":900}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(data)), Header: http.Header{}}, nil
	})}
	options, e := p.Posters(testContext(context.Background()), "fixture", MetadataSelection{Kind: "films", ID: "42"}, "fr-FR")
	if e != nil || len(options) != 3 || options[0].Ref != "/fr.jpg" || options[1].Ref != "/none.jpg" {
		t.Fatal(options, e)
	}
}
func TestPosterSchemaSevenMigration(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	id, e := c.rememberWork(ctx, "films", Metadata{Title: "Keep", Provider: "sample", ProviderID: "1"}, true)
	if e != nil {
		t.Fatal(e)
	}
	dropAccountsForMigration(t, c)
	if _, e = c.db.Exec(`UPDATE metadata_queue_state SET paused=1;DROP TABLE poster_choices;DROP TABLE custom_artwork;PRAGMA user_version=7`); e != nil {
		t.Fatal(e)
	}
	c.close()
	next, e := openMigrationCatalog(t, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer next.close()
	w, e := next.work(ctx, id)
	if e != nil || w.Metadata.Title != "Keep" || !queueState(t, next).Paused {
		t.Fatal(w, e)
	}
	h, _ := handlerFor(t, next, "video")
	r := call(t, h, "GET", "/api/v1/posters/works/"+id+"/options", "", nil)
	var data map[string]any
	if e = json.Unmarshal(r.Body.Bytes(), &data); e != nil || r.Code != 200 {
		t.Fatal(r.Body.String(), e)
	}
}
