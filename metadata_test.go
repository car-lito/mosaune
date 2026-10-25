package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPicture() []byte {
	im := image.NewRGBA(image.Rect(0, 0, 4, 4))
	im.Set(0, 0, color.RGBA{R: 200, A: 255})
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	return b.Bytes()
}
func testMP3() []byte {
	var frames bytes.Buffer
	frame := func(id string, b []byte) {
		frames.WriteString(id)
		_ = binary.Write(&frames, binary.BigEndian, uint32(len(b)))
		frames.Write([]byte{0, 0})
		frames.Write(b)
	}
	frame("TIT2", append([]byte{3}, []byte("Titre de test")...))
	frame("TPE1", append([]byte{3}, []byte("Artiste")...))
	frame("TALB", append([]byte{3}, []byte("Album")...))
	frame("TRCK", []byte{0, '2', '/', '9'})
	picture := append([]byte{0}, []byte("image/png\x00\x03\x00")...)
	picture = append(picture, testPicture()...)
	frame("APIC", picture)
	size := frames.Len()
	header := []byte{'I', 'D', '3', 3, 0, 0, byte(size>>21) & 127, byte(size>>14) & 127, byte(size>>7) & 127, byte(size) & 127}
	return append(header, frames.Bytes()...)
}
func TestLocalMetadataAndArtwork(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.mp3"), testMP3(), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	item := getItems(t, c)[0]
	if item.Metadata == nil || item.Metadata.ImageID == "" || item.Metadata.Title != "Titre de test" || item.Metadata.Artist != "Artiste" || item.Metadata.Album != "Album" || item.Metadata.Track != 2 {
		t.Fatal("missing artwork")
	}
	h, _ := handlerFor(t, c, "music")
	w := call(t, h, "GET", "/api/v1/artwork/"+item.Metadata.ImageID, "", nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal(w.Code)
	}
	if _, _, err := image.DecodeConfig(w.Body); err != nil {
		t.Fatal(err)
	}
	id, err := storeArtwork(testContext(context.Background()), c.db, []byte(`<svg onload="alert(1)"/>`))
	if err != nil || id != "" {
		t.Fatal("active image accepted")
	}

}
func TestFilenameIdentification(t *testing.T) {
	for _, tc := range []struct {
		file, kind, title, series  string
		year, season, episode, end int
	}{
		{"Marvels.Cloak.and.Dagger.S01E02.Suicide.Sprints.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb.mkv", "series", "Suicide Sprints", "Marvels Cloak and Dagger", 0, 1, 2, 0},
		{"Show/Season 01/S01E01.mkv", "series", "Épisode 1", "Show", 0, 1, 1, 0},
		{"Show.S00E02.Special.mkv", "series", "Special", "Show", 0, 0, 2, 0},
		{"Show.2x03.Title.mp4", "series", "Title", "Show", 0, 2, 3, 0},
		{"Show.S01E01E02.Double.mkv", "series", "Double", "Show", 0, 1, 1, 2},
		{"Iron.Man.2008.1080p.mkv", "films", "Iron Man", "", 2008, 0, 0, 0},
	} {
		reader, err := newMediaRecognizer(context.Background(), tc.kind)
		if err != nil {
			t.Fatal(err)
		}
		items, err := reader.recognize(context.Background(), []string{tc.file})
		reader.close()
		if err != nil {
			t.Fatal(err)
		}
		m := items[0]
		if m.Title != tc.title || m.Series != tc.series || m.Year != tc.year || m.Season != tc.season || m.Episode != tc.episode || m.EpisodeEnd != tc.end {
			t.Errorf("%s: %+v", tc.file, m)
		}
	}
	q := MetadataQuery{Title: "Marvels Cloak and Dagger"}
	candidate := MetadataCandidate{ID: "1", Title: "Marvel's Cloak & Dagger"}
	if _, ok := exactCandidate(q, []MetadataCandidate{candidate}); !ok {
		t.Fatal("punctuation prevented exact match")
	}
	if _, ok := exactCandidate(q, []MetadataCandidate{candidate, candidate}); ok {
		t.Fatal("ambiguous match selected")
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTMDBContractCacheAndErrors(t *testing.T) {
	c, _ := testCatalog(t)
	p := newTMDBProvider(c.db)
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing auth")
		}
		body := `{"results":[{"id":8,"name":"Série","original_name":"Show","first_air_date":"2018-01-01"}]}`
		code := 200
		switch r.URL.Path {
		case "/3/tv/8":
			body = `{"id":8,"name":"Série","overview":"Résumé","credits":{"cast":[{"name":"Actrice"}]}}`
		case "/3/tv/8/season/1/episode/2":
			body = `{"name":"Deuxième épisode","overview":"Résumé épisode"}`
		}
		if r.URL.Query().Get("query") == "rate" {
			code = 429
		}
		if r.URL.Query().Get("query") == "bad-key" {
			code = 401
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	ctx := testContext(context.Background())
	results, err := p.Search(ctx, "test-token", MetadataQuery{Kind: "series", Title: "Show"})
	if err != nil || len(results) != 1 || results[0].ID != "8" {
		t.Fatal(results, err)
	}
	_, err = p.Search(ctx, "test-token", MetadataQuery{Kind: "series", Title: "Show"})
	if err != nil || calls != 1 {
		t.Fatal("cache missed", err, calls)
	}
	md, err := p.Details(ctx, "test-token", MetadataSelection{Kind: "series", ID: "8", Season: 1, Episode: 2})
	if err != nil || md.Title != "Deuxième épisode" || md.Series != "Série" || md.Cast[0] != "Actrice" {
		t.Fatal(md, err)
	}
	for _, q := range []string{"rate", "bad-key"} {
		if _, err = p.Search(ctx, "test-token", MetadataQuery{Title: q}); err == nil || strings.Contains(err.Error(), "test-token") {
			t.Fatal("unsafe error", err)
		}
	}
	if _, err = p.Artwork(ctx, "test-token", "https://localhost/private.jpg"); err == nil {
		t.Fatal("arbitrary image URL accepted")
	}
	if _, err = p.Details(ctx, "test-token", MetadataSelection{ID: "../private"}); err == nil {
		t.Fatal("path injection accepted")
	}
}

type sampleProvider struct{ searches int }

func (p *sampleProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "sample", Name: "Sample provider", ContractVersion: 1, Kinds: []string{"films"}}
}
func (p *sampleProvider) Search(context.Context, string, MetadataQuery) ([]MetadataCandidate, error) {
	p.searches++
	return []MetadataCandidate{{ID: "opaque-external-id", Title: "Example", Year: 2020}}, nil
}
func (p *sampleProvider) Details(_ context.Context, _ string, s MetadataSelection) (Metadata, error) {
	return Metadata{Title: "Titre fourni", Provider: "sample", ProviderID: s.ID, MatchState: "matched", Origins: map[string]string{"title": "sample"}}, nil
}
func (p *sampleProvider) Artwork(context.Context, string, string) ([]byte, error) { return nil, nil }
func TestProviderReplacementAndManualChoice(t *testing.T) {
	c, _ := testCatalog(t)
	p := &sampleProvider{}
	c.providers = []MetadataProvider{p}
	dir := t.TempDir()
	putFile(t, dir, "Example.2020.mp4", "data")
	l, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	item := getItems(t, c)[0]
	h, token := handlerFor(t, c, "video")
	secret := "fake-secret-for-unit-test-only"
	if w := call(t, h, "PUT", "/api/v1/metadata/providers/sample", token, map[string]string{"secret": secret}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := call(t, h, "GET", "/api/v1/metadata/providers", "", nil)
	if strings.Contains(w.Body.String(), secret) {
		t.Fatal("secret exposed")
	}
	c.metadataStep(testContext(context.Background()), func(string) bool { return true })
	if getItems(t, c)[0].Metadata.ProviderID == "" || p.searches != 1 {
		t.Fatal("automatic identification", p.searches)
	}
	w = call(t, h, "PUT", "/api/v1/media/"+item.ID+"/identify", token, map[string]string{"provider": "sample", "id": "manual-id"})
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, err = c.scan(testContext(context.Background()), l, false); err != nil {
		t.Fatal(err)
	}
	c.metadataStep(testContext(context.Background()), func(string) bool { return true })
	md := getItems(t, c)[0].Metadata
	if !md.Manual || md.ProviderID != "manual-id" {
		t.Fatal("manual choice overwritten", md)
	}
	if w = call(t, h, "DELETE", "/api/v1/metadata/providers/sample", token, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if getItems(t, c)[0].Metadata.ProviderID != "manual-id" {
		t.Fatal("disconnect removed metadata")
	}
}

func TestMetadataMigrationPreservesProgress(t *testing.T) {
	c, dir := testCatalog(t)
	mediaDir := t.TempDir()
	putFile(t, mediaDir, "Movie.mp4", "data")
	_, err := c.scan(testContext(context.Background()), library{Path: mediaDir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	id := getItems(t, c)[0].ID
	dropAccountsForMigration(t, c)
	removeRecognitionForLegacyFixture(t, c)
	_, err = c.db.Exec(`UPDATE media SET position=42,duration=100 WHERE id=?; DROP TABLE album_links; DROP TABLE playlist_art; ALTER TABLE media DROP COLUMN added_at; ALTER TABLE media DROP COLUMN last_played; DROP TABLE works; DROP TABLE playlists; DROP TABLE metadata; DROP TABLE artwork; DROP TABLE provider_cache; DROP TABLE provider_settings; PRAGMA user_version=1;`, id)
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openMigrationCatalog(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	if items := getItems(t, next); len(items) != 1 || items[0].Position != 42 || items[0].ID != id {
		t.Fatal("migration lost state", items)
	}
}

func TestProvidedMedia(t *testing.T) {
	dir := os.Getenv("MOSAUNE_TEST_MEDIA_DIR")
	if dir == "" {
		t.Skip("set MOSAUNE_TEST_MEDIA_DIR for the user-provided read-only fixtures")
	}
	c, _ := testCatalog(t)
	for _, pair := range [][2]string{{"musiques", "music"}, {"series", "series"}} {
		l, err := c.scan(testContext(context.Background()), library{Path: filepath.Join(dir, pair[0]), Kind: pair[1]}, true)
		if err != nil {
			t.Fatal(err)
		}
		if l.Count < 2 {
			t.Errorf("expected at least two provided %s fixtures, got %d", pair[1], l.Count)
		}
	}
	for _, item := range getItems(t, c) {
		if item.Metadata == nil || item.Metadata.Title == "" {
			t.Fatal("missing local metadata")
		}
	}
}
