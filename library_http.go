package main

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
)

func sendJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func problem(w http.ResponseWriter, code int, message string) {
	sendJSON(w, code, map[string]string{"error": message})
}
func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	return readJSONLimit(w, r, value, 8192)
}
func readJSONLimit(w http.ResponseWriter, r *http.Request, value any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		problem(w, 400, "Données invalides.")
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		problem(w, 400, "Données invalides.")
		return false
	}
	return true
}

func libraryRoutes(mux *secureMux, c *catalog, modules []module) string {
	token := rand.Text()
	allowed := func(kind string) bool {
		for _, m := range modules {
			for _, k := range m.Kinds {
				if k == kind {
					return true
				}
			}
		}
		return false
	}
	detailListRoutes(mux, c, allowed)
	detailSectionsRoutes(mux, c, allowed)
	detailHeaderRoutes(mux, c, allowed)
	detailExperienceRoutes(mux, c, allowed)
	posterRoutes(mux, c, allowed)
	metadataRoutes(mux, c, allowed)
	musicRoutes(mux, c, allowed)
	discoveryRoutes(mux, c, allowed)
	experienceRoutes(mux, c, allowed)
	acquisitionRoutes(mux, c, allowed)
	mux.HandleFunc("GET /api/v1/library/scan", func(w http.ResponseWriter, r *http.Request) {
		s := c.getScanProgress()
		if !allowed(s.Kind) {
			s = scanProgress{State: "idle"}
		}
		sendJSON(w, 200, s)
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, 200, map[string]any{"version": version, "stage": "local-prototype", "modules": accountModules(r.Context(), modules), "libraryConnected": true, "token": viewer(r.Context()).CSRF, "user": viewer(r.Context()).account})
	})
	mux.HandleFunc("GET /api/v1/library", func(w http.ResponseWriter, r *http.Request) {
		libs, err := c.libraries(r.Context())
		if err != nil {
			problem(w, 500, "Catalogue indisponible.")
			return
		}
		items, err := c.items(r.Context())
		if err != nil {
			problem(w, 500, "Catalogue indisponible.")
			return
		}
		filteredLibs := []library{}
		for _, l := range libs {
			if allowed(l.Kind) {
				filteredLibs = append(filteredLibs, l)
			}
		}
		filteredItems := []media{}
		for _, m := range items {
			if allowed(m.Kind) {
				filteredItems = append(filteredItems, m)
			}
		}
		warnings := c.projectCatalogue(r.Context(), filteredItems)
		sendJSON(w, 200, map[string]any{"libraries": filteredLibs, "items": filteredItems, "catalogueWarnings": warnings})
	})
	mux.HandleFunc("POST /api/v1/libraries", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Kind string `json:"kind"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		if !allowed(input.Kind) {
			problem(w, 400, "Choisissez un type de média actif.")
			return
		}
		l, err := c.scan(r.Context(), library{Name: strings.TrimSpace(input.Name), Path: input.Path, Kind: input.Kind}, true)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		l.MetadataMessage = c.metadataQueuedMessage(r.Context())
		sendJSON(w, 201, l)
	})
	mux.HandleFunc("POST /api/v1/libraries/{id}/scan", func(w http.ResponseWriter, r *http.Request) {
		var l library
		err := c.db.QueryRowContext(r.Context(), `SELECT id,name,path,kind FROM libraries WHERE id=?`, r.PathValue("id")).Scan(&l.ID, &l.Name, &l.Path, &l.Kind)
		if err != nil || !allowed(l.Kind) {
			problem(w, 404, "Dossier inconnu.")
			return
		}
		l, err = c.scan(r.Context(), l, false)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		l.MetadataMessage = c.metadataQueuedMessage(r.Context())
		sendJSON(w, 200, l)
	})
	mux.HandleFunc("GET /api/v1/media/{id}/stream", func(w http.ResponseWriter, r *http.Request) {
		f, kind, err := c.openMedia(r.Context(), r.PathValue("id"))
		if err != nil {
			problem(w, 404, "Fichier indisponible. Vérifiez le dossier puis actualisez-le.")
			return
		}
		defer f.Close()
		if !allowed(kind) {
			http.NotFound(w, r)
			return
		}
		info, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// Explicit types avoid Windows registry MIME mappings varying between machines.
		types := map[string]string{".mp3": "audio/mpeg", ".flac": "audio/flac", ".wav": "audio/wav", ".m4a": "audio/mp4", ".aac": "audio/aac", ".opus": "audio/ogg", ".ogg": "audio/ogg", ".mp4": "video/mp4", ".m4v": "video/mp4", ".mkv": "video/x-matroska", ".webm": "video/webm", ".mov": "video/quicktime"}
		if mime := types[strings.ToLower(filepath.Ext(info.Name()))]; mime != "" {
			w.Header().Set("Content-Type", mime)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.Header().Set("Cache-Control", "private, no-cache")
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
	mux.HandleFunc("PUT /api/v1/media/{id}/progress", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Position  float64 `json:"position"`
			Duration  float64 `json:"duration"`
			Completed bool    `json:"completed"`
			Revision  string  `json:"revision"`
		}
		if !readJSON(w, r, &p) {
			return
		}
		if math.IsNaN(p.Position) || math.IsInf(p.Position, 0) || math.IsNaN(p.Duration) || math.IsInf(p.Duration, 0) || p.Position < 0 || p.Duration <= 0 || p.Position > p.Duration || p.Duration > 31*24*3600 {
			problem(w, 400, "Progression invalide.")
			return
		}
		var kind string
		err := c.db.QueryRowContext(r.Context(), `SELECT l.kind FROM media m JOIN libraries l ON l.id=m.library_id WHERE m.id=?`, r.PathValue("id")).Scan(&kind)
		if err != nil || !allowed(kind) {
			problem(w, 404, "Vidéo inconnue.")
			return
		}
		result, err := c.db.ExecContext(r.Context(), `INSERT INTO user_progress(user_id,media_id,revision,position,duration,completed,last_played) SELECT ?,id,revision,?,?,?,unixepoch() FROM media WHERE id=? AND revision=? AND available=1 ON CONFLICT(user_id,media_id) DO UPDATE SET revision=excluded.revision,position=excluded.position,duration=excluded.duration,completed=excluded.completed,last_played=excluded.last_played`, userID(r.Context()), p.Position, p.Duration, p.Completed, r.PathValue("id"), p.Revision)
		if err != nil {
			problem(w, 500, "Progression non enregistrée.")
			return
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			problem(w, 409, "Le fichier a changé. Actualisez la bibliothèque.")
			return
		}
		w.WriteHeader(204)
	})
	return token
}
