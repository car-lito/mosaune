package main

import (
	"context"
	"encoding/json"
	"io"
	video "mosaune/extensionkit/capabilities/video/v1"
	"net/http"
	"strings"
	"testing"
)

func TestTMDBExtensionRichDetails(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer fixture-only" {
			t.Error("credential not set by host")
		}
		body := `{"id":42,"title":"Le Voyage","name":"La Série","release_date":"2024-02-01","overview":"Résumé","poster_path":"/poster.jpg","runtime":123,"tagline":"Une aventure","vote_average":7.2,"vote_count":52,"status":"Released","genres":[{"name":"Aventure"}],"production_countries":[{"name":"France"}],"spoken_languages":[{"english_name":"French"}],"credits":{"cast":[{"id":5,"name":"Actrice","character":"Rôle","profile_path":"/portrait.jpg"}]},"seasons":[{"season_number":1,"name":"Saison 1","episode_count":8,"poster_path":"/season.jpg"}],"videos":{"results":[{"name":"Original trailer","site":"YouTube","type":"Trailer","key":"abcdefghijk","iso_639_1":"en"},{"name":"Bande-annonce VF","site":"YouTube","type":"Trailer","key":"ABCDEFGHIJK","iso_639_1":"fr","official":true}]}}`
		if strings.Contains(r.URL.Path, "/episode/") {
			body = `{"name":"Le deuxième épisode","overview":"Résumé épisode"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	for _, s := range []MetadataSelection{{Kind: "films", ID: "42"}, {Kind: "series", ID: "42"}, {Kind: "series", ID: "42", Season: 1, Episode: 2}} {
		got, err := p.Details(ctx, "fixture-only", s)
		if err != nil {
			t.Fatal(err)
		}
		if got.Provider != "tmdb" || got.ProviderID != "42" || got.Runtime != 123 || got.Rating != 7.2 || got.VoteCount != 52 || got.Tagline != "Une aventure" || len(got.Genres) != 1 || got.Genres[0] != "Aventure" || got.DetailVersion != 2 {
			t.Fatalf("rich fields lost: %#v", got)
		}
		if s.Episode > 0 {
			if got.Title != "Le deuxième épisode" || got.Overview != "Résumé épisode" || got.Season != 1 || got.Episode != 2 {
				t.Fatalf("episode fields: %#v", got)
			}
		} else if got.Title != "Le Voyage" || got.Year != 2024 {
			t.Fatalf("title/year: %#v", got)
		}

		if !p.extensionStatus()["loaded"] {
			t.Fatal("extension not executed")
		}
		if got.Trailers[0].Language != "fr" || got.Contributors[0].Image != assetURL("tmdb", "/portrait.jpg") || got.Seasons[0].Image != assetURL("tmdb", "/season.jpg") {
			t.Fatal("lost image or trailer")
		}
	}
}
func TestTMDBExtensionDetailsFallbackAndRequiredFailure(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	paths := []string{}
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		code := 200
		body := `{"name":"Série","videos":{"results":[]}}`
		if strings.HasSuffix(r.URL.Path, "/videos") {
			body = `{"results":[{"site":"YouTube","type":"Trailer","key":"abcdefghijk","name":"Trailer","iso_639_1":"en"}]}`
			if r.URL.Query().Get("language") != "" {
				t.Error("fallback language")
			}
		}
		if strings.Contains(r.URL.Path, "/episode/") {
			code = 404
			body = `{}`
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	_ = p.setExtension(ctx, true)
	d, err := p.Details(ctx, "fixture-only", MetadataSelection{Kind: "series", ID: "42"})
	if err != nil || len(d.Trailers) != 1 || d.Trailers[0].Language != "en" || len(paths) != 2 {
		t.Fatal(d, paths, err)
	}
	_, err = p.Details(ctx, "fixture-only", MetadataSelection{Kind: "series", ID: "42", Season: 1, Episode: 2})
	if err == nil || strings.Contains(err.Error(), "fixture-only") {
		t.Fatal("missing episode accepted or secret leaked", err)
	}
	// Optional trailer failure must not discard the otherwise valid rich document.
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		code := 200
		body := `{"title":"Film sans bande-annonce"}`
		if strings.HasSuffix(r.URL.Path, "/videos") {
			code = 503
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	if d, err = p.Details(ctx, "fixture-only", MetadataSelection{Kind: "films", ID: "99"}); err != nil || d.Title != "Film sans bande-annonce" {
		t.Fatal(d, err)
	}
}
func TestTMDBDetailsGrantAndOutputBoundaries(t *testing.T) {
	s := video.Selection{ID: "42", Kind: "series", Season: 1, Episode: 2, Language: "fr-FR"}
	for _, raw := range []string{
		`{"path":"/tv/42","query":{"language":"fr-FR","append_to_response":"credits,videos","include_video_language":"fr,en,null"}}`,
		`{"path":"/tv/42/videos","query":{"language":""}}`,
		`{"path":"/tv/42/season/1/episode/2","query":{"language":"fr-FR"}}`,
	} {
		if _, _, e := tmdbDetailsGrant("get", json.RawMessage(raw), s); e != nil {
			t.Fatal(e)
		}
	}
	for _, raw := range []string{
		`{"path":"/tv/43","query":{"language":"fr-FR"}}`,
		`{"path":"/tv/42/season/1/episode/3","query":{"language":"fr-FR"}}`,
		`{"path":"/tv/42/videos","query":{}}`,
		`{"path":"/tv/42/videos","query":{"language":"","api_key":"injected"}}`,
		`{"path":"https://other.invalid/tv/42","query":{}}`,
	} {
		if _, _, e := tmdbDetailsGrant("get", json.RawMessage(raw), s); e == nil {
			t.Fatal("accepted unauthorized request", raw)
		}
	}
	s = video.Selection{ID: "42", Kind: "films", Language: "fr-FR"}
	good := video.Details{Title: "Film", ArtworkRef: "/poster.jpg"}
	raw, _ := json.Marshal(good)
	if _, e := tmdbExtensionMetadata(raw, s); e != nil {
		t.Fatal(e)
	}
	for _, edit := range []func(*video.Details){
		func(d *video.Details) { d.ArtworkRef = "http://private.invalid/file" },
		func(d *video.Details) { d.Contributors = []video.Contributor{{ID: "1", Image: "/../secret"}} },
		func(d *video.Details) { d.Trailers = []video.Trailer{{URL: "javascript:alert(1)"}} },
		func(d *video.Details) { d.Series = "Not a film" },
		func(d *video.Details) { d.Rating = 20 },
	} {
		bad := good
		edit(&bad)
		raw, _ = json.Marshal(bad)
		if _, e := tmdbExtensionMetadata(raw, s); e == nil {
			t.Fatal("untrusted output accepted", string(raw))
		}
	}
	if _, e := tmdbExtensionMetadata(json.RawMessage(`{"title":"Film","manual":true,"provider":"other"}`), s); e == nil {
		t.Fatal("provenance injection")
	}
}
