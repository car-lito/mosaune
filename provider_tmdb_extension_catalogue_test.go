package main

import (
	"context"
	"encoding/json"
	"io"
	videocatalog "mosaune/extensionkit/capabilities/videocatalog/v1"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTMDBCatalogueInstalledPackageAllOperations(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	pack, e := packageformat.Parse(providerCatalogueExample, trustedProviderKey(), version)
	if e != nil {
		t.Fatal(e)
	}
	removeTMDBPackageFixture(t, p)
	if e = p.installPackage(ctx, pack, "", false); e != nil {
		t.Fatal(e)
	}
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-only" || r.URL.Host != "api.themoviedb.org" {
			t.Error("credential or host")
		}
		body := `{"total_pages":3,"results":[{"id":42,"title":"Voyage","release_date":"2026-01-01","overview":"Résumé","poster_path":"/poster.jpg"},{"id":42,"title":"Doublon"},{"id":0,"title":"Invalide"}]}`
		if strings.Contains(r.URL.Path, "/person/") {
			body = `{"name":"Actrice","biography":"Biographie","birthday":"2000-01-01","place_of_birth":"Paris","profile_path":"/portrait.jpg","combined_credits":{"cast":[{"id":42,"title":"Voyage","media_type":"movie"},{"id":43,"name":"Série","media_type":"tv","first_air_date":"2025-01-01"},{"id":42,"title":"Doublon","media_type":"movie"},{"id":44,"name":"Personne","media_type":"person"}]}}`
		}
		if strings.HasSuffix(r.URL.Path, "/images") {
			if r.URL.Query().Get("language") != "" {
				t.Error("image language filter")
			}
			body = `{"posters":[{"file_path":"/en.jpg","iso_639_1":"en","width":600,"height":900,"vote_count":90},{"file_path":"/fr.jpg","iso_639_1":"fr","width":600,"height":900,"vote_count":1},{"file_path":"/neutral.png","width":600,"height":900},{"file_path":"https://outside/secret.jpg","width":600,"height":900},{"file_path":"/fr.jpg","width":600,"height":900}]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	for _, q := range []DiscoveryQuery{{Kind: "films", Query: "Voyage", Page: 2}, {Kind: "series", Seed: "42", Page: 1}} {
		r, e := p.Discover(ctx, "fixture-only", q)
		if e != nil || len(r.Items) != 1 || r.Items[0].Provider != "tmdb" || r.Page != q.Page || r.Pages != 3 {
			t.Fatal(r, e)
		}
	}
	for _, s := range []struct{ k, n string }{{"films", "now_playing"}, {"films", "upcoming"}, {"series", "popular"}} {
		r, e := p.Shelf(ctx, "fixture-only", s.n, s.k)
		if e != nil || len(r.Items) != 1 || r.Items[0].ReleaseDate != "2026-01-01" || r.Pages != 1 {
			t.Fatal(r, e)
		}
	}
	profile, e := p.Profile(ctx, "fixture-only", "5", 1)
	if e != nil || profile.Name != "Actrice" || len(profile.Items) != 2 || profile.Items[1].Kind != "series" || profile.Image != assetURL("tmdb", "/portrait.jpg") {
		t.Fatal(profile, e)
	}
	posters, e := p.Posters(ctx, "fixture-only", MetadataSelection{Kind: "films", ID: "42"}, "fr-FR")
	if e != nil || len(posters) != 3 || posters[0].Ref != "/fr.jpg" || posters[1].Ref != "/neutral.png" {
		t.Fatal(posters, e)
	}
	if calls != 7 || !p.extensionStatus()["loaded"] {
		t.Fatal("not all operations reached installed module", calls)
	}
	if _, e = p.Discover(ctx, "fixture-only", DiscoveryQuery{Kind: "films", Query: "Voyage", Page: 2}); e != nil || calls != 7 {
		t.Fatal("cache not reused", e, calls)
	}
	var count int
	if e = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE key LIKE '%fixture-only%' OR CAST(body AS TEXT) LIKE '%fixture-only%'").Scan(&count); e != nil || count != 0 {
		t.Fatal("credential persisted", e, count)
	}
}
func TestTMDBCatalogueOldPackageAndSafeModeNeverFallback(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"results":[]}`))}, nil
	})
	old, e := packageformat.Parse(providerUpdateExample, trustedProviderKey(), version)
	if e != nil {
		t.Fatal(e)
	}
	removeTMDBPackageFixture(t, p)
	if e = p.installPackage(ctx, old, "", false); e != nil {
		t.Fatal(e)
	}
	_ = p.setExtension(ctx, true)
	if _, e = p.Shelf(ctx, "fixture", "upcoming", "films"); e == nil || !strings.Contains(e.Error(), "0.5.0") || calls != 0 {
		t.Fatal("silent old-package fallback", e, calls)
	}
	c.extensionSafeMode = true
	if _, e = p.Discover(ctx, "fixture", DiscoveryQuery{Kind: "films", Query: "Film", Page: 1}); e == nil || calls != 0 {
		t.Fatal("safe-mode fallback", e, calls)
	}
	c.extensionSafeMode = false
	// A disabled module has no native substitute.
	_ = p.setExtension(ctx, false)
	if _, e = p.Shelf(ctx, "fixture", "upcoming", "films"); e == nil || calls != 0 {
		t.Fatal("disabled module bypass", e, calls)
	}
}
func TestTMDBCatalogueGrantAndResultBoundaries(t *testing.T) {
	q := videocatalog.Query{ID: "42", Page: 1, Language: "fr-FR"}
	good := json.RawMessage(`{"path":"/person/42","query":{"language":"fr-FR","append_to_response":"combined_credits"}}`)
	if _, _, e := tmdbCatalogueGrant("get", good, "profile", q); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{`{"path":"/person/43","query":{"language":"fr-FR","append_to_response":"combined_credits"}}`, `{"path":"https://outside/person/42","query":{"language":"fr-FR","append_to_response":"combined_credits"}}`, `{"path":"/person/42","query":{"language":"en","append_to_response":"combined_credits"}}`, `{"path":"/person/42","query":{"language":"fr-FR","append_to_response":"combined_credits","api_key":"secret"}}`} {
		if _, _, e := tmdbCatalogueGrant("get", json.RawMessage(raw), "profile", q); e == nil {
			t.Fatal("grant accepted", raw)
		}
	}
	if _, _, e := tmdbCatalogueGrant("post", good, "profile", q); e == nil {
		t.Fatal("write service")
	}
	result := videocatalog.Result{Person: &videocatalog.Person{ID: "42", Name: "Actrice"}, Page: 1, Pages: 1, Items: []videocatalog.Item{{ID: "3", Kind: "films", Title: "Film", ArtworkRef: "/ok.jpg"}}}
	check := func(r videocatalog.Result, valid bool) {
		raw, _ := json.Marshal(r)
		_, e := decodeTMDBCatalogue(raw, "profile", q)
		if (e == nil) != valid {
			t.Fatal(r, e)
		}
	}
	check(result, true)
	result.Person.ArtworkRef = "https://outside/image.jpg"
	check(result, false)
	result.Person.ArtworkRef = ""
	result.Items[0].ID = "../3"
	check(result, false)
	result.Items[0].ID = "3"
	result.Items = append(result.Items, result.Items[0])
	check(result, false)
	q.Page = 0
	if _, _, e := tmdbCatalogueRequest("profile", q); e == nil {
		t.Fatal("invalid pagination")
	}
	q.Page = 1
	q.ID = "00042"
	if _, _, e := tmdbCatalogueRequest("profile", q); e == nil {
		t.Fatal("noncanonical id")
	}
}
func TestTMDBCatalogueRevocationDropsLateResponse(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	_ = p.setExtension(ctx, true)
	entered := make(chan struct{})
	finish := make(chan struct{})
	var calls atomic.Int32
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		close(entered)
		<-finish
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"name":"Late person","combined_credits":{"cast":[]}}`))}, nil
	})
	done := make(chan error, 1)
	go func() { _, e := p.Profile(ctx, "fixture", "42", 1); done <- e }()
	select {
	case <-entered:
	case <-time.After(20 * time.Second):
		t.Fatal("module did not call service")
	}
	disabled := make(chan error, 1)
	go func() { disabled <- p.setProviderEnabled(ctx, false) }()
	// setProviderEnabled cancels leases before closing the engine; the transport deliberately ignores cancellation.
	deadline := time.Now().Add(5 * time.Second)
	for providerEnabled(p) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if providerEnabled(p) {
		close(finish)
		t.Fatal("provider still enabled")
	}
	close(finish)
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("late response accepted")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("call did not finish")
	}
	if e := <-disabled; e != nil {
		t.Fatal(e)
	}
	var n int
	_ = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE key LIKE '/person/42%'").Scan(&n)
	if n != 0 {
		t.Fatal("revoked response cached")
	}
	if _, e := p.Profile(ctx, "fixture", "42", 1); e == nil || calls.Load() != 1 {
		t.Fatal("revoked call", e)
	}
}
