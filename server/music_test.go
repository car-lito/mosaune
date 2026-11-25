package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPlaylistsPersistenceAndAtomicEdits(t *testing.T) {
	c, dbDir := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "first.mp3", "first")
	putFile(t, dir, "second.flac", "second")
	l, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	ids := []string{items[1].ID, items[0].ID, items[1].ID}
	h, token := handlerFor(t, c, "video,music")
	w := call(t, h, "POST", "/api/v1/music/playlists", token, playlistInput{Name: " Soirée ", TrackIDs: ids})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var p playlist
	if err = json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "Soirée" || len(p.Tracks) != 3 || p.Tracks[0].ID != ids[0] || p.Tracks[2].ID != ids[2] || p.Revision != 1 {
		t.Fatal(p)
	}
	endpoint := "/api/v1/music/playlists/" + p.ID
	for _, bad := range []playlistInput{
		{Name: "changed", Revision: 1, TrackIDs: []string{ids[0], "unknown"}},
		{Name: "  ", Revision: 1, TrackIDs: ids},
		{Name: "bad\nname", Revision: 1, TrackIDs: ids},
		{Name: "changed", Revision: 0, TrackIDs: ids},
		{Name: "too large", Revision: 1, TrackIDs: make([]string, 201)},
	} {
		if response := call(t, h, "PUT", endpoint, token, bad); response.Code != 400 {
			t.Fatal(response.Code, response.Body.String())
		}
	}
	// Wrong revision must not overwrite edits from a different tab.
	w = call(t, h, "PUT", endpoint, token, playlistInput{Name: "stale", TrackIDs: ids, Revision: 7})
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	if err = os.Remove(filepath.Join(dir, "first.mp3")); err != nil {
		t.Fatal(err)
	}
	if _, err = c.scan(testContext(context.Background()), l, false); err != nil {
		t.Fatal(err)
	}
	w = call(t, h, "PUT", endpoint, token, playlistInput{Name: "Voyage", TrackIDs: ids, Revision: 1})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Revision != 2 || len(p.Tracks) != 3 {
		t.Fatal(p)
	}
	c.close()
	reopened, err := openCatalog(dbDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	h, token = handlerFor(t, reopened, "music")
	w = call(t, h, "GET", "/api/v1/music/playlists", "", nil)
	var saved []playlist
	if err = json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || !reflect.DeepEqual(saved[0], p) {
		t.Fatal(saved)
	}
	w = call(t, h, "DELETE", endpoint, token, map[string]int{"revision": 1})
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	w = call(t, h, "DELETE", endpoint, token, map[string]int{"revision": 2})
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, err = os.Stat(filepath.Join(dir, "second.flac")); err != nil {
		t.Fatal("playlist delete touched media", err)
	}
	w = call(t, h, "GET", "/api/v1/music/playlists", "", nil)
	if w.Body.String() != "[]\n" {
		t.Fatal(w.Body.String())
	}
}

func TestMusicModuleAndPlaylistAccess(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "video.mp4", "video")
	if _, err := c.scan(testContext(context.Background()), library{Path: dir, Kind: "films"}, true); err != nil {
		t.Fatal(err)
	}
	video := getItems(t, c)[0]
	h, token := handlerFor(t, c, "music")
	w := call(t, h, "POST", "/api/v1/music/playlists", token, playlistInput{Name: "videos", TrackIDs: []string{video.ID}})
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		path := "/api/v1/music/playlists"
		if method != "POST" {
			path += "/anything"
		}
		w = call(t, h, method, path, "", playlistInput{Name: "no token"})
		if w.Code != 403 {
			t.Fatal(method, w.Code)
		}
	}
	h, token = handlerFor(t, c, "video")
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		path := "/api/v1/music/playlists"
		if method == "PUT" || method == "DELETE" {
			path += "/anything"
		}
		w = call(t, h, method, path, token, playlistInput{Name: "disabled"})
		want := 404
		// Private playlist identifiers are checked before the module handler.
		if method == "PUT" || method == "DELETE" {
			want = 403
		}
		if w.Code != want {
			t.Fatal(method, w.Code)
		}
	}
}

func TestMusicMigrationPreservesExistingCatalog(t *testing.T) {
	c, dir := testCatalog(t)
	dropAccountsForMigration(t, c)
	removeRecognitionForLegacyFixture(t, c)
	if _, err := c.db.Exec(`DROP TABLE album_links; DROP TABLE playlist_art; ALTER TABLE media DROP COLUMN added_at; ALTER TABLE media DROP COLUMN last_played; DROP TABLE works; DROP TABLE playlists; PRAGMA user_version=2; INSERT INTO provider_settings(id,secret) VALUES('example','test-value');`); err != nil {
		t.Fatal(err)
	}
	c.close()
	reopened, err := openMigrationCatalog(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	var schema int
	var value string
	if err = reopened.db.QueryRow(`PRAGMA user_version`).Scan(&schema); err != nil || schema != catalogSchemaVersion {
		t.Fatal(schema, err)
	}
	if err = reopened.db.QueryRow(`SELECT secret FROM provider_settings WHERE id='example'`).Scan(&value); err != nil || value != "test-value" {
		t.Fatal("provider configuration lost", err)
	}
}
