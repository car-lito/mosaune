package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Playlists belong to the optional music module. They reference catalog IDs,
// never file paths; removing a track from a playlist cannot delete a media file.
type playlistTrack struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type playlist struct {
	ImageID  string          `json:"imageId,omitempty"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Revision int             `json:"revision"`
	Tracks   []playlistTrack `json:"tracks"`
}
type playlistInput struct {
	Name     string   `json:"name"`
	TrackIDs []string `json:"trackIds"`
	Revision int      `json:"revision"`
}

func migrateMusic(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= 3 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`CREATE TABLE playlists (id TEXT PRIMARY KEY, name TEXT NOT NULL, revision INTEGER NOT NULL, tracks TEXT NOT NULL); PRAGMA user_version=3;`); err != nil {
		return err
	}
	return tx.Commit()
}

func musicRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !allowed("music") {
				problem(w, 404, "Le module musique est désactivé.")
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("GET /api/v1/music/playlists", guard(func(w http.ResponseWriter, r *http.Request) {
		rows, err := c.db.QueryContext(r.Context(), `SELECT p.id,p.name,p.revision,p.tracks,COALESCE(a.image_id,'') FROM playlists p LEFT JOIN playlist_art a ON a.playlist_id=p.id WHERE p.owner=? ORDER BY p.rowid`, userID(r.Context()))
		if err != nil {
			problem(w, 500, "Playlists indisponibles.")
			return
		}
		defer rows.Close()
		result := []playlist{}
		for rows.Next() {
			var p playlist
			var raw string
			if err = rows.Scan(&p.ID, &p.Name, &p.Revision, &raw, &p.ImageID); err != nil {
				problem(w, 500, "Playlists indisponibles.")
				return
			}
			if err = json.Unmarshal([]byte(raw), &p.Tracks); err != nil {
				problem(w, 500, "Playlist illisible.")
				return
			}
			result = append(result, p)
		}
		if rows.Err() != nil {
			problem(w, 500, "Playlists indisponibles.")
			return
		}
		rows.Close()
		for i := range result {
			filtered := []playlistTrack{}
			for _, track := range result[i].Tracks {
				if c.mediaPermission(r.Context(), track.ID, "view") {
					filtered = append(filtered, track)
				}
			}
			result[i].Tracks = filtered
		}
		sendJSON(w, 200, result)
	}))
	save := guard(func(w http.ResponseWriter, r *http.Request) {
		var input playlistInput
		if !readJSON(w, r, &input) {
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		if input.Name == "" || utf8.RuneCountInString(input.Name) > 80 || strings.IndexFunc(input.Name, unicode.IsControl) >= 0 || len(input.TrackIDs) > 200 {
			problem(w, 400, "Choisissez un nom de 1 à 80 caractères et au maximum 200 morceaux.")
			return
		}
		if u := viewer(r.Context()); u != nil && !u.Admin {
			for _, id := range input.TrackIDs {
				if !c.mediaPermission(r.Context(), id, "view") {
					problem(w, 403, "Morceau non autorisé.")
					return
				}
			}
		}
		create := r.Method == "POST"
		if (create && input.Revision != 0) || (!create && input.Revision < 1) {
			problem(w, 400, "Version de playlist invalide.")
			return
		}
		tx, err := c.db.BeginTx(r.Context(), nil)
		if err != nil {
			problem(w, 500, "Enregistrement impossible.")
			return
		}
		defer tx.Rollback()
		p := playlist{ID: r.PathValue("id"), Name: input.Name, Revision: input.Revision + 1, Tracks: []playlistTrack{}}
		if create {
			var count int
			if err = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM playlists WHERE owner=?`, userID(r.Context())).Scan(&count); err != nil {
				problem(w, 500, "Enregistrement impossible.")
				return
			}
			if count >= 100 {
				problem(w, 400, "La limite de cet atelier est de 100 playlists.")
				return
			}
			p.ID = rand.Text()
		}
		// Retain unavailable tracks and duplicates, including their order. A missing
		// disk should never silently truncate a saved playlist when it is renamed.
		for _, id := range input.TrackIDs {
			var name string
			if err = tx.QueryRowContext(r.Context(), `SELECT m.name FROM media m JOIN libraries l ON l.id=m.library_id WHERE m.id=? AND l.kind='music'`, id).Scan(&name); err != nil {
				if err == sql.ErrNoRows {
					problem(w, 400, "Un morceau ne fait pas partie de la bibliothèque musicale.")
				} else {
					problem(w, 500, "Enregistrement impossible.")
				}
				return
			}
			p.Tracks = append(p.Tracks, playlistTrack{ID: id, Name: name})
		}
		raw, err := json.Marshal(p.Tracks)
		if err != nil {
			problem(w, 500, "Enregistrement impossible.")
			return
		}
		if create {
			_, err = tx.ExecContext(r.Context(), `INSERT INTO playlists(id,name,revision,tracks,owner) VALUES(?,?,?,?,?)`, p.ID, p.Name, p.Revision, string(raw), userID(r.Context()))
		} else {
			var result sql.Result
			result, err = tx.ExecContext(r.Context(), `UPDATE playlists SET name=?,revision=?,tracks=? WHERE id=? AND revision=? AND owner=?`, p.Name, p.Revision, string(raw), p.ID, input.Revision, userID(r.Context()))
			if err == nil {
				n, _ := result.RowsAffected()
				if n != 1 {
					problem(w, 409, "Cette playlist a changé dans une autre fenêtre. Rechargez les playlists avant de réessayer.")
					return
				}
			}
		}
		if err != nil {
			problem(w, 500, "Enregistrement impossible.")
			return
		}
		p.ImageID = playlistImage(r.Context(), tx, p.ID)
		if err = tx.Commit(); err != nil {
			problem(w, 500, "Enregistrement impossible.")
			return
		}
		code := 200
		if create {
			code = 201
		}
		sendJSON(w, code, p)
	})
	mux.HandleFunc("POST /api/v1/music/playlists", save)
	mux.HandleFunc("PUT /api/v1/music/playlists/{id}", save)
	mux.HandleFunc("DELETE /api/v1/music/playlists/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Revision int `json:"revision"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		result, err := c.db.ExecContext(r.Context(), `DELETE FROM playlists WHERE id=? AND revision=? AND owner=?`, r.PathValue("id"), input.Revision, userID(r.Context()))
		if err != nil {
			problem(w, 500, "Suppression impossible.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 409, "Cette playlist a changé. Rechargez les playlists avant de réessayer.")
			return
		}
		w.WriteHeader(204)
	}))
}
