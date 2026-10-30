package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExperienceProgressAndAlbumAssociation(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "first.flac", "immutable bytes")
	putFile(t, dir, "second.flac", "second")
	l, err := c.scan(ctx, library{Kind: "music", Path: dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	if items[0].AddedAt == 0 {
		t.Fatal("missing added date")
	}
	added := items[0].AddedAt
	for i, item := range items {
		if err = c.saveMetadata(ctx, item, Metadata{Title: []string{"Premier", "Second"}[i], Album: "Album", AlbumArtist: "Artiste", Artist: "Invité", Track: i + 1, Disc: 1, Provider: "tags"}); err != nil {
			t.Fatal(err)
		}
	}
	h, token := handlerFor(t, c, "video,music")
	e := getItems(t, c)[0]
	progress := map[string]any{"position": 42, "duration": 180, "completed": false, "revision": e.Revision}
	if r := call(t, h, "PUT", "/api/v1/media/"+e.ID+"/progress", token, progress); r.Code != 204 {
		t.Fatal(r.Code, r.Body.String())
	}
	if current := getItems(t, c)[0]; current.LastPlayed == 0 || current.Position != 42 {
		t.Fatal(current)
	}
	endpoint := "/api/v1/media/" + e.ID + "/state"
	for _, completed := range []bool{true, false} {
		r := call(t, h, "PATCH", endpoint, token, map[string]any{"completed": completed, "revision": e.Revision})
		if r.Code != 204 {
			t.Fatal(r.Code, r.Body.String())
		}
		current := getItems(t, c)[0]
		if current.Position != 0 || current.Completed != completed || current.Duration != 180 {
			t.Fatal(current)
		}
	}
	if r := call(t, h, "PATCH", endpoint, token, map[string]any{"completed": true, "revision": "stale"}); r.Code != 409 {
		t.Fatal(r.Code)
	}
	if r := call(t, h, "PATCH", endpoint, "", map[string]any{"completed": true, "revision": e.Revision}); r.Code != 403 {
		t.Fatal(r.Code)
	}
	id, err := c.rememberWork(ctx, "music", Metadata{Title: "Album", Album: "Album", Artist: "Artiste", Provider: "musicbrainz", ProviderID: "edition", Tracks: []CatalogTrack{{Title: "Premier", Number: 1, Disc: 1}, {Title: "Second", Number: 2, Disc: 1}, {Title: "Absent", Number: 3, Disc: 1}}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if r := call(t, h, "POST", "/api/v1/music/associate", token, map[string]string{"mediaId": e.ID, "workId": id}); r.Code != 204 {
		t.Fatal(r.Code, r.Body.String())
	}
	works, err := c.localWorks(ctx, func(string) bool { return true })
	if err != nil || len(works) != 1 || len(works[0].LocalIDs) != 2 {
		t.Fatal(works, err)
	}
	if _, err = c.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	for _, item := range getItems(t, c) {
		if item.Metadata.Provider != "tags" || item.Metadata.Artist != "Invité" || item.WorkID != id || item.AddedAt != added {
			t.Fatal("lost original metadata / association", item)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "first.flac"))
	if err != nil || string(data) != "immutable bytes" {
		t.Fatal("original changed")
	}
	disabled, otherToken := handlerFor(t, c, "video")
	if r := call(t, disabled, "PATCH", endpoint, otherToken, map[string]any{"completed": true, "revision": e.Revision}); r.Code != 404 {
		t.Fatal(r.Code)
	}
}

func TestPlaylistCoverRevisionAndPersistence(t *testing.T) {
	c, dir := testCatalog(t)
	h, token := handlerFor(t, c, "music")
	r := call(t, h, "POST", "/api/v1/music/playlists", token, playlistInput{Name: "Playlist", TrackIDs: []string{}})
	if r.Code != 201 {
		t.Fatal(r.Code)
	}
	var p playlist
	if err := json.Unmarshal(r.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	upload := func(body []byte, query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PUT", "http://127.0.0.1/api/v1/music/playlists/"+p.ID+"/cover?revision="+query, bytes.NewReader(body))
		req.Header.Set("X-Mosaune-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if r = upload([]byte(`<svg onload="alert(1)"></svg>`), "1"); r.Code != 400 {
		t.Fatal(r.Code)
	}
	if r = upload(testPicture(), "1"); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r = upload(testPicture(), "1"); r.Code != 409 {
		t.Fatal("stale cover accepted", r.Code)
	}
	if r = call(t, h, "PUT", "/api/v1/music/playlists/"+p.ID, token, playlistInput{Name: "Renommée", Revision: 2, TrackIDs: []string{}}); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if err := json.Unmarshal(r.Body.Bytes(), &p); err != nil || p.ImageID == "" || p.Revision != 3 {
		t.Fatal("cover lost on rename", p, err)
	}
	if r = call(t, h, "GET", "/api/v1/music/playlists/"+p.ID+"/cover", "", nil); r.Code != 200 || r.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal(r.Code)
	}
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	h, token = handlerFor(t, next, "music")
	r = call(t, h, "GET", "/api/v1/music/playlists", "", nil)
	if !strings.Contains(r.Body.String(), p.ImageID) {
		t.Fatal("cover lost on restart")
	}
	disabled, _ := handlerFor(t, next, "video")
	if r = call(t, disabled, "GET", "/api/v1/music/playlists/"+p.ID+"/cover", "", nil); r.Code != 404 {
		t.Fatal(r.Code)
	}
}

func TestMusicBrainzContractsAndCache(t *testing.T) {
	c, _ := testCatalog(t)
	p := newMusicBrainzProvider(c.db)
	id := "12345678-1234-1234-1234-123456789abc"
	artist := "aaaaaaaa-1234-1234-1234-123456789abc"
	calls := 0
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Mosaune/") || r.URL.Query().Get("fmt") != "json" {
			t.Fatal("API headers")
		}
		body := `{"count":1,"releases":[{"id":"` + id + `","title":"Album","date":"2026-10-20","country":"BE","artist-credit":[{"name":"Artiste","artist":{"id":"` + artist + `","name":"Artiste"}}]}]}`
		if strings.HasSuffix(r.URL.Path, id) {
			if r.URL.Query().Get("inc") != "recordings+artist-credits+labels+release-groups" {
				t.Fatal("track credits missing")
			}
			body = `{"id":"` + id + `","title":"Album","artist-credit":[{"artist":{"id":"` + artist + `","name":"Artiste"}}],"media":[{"position":1,"tracks":[{"id":"track","position":1,"length":180000,"recording":{"title":"Titre"}}]}]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	for range 2 {
		page, err := p.Discover(testContext(context.Background()), "", DiscoveryQuery{Kind: "music", Query: `Artist "quoted"`, Page: 1})
		if err != nil || len(page.Items) != 1 || page.Items[0].Artist != "Artiste" {
			t.Fatal(page, err)
		}
	}
	if calls != 1 {
		t.Fatal("cache missed")
	}
	m, err := p.Details(testContext(context.Background()), "", MetadataSelection{Kind: "music", ID: id})
	if err != nil || len(m.Tracks) != 1 || m.Tracks[0].Duration != 180 || m.Tracks[0].Artists[0].ID != artist {
		t.Fatal(m, err)
	}
	if _, err = p.Details(testContext(context.Background()), "", MetadataSelection{Kind: "music", ID: "../private"}); err == nil {
		t.Fatal("unsafe ID")
	}
	real := newMusicBrainzProvider(nil)
	req, _ := http.NewRequest("GET", "https://archive.org.evil.test/image.jpg", nil)
	if real.client.CheckRedirect(req, nil) == nil {
		t.Fatal("unsafe cover redirect")
	}
	req, _ = http.NewRequest("GET", "https://archive.org/download/image.jpg", nil)
	if real.client.CheckRedirect(req, nil) != nil {
		t.Fatal("valid cover rejected")
	}
}

func TestTMDBRichDetailsShelfAndProfile(t *testing.T) {
	p := newTMDBProvider(nil)
	t.Cleanup(p.closeExtension)
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/3/movie/42":
			body = `{"id":42,"title":"Film","runtime":120,"vote_average":7.2,"vote_count":10,"release_date":"2026-10-01","credits":{"cast":[{"id":7,"name":"Actrice","character":"Rôle"}]},"videos":{"results":[{"name":"Trailer","site":"YouTube","type":"Trailer","key":"abcdefghijk"},{"name":"Bad","site":"YouTube","type":"Trailer","key":"javascript:"}]}}`
		case "/3/movie/now_playing":
			if r.URL.Query().Get("region") != "BE" {
				t.Fatal("region")
			}
			body = `{"results":[{"id":42,"title":"Film"}]}`
		case "/3/person/7":
			body = `{"name":"Actrice","biography":"Bio","combined_credits":{"cast":[{"id":42,"title":"Film","media_type":"movie"},{"id":42,"title":"Film","media_type":"movie"},{"id":42,"name":"Série","media_type":"tv"}]}}`
		default:
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	m, err := p.Details(testContext(context.Background()), "fixture", MetadataSelection{Kind: "films", ID: "42"})
	if err != nil || m.Runtime != 120 || len(m.Trailers) != 1 || len(m.Contributors) != 1 || m.Contributors[0].ID != "7" {
		t.Fatal(m, err)
	}
	page, err := p.Shelf(testContext(context.Background()), "fixture", "now_playing", "films")
	if err != nil || len(page.Items) != 1 {
		t.Fatal(page, err)
	}
	profile, err := p.Profile(testContext(context.Background()), "fixture", "7", 1)
	if err != nil || len(profile.Items) != 2 {
		t.Fatal(profile, err)
	}
}

func TestExperienceMigrationFromV4PreservesLists(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	id, err := c.rememberWork(ctx, "films", Metadata{Title: "Film sauvegardé", Provider: "tmdb", ProviderID: "42"}, true)
	if err != nil {
		t.Fatal(err)
	}
	dropAccountsForMigration(t, c)
	removeRecognitionForLegacyFixture(t, c)
	if _, err = c.db.Exec(`UPDATE works SET favorite=1,watchlist=1 WHERE id=?; INSERT INTO playlists VALUES('old','Avant refonte',3,'[]'); DROP TABLE album_links; DROP TABLE playlist_art; ALTER TABLE media DROP COLUMN added_at; ALTER TABLE media DROP COLUMN last_played; PRAGMA user_version=4;`, id); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openMigrationCatalog(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	work, err := next.work(ctx, id)
	if err != nil || !work.Favorite || !work.Watchlist || work.Metadata.Title != "Film sauvegardé" {
		t.Fatal("migration lost lists", work, err)
	}
	var name string
	if err = next.db.QueryRow(`SELECT name FROM playlists WHERE id='old'`).Scan(&name); err != nil || name != "Avant refonte" {
		t.Fatal(err, name)
	}
}
