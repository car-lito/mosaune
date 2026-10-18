package main

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ShelfProvider interface {
	Shelf(context.Context, string, string, string) (DiscoveryPage, error)
}
type ProfileItem struct {
	Kind     string
	Metadata Metadata
}
type PersonProfile struct {
	Image        string          `json:"image,omitempty"`
	ImageSource  string          `json:"imageSource,omitempty"`
	ImageCredit  string          `json:"imageCredit,omitempty"`
	ImageLicense string          `json:"imageLicense,omitempty"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Biography    string          `json:"biography"`
	Detail       string          `json:"detail"`
	Items        []ProfileItem   `json:"-"`
	Works        []discoveryWork `json:"works"`
	Page         int             `json:"page"`
	Pages        int             `json:"pages"`
}
type ProfileProvider interface {
	Profile(context.Context, string, string, int) (PersonProfile, error)
}

func migrateExperience(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 5 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`ALTER TABLE media ADD COLUMN added_at INTEGER NOT NULL DEFAULT 0;
 ALTER TABLE media ADD COLUMN last_played INTEGER NOT NULL DEFAULT 0;
 CREATE TABLE playlist_art (playlist_id TEXT PRIMARY KEY REFERENCES playlists(id) ON DELETE CASCADE,image_id TEXT NOT NULL REFERENCES artwork(id));
 CREATE TABLE album_links (media_id TEXT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,work_id TEXT NOT NULL REFERENCES works(id),revision TEXT NOT NULL);
 UPDATE works SET detailed=0 WHERE json_extract(data,'$.provider')='tmdb';
 PRAGMA user_version=5;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (c *catalog) rememberPage(ctx context.Context, provider, kind string, page DiscoveryPage, allowed func(string) bool) ([]discoveryWork, error) {
	result := []discoveryWork{}
	if !allowed(kind) || !canKind(ctx, kind) {
		return result, nil
	}
	items, err := c.items(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, md := range page.Items {
		if len(result) >= 20 {
			break
		}
		md.Provider = provider
		id, err := c.rememberWork(ctx, kind, md, false)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		work, err := c.work(ctx, id)
		if err != nil {
			return nil, err
		}
		attachAvailability(&work, items)
		result = append(result, work)
	}
	return result, nil
}
func experienceRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	visualRoutes(mux, c, allowed)
	mux.HandleFunc("POST /api/v1/shelves", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Provider string `json:"provider"`
			Kind     string `json:"kind"`
			Shelf    string `json:"shelf"`
			Artist   string `json:"artist"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p := c.provider(input.Provider)
		if p == nil || (!allowed(input.Kind) || !canKind(r.Context(), input.Kind)) || !supportsProvider(p, input.Kind) {
			problem(w, 404, "Source indisponible.")
			return
		}
		sp, ok := p.(ShelfProvider)
		if !ok {
			problem(w, 400, "Sélection non proposée par ce fournisseur.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		// Artist-based upcoming releases require a known provider identity, not a guessed name.
		selector := input.Kind
		if input.Kind == "music" {
			selector = input.Artist
		}
		ctx, done, e := beginProvider(ctx, p)
		if e != nil {
			problem(w, 409, e.Error())
			return
		}
		defer done()
		page, err := sp.Shelf(ctx, c.credential(ctx, p.Info().ID), input.Shelf, selector)
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		works, err := c.rememberPage(ctx, p.Info().ID, input.Kind, page, allowed)
		if err != nil {
			problem(w, 500, "Sélection indisponible.")
			return
		}
		sendJSON(w, 200, works)
	})
	mux.HandleFunc("GET /api/v1/people/{provider}/{id}", func(w http.ResponseWriter, r *http.Request) {
		p := c.provider(r.PathValue("provider"))
		if p == nil {
			problem(w, 404, "Fournisseur inconnu.")
			return
		}
		enabled := false
		for _, k := range p.Info().Kinds {
			enabled = enabled || allowed(k)
		}
		if !enabled {
			problem(w, 404, "Module désactivé.")
			return
		}
		pp, ok := p.(ProfileProvider)
		if !ok {
			problem(w, 404, "Profils indisponibles.")
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		if page < 1 || page > 50 {
			problem(w, 400, "Page invalide.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		ctx, done, e := beginProvider(ctx, p)
		if e != nil {
			problem(w, 409, e.Error())
			return
		}
		defer done()
		profile, err := pp.Profile(ctx, c.credential(ctx, p.Info().ID), r.PathValue("id"), page)
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		profile.Works = []discoveryWork{}
		for _, kind := range []string{"films", "series", "music"} {
			mds := []Metadata{}
			for _, item := range profile.Items {
				if item.Kind == kind {
					mds = append(mds, item.Metadata)
				}
			}
			if len(mds) == 0 {
				continue
			}
			works, e := c.rememberPage(ctx, p.Info().ID, kind, DiscoveryPage{Items: mds}, allowed)
			if e != nil {
				problem(w, 500, "Filmographie ou discographie indisponible.")
				return
			}
			profile.Works = append(profile.Works, works...)
		}
		sendJSON(w, 200, profile)
	})
	mux.HandleFunc("PATCH /api/v1/media/{id}/state", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Completed *bool  `json:"completed"`
			Revision  string `json:"revision"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		if input.Completed == nil {
			problem(w, 400, "État manquant.")
			return
		}
		m, err := c.mediaMetadata(r.Context(), r.PathValue("id"))
		if err != nil || !allowed(m.Kind) {
			problem(w, 404, "Média indisponible.")
			return
		}
		result, err := c.db.ExecContext(r.Context(), `INSERT INTO user_progress(user_id,media_id,revision,position,completed,last_played) SELECT ?,id,revision,0,?,unixepoch() FROM media WHERE id=? AND revision=? AND available=1 ON CONFLICT(user_id,media_id) DO UPDATE SET revision=excluded.revision,position=0,completed=excluded.completed,last_played=excluded.last_played`, userID(r.Context()), *input.Completed, m.ID, input.Revision)
		if err != nil {
			problem(w, 500, "État non enregistré.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 409, "Le fichier a changé. Rechargez la bibliothèque.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("POST /api/v1/music/associate", func(w http.ResponseWriter, r *http.Request) {
		if !allowed("music") {
			problem(w, 404, "Module désactivé.")
			return
		}
		var input struct {
			MediaID string `json:"mediaId"`
			WorkID  string `json:"workId"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		m, err := c.mediaMetadata(r.Context(), input.MediaID)
		if err != nil || m.Kind != "music" || !canLibrary(r.Context(), m.LibraryID, m.Kind, "edit") || m.Metadata.Album == "" {
			problem(w, 400, "Album local introuvable.")
			return
		}
		work, err := c.work(r.Context(), input.WorkID)
		if err != nil || work.Kind != "music" || !work.detailed || len(work.Metadata.Tracks) == 0 {
			problem(w, 400, "Ouvrez d’abord la fiche de cette édition pour vérifier ses pistes.")
			return
		}
		items, err := c.items(r.Context())
		if err != nil {
			problem(w, 500, "Bibliothèque indisponible.")
			return
		}
		tx, err := c.db.BeginTx(r.Context(), nil)
		if err != nil {
			problem(w, 500, "Association impossible.")
			return
		}
		defer tx.Rollback()
		for _, item := range items {
			if canLibrary(r.Context(), item.LibraryID, item.Kind, "edit") && item.Kind == "music" && item.Metadata != nil && strings.EqualFold(item.Metadata.Album, m.Metadata.Album) && strings.EqualFold(albumArtist(*item.Metadata), albumArtist(*m.Metadata)) {
				if _, err = tx.ExecContext(r.Context(), `INSERT INTO album_links VALUES(?,?,?) ON CONFLICT(media_id) DO UPDATE SET work_id=excluded.work_id,revision=excluded.revision`, item.ID, work.ID, item.Revision); err != nil {
					problem(w, 500, "Association impossible.")
					return
				}
			}
		}
		if err = tx.Commit(); err != nil {
			problem(w, 500, "Association impossible.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("PUT /api/v1/music/playlists/{id}/cover", func(w http.ResponseWriter, r *http.Request) {
		if !allowed("music") {
			problem(w, 404, "Module désactivé.")
			return
		}
		revision, err := strconv.Atoi(r.URL.Query().Get("revision"))
		if err != nil || revision < 1 {
			problem(w, 400, "Version de playlist invalide.")
			return
		}
		data, err := readBounded(r.Body, 8<<20)
		if err != nil {
			problem(w, 400, "Image trop volumineuse (8 Mo maximum).")
			return
		}
		c.posterMu.Lock()
		defer c.posterMu.Unlock()
		tx, err := c.db.BeginTx(r.Context(), nil)
		if err != nil {
			problem(w, 500, "Pochette non enregistrée.")
			return
		}
		defer tx.Rollback()
		imageID, err := storeArtwork(r.Context(), tx, data)
		if err != nil || imageID == "" {
			problem(w, 400, "Choisissez une image JPEG ou PNG, au maximum 16 millions de pixels. Le cache peut aussi être plein.")
			return
		}
		result, err := tx.ExecContext(r.Context(), `UPDATE playlists SET revision=revision+1 WHERE id=? AND revision=?`, r.PathValue("id"), revision)
		if err != nil {
			problem(w, 500, "Pochette non enregistrée.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 409, "Playlist modifiée. Rechargez-la avant de réessayer.")
			return
		}
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO playlist_art VALUES(?,?) ON CONFLICT(playlist_id) DO UPDATE SET image_id=excluded.image_id`, r.PathValue("id"), imageID); err != nil {
			problem(w, 500, "Pochette non enregistrée.")
			return
		}
		if err = tx.Commit(); err != nil {
			problem(w, 500, "Pochette non enregistrée.")
			return
		}
		sendJSON(w, 200, map[string]any{"imageId": imageID, "revision": revision + 1})
	})
	mux.HandleFunc("GET /api/v1/music/playlists/{id}/cover", func(w http.ResponseWriter, r *http.Request) {
		if !allowed("music") {
			http.NotFound(w, r)
			return
		}
		var data []byte
		if c.db.QueryRowContext(r.Context(), `SELECT a.data FROM playlist_art p JOIN artwork a ON a.id=p.image_id WHERE p.playlist_id=?`, r.PathValue("id")).Scan(&data) != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private,no-cache")
		http.ServeContent(w, r, "cover.jpg", time.Time{}, bytes.NewReader(data))
	})
}
func albumArtist(m Metadata) string {
	if m.AlbumArtist != "" {
		return m.AlbumArtist
	}
	return m.Artist
}

// Retain saved cover references in API responses after ordinary playlist edits.
func playlistImage(ctx context.Context, db databaseAccess, id string) string {
	var image string
	_ = db.QueryRowContext(ctx, `SELECT image_id FROM playlist_art WHERE playlist_id=?`, id).Scan(&image)
	return image
}
