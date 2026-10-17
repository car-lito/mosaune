package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

const visualRelease = "11111111-1111-1111-1111-111111111111"
const visualGroup = "22222222-2222-2222-2222-222222222222"
const visualArtist = "33333333-3333-3333-3333-333333333333"

func visualResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestVisualTMDBSeasonsPortraitsAndTrailers(t *testing.T) {
	p := newTMDBProvider(nil)
	fallback := false
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/3/tv/42":
			return visualResponse(200, `{"id":42,"name":"Série","seasons":[{"season_number":1,"name":"Saison 1","poster_path":"/season.jpg","episode_count":8}],"credits":{"cast":[{"id":7,"name":"Actrice","profile_path":"/face.jpg"}]},"videos":{"results":[{"name":"English","key":"abcdefghijk","site":"YouTube","type":"Trailer","iso_639_1":"en","official":true},{"name":"Français","key":"lmnopqrstuv","site":"YouTube","type":"Trailer","iso_639_1":"fr"}]}}`), nil
		case "/3/movie/43":
			return visualResponse(200, `{"id":43,"title":"Sans VF"}`), nil
		case "/3/movie/43/videos":
			fallback = true
			if !r.URL.Query().Has("language") || r.URL.Query().Get("language") != "" {
				t.Fatal("fallback must include all languages")
			}
			return visualResponse(200, `{"results":[{"name":"Autre langue","key":"abcdefghijk","site":"YouTube","type":"Trailer","iso_639_1":"ja"}]}`), nil
		case "/3/person/7":
			return visualResponse(200, `{"name":"Actrice","profile_path":"/face.jpg"}`), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})}
	md, err := p.Details(testContext(context.Background()), "fixture", MetadataSelection{Kind: "series", ID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if len(md.Seasons) != 1 || md.Seasons[0].Episodes != 8 || !strings.Contains(md.Seasons[0].Image, "season.jpg") || !strings.Contains(md.Contributors[0].Image, "face.jpg") {
		t.Fatal(md)
	}
	if len(md.Trailers) != 2 || md.Trailers[0].Language != "fr" || !strings.Contains(md.Trailers[0].Thumbnail, "lmnopqrstuv") {
		t.Fatal(md.Trailers)
	}
	md, err = p.Details(testContext(context.Background()), "fixture", MetadataSelection{Kind: "films", ID: "43"})
	if err != nil || !fallback || len(md.Trailers) != 1 || md.Trailers[0].Language != "ja" {
		t.Fatal(md, err)
	}
	profile, err := p.Profile(testContext(context.Background()), "fixture", "7", 1)
	if err != nil || !strings.Contains(profile.Image, "face.jpg") {
		t.Fatal(profile, err)
	}
	unsafe := selectTrailers([]tmdbVideo{{Site: "YouTube", Type: "Trailer", Key: "../bad"}}, "fr")
	if len(unsafe) != 0 {
		t.Fatal(unsafe)
	}
	dubbed := selectTrailers([]tmdbVideo{{Site: "YouTube", Type: "Trailer", Key: "abcdefghijk", Name: "Bande annonce VOSTFR", Language: "fr", Official: true}, {Site: "YouTube", Type: "Trailer", Key: "lmnopqrstuv", Name: "Bande annonce VF", Language: "fr"}}, "fr")
	if dubbed[0].Name != "Bande annonce VF" {
		t.Fatal("VF should precede subtitled trailers", dubbed)
	}
}

func TestMusicArtworkFallbackPhotoCreditsAndValidation(t *testing.T) {
	c, _ := testCatalog(t)
	p := newMusicBrainzProvider(c.db)
	calls := map[string]int{}
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls[r.URL.Host+r.URL.Path]++
		switch r.URL.Host {
		case "coverartarchive.org":
			if strings.Contains(r.URL.Path, "/release-group/") {
				return visualResponse(200, string(testPicture())), nil
			}
			return visualResponse(404, ""), nil
		case "musicbrainz.org":
			if r.URL.Query().Get("inc") != "url-rels" {
				t.Fatal("missing relationship lookup")
			}
			return visualResponse(200, `{"relations":[{"type":"wikidata","url":{"resource":"https://www.wikidata.org/wiki/Q42"}}]}`), nil
		case "www.wikidata.org":
			if r.URL.Query().Get("property") != "P18" {
				t.Fatal("wrong property")
			}
			return visualResponse(200, `{"claims":{"P18":[{"rank":"normal","mainsnak":{"datavalue":{"value":"Portrait test.jpg"}}}]}}`), nil
		case "commons.wikimedia.org":
			return visualResponse(200, `{"query":{"pages":{"1":{"imageinfo":[{"thumburl":"https://upload.wikimedia.org/test.jpg","extmetadata":{"CommonsMetadataExtension":{"value":1},"Artist":{"value":"<a>Photo &amp; Co</a>"},"LicenseShortName":{"value":"CC BY-SA 4.0"}}}]}}}}`), nil
		case "upload.wikimedia.org":
			return visualResponse(200, string(testPicture())), nil
		default:
			t.Fatal(r.URL)
			return nil, nil
		}
	})}
	for i := 0; i < 2; i++ {
		data, err := c.visualImage(testContext(context.Background()), p, visualRelease+"/"+visualGroup)
		if err != nil || len(data) == 0 {
			t.Fatal(err)
		}
	}
	if calls["coverartarchive.org/release/"+visualRelease+"/front-500"] != 1 || calls["coverartarchive.org/release-group/"+visualGroup+"/front-500"] != 1 {
		t.Fatal("image cache or fallback failed", calls)
	}
	photo, err := p.artistPhoto(testContext(context.Background()), visualArtist)
	if err != nil || photo.Credit != "Photo & Co" || photo.License != "CC BY-SA 4.0" || !strings.Contains(photo.Source, "Portrait%20test.jpg") {
		t.Fatal(photo, err)
	}
	data, err := p.Artwork(testContext(context.Background()), "", "commons:"+photo.File)
	if err != nil || len(data) == 0 {
		t.Fatal(err)
	}
	for _, ref := range []string{"https://localhost/secret", "commons:../../secret.jpg", visualRelease + "/bad"} {
		if _, err = p.Artwork(testContext(context.Background()), "", ref); err == nil {
			t.Fatal("unsafe ref", ref)
		}
	}
	if safeUpload("https://upload.wikimedia.org.evil.test/x") || safeUpload("http://upload.wikimedia.org/x") || safeUpload("https://user@upload.wikimedia.org/x") {
		t.Fatal("unsafe upload accepted")
	}
	real := newMusicBrainzProvider(nil)
	request, _ := http.NewRequest("GET", "http://archive.org/download/a.jpg", nil)
	if err = real.client.CheckRedirect(request, nil); err != nil || request.URL.Scheme != "https" {
		t.Fatal("archive redirect not upgraded", err)
	}
	request, _ = http.NewRequest("GET", "https://127.0.0.1/a.jpg", nil)
	if real.client.CheckRedirect(request, nil) == nil {
		t.Fatal("unsafe redirect")
	}
}

func TestAlbumMatchRoutesAndImageGuards(t *testing.T) {
	c, _ := testCatalog(t)
	p := newMusicBrainzProvider(c.db)
	c.providers = []MetadataProvider{p}
	calls := 0
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path == "/ws/2/artist/" {
			return visualResponse(200, `{"artists":[{"id":"`+visualArtist+`","name":"Artiste"}]}`), nil
		}
		if r.URL.Path == "/ws/2/release/"+visualRelease {
			return visualResponse(200, `{"id":"`+visualRelease+`","title":"Album choisi","media":[{"position":1,"tracks":[{"position":1,"title":"Piste"}]}]}`), nil
		}
		if !strings.Contains(r.URL.Query().Get("query"), `release:"Album" AND artist:"Artiste"`) {
			t.Fatal(r.URL)
		}
		return visualResponse(200, `{"count":1,"releases":[{"id":"`+visualRelease+`","title":"Album","release-group":{"id":"`+visualGroup+`"}}]}`), nil
	})}
	h, token := handlerFor(t, c, "video,music")
	result := call(t, h, "POST", "/api/v1/music/matches", token, map[string]any{"title": "Album", "artist": "Artiste"})
	if result.Code != 200 || !strings.Contains(result.Body.String(), "Album") {
		t.Fatal(result.Code, result.Body)
	}
	if call(t, h, "POST", "/api/v1/music/matches", "", map[string]any{"title": "Album"}).Code != 403 {
		t.Fatal("token bypass")
	}
	if call(t, h, "POST", "/api/v1/music/matches", token, map[string]any{"id": "https://evil.test/release/" + visualRelease}).Code != 400 {
		t.Fatal("invalid id accepted")
	}
	result = call(t, h, "POST", "/api/v1/people/musicbrainz/resolve", token, map[string]any{"name": "Artiste"})
	if result.Code != 200 || !strings.Contains(result.Body.String(), visualArtist) {
		t.Fatal(result.Code, result.Body)
	}
	if call(t, h, "GET", "/api/v1/metadata/images/musicbrainz?ref=https%3A%2F%2Flocalhost", "", nil).Code != 404 {
		t.Fatal("image ref not rejected")
	}
	result = call(t, h, "POST", "/api/v1/music/matches", token, map[string]any{"id": "https://musicbrainz.org/release/" + visualRelease})
	if result.Code != 200 || !strings.Contains(result.Body.String(), "Album choisi") {
		t.Fatal(result.Code, result.Body)
	}
	disabled, otherToken := handlerFor(t, c, "video")
	if call(t, disabled, "POST", "/api/v1/music/matches", otherToken, map[string]any{"title": "Album"}).Code != 404 || call(t, disabled, "GET", "/api/v1/metadata/images/musicbrainz?ref="+visualRelease, "", nil).Code != 404 {
		t.Fatal("disabled module exposed")
	}
	if calls != 3 {
		t.Fatal("invalid request contacted provider", calls)
	}
}

// Optional public integration check: no personal credentials or local catalog.
func TestPublicVisualSources(t *testing.T) {
	if os.Getenv("MOSAUNE_TEST_PUBLIC_VISUAL") != "1" {
		t.Skip("public API check disabled")
	}
	c, _ := testCatalog(t)
	p := newMusicBrainzProvider(c.db)
	ctx := testContext(context.Background())
	artists, err := p.ResolveArtist(ctx, "Orelsan")
	if err != nil {
		t.Fatal(err)
	}
	artist := ""
	for _, a := range artists {
		if strings.EqualFold(a.Name, "Orelsan") {
			artist = a.ID
			break
		}
	}
	if artist == "" {
		t.Fatal("artist missing")
	}
	photo, err := p.artistPhoto(ctx, artist)
	if err != nil {
		t.Fatal("portrait", err)
	}
	if data, err := c.visualImage(ctx, p, "commons:"+photo.File); err != nil || len(data) == 0 {
		t.Fatal("portrait bytes", err)
	}
	releases, err := p.searchReleases(ctx, `release:"Le Chant des sirènes" AND artist:"Orelsan"`, 1)
	if err != nil || len(releases.Items) == 0 {
		t.Fatal("album search", err)
	}
	covered := false
	for _, md := range releases.Items[:min(3, len(releases.Items))] {
		if data, err := c.visualImage(ctx, p, md.ArtworkRef); err == nil && len(data) > 0 {
			covered = true
			break
		}
	}
	if !covered {
		t.Fatal("no cover in first three editions")
	}
	raw, _ := json.Marshal(map[string]any{"portrait": true, "cover": true, "credit": photo.Credit, "source": photo.Source, "editions": len(releases.Items)})
	t.Log(string(raw))
}
