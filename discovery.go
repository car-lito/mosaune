package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Optional capability: metadata-only providers keep their version 1 contract.
// IDs and artwork references come from trusted providers, never from the client.
type DiscoveryProvider interface {
	Discover(context.Context, string, DiscoveryQuery) (DiscoveryPage, error)
}
type DiscoveryQuery struct {
	Kind, Query, Seed string
	Page              int
}
type DiscoveryPage struct {
	Items       []Metadata
	Page, Pages int
}
type discoveryWork struct {
	PosterURL string   `json:"posterUrl,omitempty"`
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Metadata  Metadata `json:"metadata"`
	Favorite  bool     `json:"favorite"`
	Watchlist bool     `json:"watchlist"`
	LocalIDs  []string `json:"localIds"`
	Watched   bool     `json:"watched"`
	Poster    bool     `json:"poster"`
	ref       string
	detailed  bool
}

func migrateDiscovery(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 4 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE works (
 id TEXT PRIMARY KEY, kind TEXT NOT NULL, data TEXT NOT NULL, artwork_ref TEXT NOT NULL DEFAULT '',
 detailed INTEGER NOT NULL DEFAULT 0, favorite INTEGER NOT NULL DEFAULT 0,
 watchlist INTEGER NOT NULL DEFAULT 0, touched INTEGER NOT NULL);
 PRAGMA user_version=4;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func workID(kind, provider, id string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + provider + "\x00" + id))
	return hex.EncodeToString(sum[:])
}
func contentKind(kind string) bool { return videoKind(kind) || kind == "music" }
func videoKind(kind string) bool   { return kind == "films" || kind == "series" }

func (c *catalog) rememberWork(ctx context.Context, kind string, m Metadata, detailed bool) (string, error) {
	if !contentKind(kind) || m.Provider == "" || m.ProviderID == "" || len(m.ProviderID) > 200 || m.Title == "" {
		return "", errors.New("Œuvre invalide.")
	}
	id := workID(kind, m.Provider, m.ProviderID)
	// A series is one work; episode-specific text is never promoted to its synopsis.
	if kind == "series" && m.Episode > 0 {
		m.Title = m.Series
		m.Overview = ""
		m.Cast = nil
		m.Genres = nil
	}
	m.Season = 0
	m.Episode = 0
	m.EpisodeEnd = 0
	m.Title = truncate(m.Title, 500)
	m.Overview = truncate(m.Overview, 8000)
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	unlock, e := providerCommit(ctx)
	if e != nil {
		return "", e
	}
	defer unlock()
	c.discoveryMu.Lock()
	defer c.discoveryMu.Unlock()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// Keep saved works indefinitely, cap transient catalog entries at 1000.
	_, err = tx.ExecContext(ctx, `INSERT INTO works(id,kind,data,artwork_ref,detailed,touched) VALUES(?,?,?,?,?,?)
 ON CONFLICT(id) DO UPDATE SET data=CASE WHEN excluded.detailed=1 THEN excluded.data ELSE works.data END,
 artwork_ref=CASE WHEN excluded.artwork_ref<>'' THEN excluded.artwork_ref ELSE works.artwork_ref END,
 detailed=MAX(works.detailed,excluded.detailed),touched=excluded.touched`, id, kind, string(data), m.ArtworkRef, detailed, time.Now().UnixNano())
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM works WHERE id IN (SELECT id FROM works WHERE favorite=0 AND watchlist=0 AND id NOT IN (SELECT work_id FROM user_works WHERE favorite=1 OR watchlist=1) AND id NOT IN (SELECT work_id FROM album_links) AND id NOT IN (SELECT work_id FROM acquisition_requests) AND ('work:'||id) NOT IN (SELECT target FROM poster_choices) ORDER BY touched DESC LIMIT -1 OFFSET 1000)`)
	if err != nil {
		return "", err
	}
	return id, tx.Commit()
}
func (c *catalog) work(ctx context.Context, id string) (discoveryWork, error) {
	w := discoveryWork{ID: id, LocalIDs: []string{}}
	var data string
	err := c.db.QueryRowContext(ctx, `SELECT kind,data,artwork_ref,detailed,favorite,watchlist FROM works WHERE id=?`, id).Scan(&w.Kind, &data, &w.ref, &w.detailed, &w.Favorite, &w.Watchlist)
	if err == nil {
		err = json.Unmarshal([]byte(data), &w.Metadata)
	}
	if u := viewer(ctx); u != nil {
		w.Favorite = false
		w.Watchlist = false
		e := c.db.QueryRowContext(ctx, `SELECT favorite,watchlist FROM user_works WHERE user_id=? AND work_id=?`, u.ID, id).Scan(&w.Favorite, &w.Watchlist)
		if e != nil && e != sql.ErrNoRows {
			return w, e
		}
		if !canKind(ctx, w.Kind) {
			return w, sql.ErrNoRows
		}
	}
	w.PosterURL = c.posterChoiceURL(ctx, "work:"+id, "works", id)
	w.Poster = w.PosterURL != "" || w.ref != "" || w.Metadata.ImageID != ""
	return w, err
}
func attachAvailability(w *discoveryWork, items []media) {
	for _, m := range items {
		if (w.Kind == "music" && m.WorkID == w.ID) || (m.Metadata != nil && m.Kind == w.Kind && m.Metadata.Provider == w.Metadata.Provider && m.Metadata.ProviderID == w.Metadata.ProviderID) {
			w.LocalIDs = append(w.LocalIDs, m.ID)
			w.Watched = w.Watched || m.Completed
		}
	}
}
func (c *catalog) localWorks(ctx context.Context, allowed func(string) bool) ([]discoveryWork, error) {
	items, err := c.items(ctx)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, m := range items {
		if !allowed(m.Kind) || !contentKind(m.Kind) || m.Metadata == nil || m.Metadata.ProviderID == "" {
			continue
		}
		id := workID(m.Kind, m.Metadata.Provider, m.Metadata.ProviderID)
		if ids[id] {
			continue
		}
		if _, err = c.rememberWork(ctx, m.Kind, *m.Metadata, false); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	query := `SELECT id FROM works WHERE favorite=1 OR watchlist=1 OR id IN (SELECT work_id FROM album_links) ORDER BY touched DESC`
	args := []any{}
	if userID(ctx) != "" {
		query = `SELECT id FROM works WHERE id IN (SELECT work_id FROM user_works WHERE user_id=? AND (favorite=1 OR watchlist=1)) ORDER BY touched DESC`
		args = append(args, userID(ctx))
		for _, m := range items {
			if m.WorkID != "" {
				ids[m.WorkID] = true
			}
		}
	}
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	saved := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		saved = append(saved, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []discoveryWork{}
	ordered := saved
	for _, m := range items {
		if m.Metadata == nil {
			continue
		}
		id := workID(m.Kind, m.Metadata.Provider, m.Metadata.ProviderID)
		if ids[id] {
			ordered = append(ordered, id)
			delete(ids, id)
		}
	}
	for id := range ids {
		ordered = append(ordered, id)
	}
	seen := map[string]bool{}
	for _, id := range ordered {
		if seen[id] {
			continue
		}
		seen[id] = true
		w, e := c.work(ctx, id)
		if e != nil {
			continue
		}
		if !allowed(w.Kind) {
			continue
		}
		attachAvailability(&w, items)
		result = append(result, w)
	}
	return result, nil
}

func discoveryRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !allowed("films") && !allowed("series") && !allowed("music") {
				problem(w, 404, "Le module vidéo est désactivé.")
				return
			}
			next(w, r)
		}
	}
	resolve := func(w http.ResponseWriter, r *http.Request) (discoveryWork, bool) {
		item, err := c.work(r.Context(), r.PathValue("id"))
		if err != nil || !allowed(item.Kind) {
			problem(w, 404, "Œuvre introuvable. Relancez la recherche.")
			return item, false
		}
		return item, true
	}
	mux.HandleFunc("GET /api/v1/discovery/library", guard(func(w http.ResponseWriter, r *http.Request) {
		result, err := c.localWorks(r.Context(), allowed)
		if err != nil {
			problem(w, 500, "Listes indisponibles.")
			return
		}
		sendJSON(w, 200, result)
	}))
	mux.HandleFunc("POST /api/v1/discovery/browse", guard(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Provider string `json:"provider"`
			Kind     string `json:"kind"`
			Query    string `json:"query"`
			Seed     string `json:"seed"`
			Page     int    `json:"page"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		input.Query = strings.TrimSpace(input.Query)
		if !contentKind(input.Kind) || (!allowed(input.Kind) || !canKind(r.Context(), input.Kind)) || input.Page < 1 || input.Page > 50 || (input.Seed == "" && (len(input.Query) < 2 || len(input.Query) > 250)) || (input.Seed != "" && input.Query != "") {
			problem(w, 400, "Recherche invalide (titre de 2 à 250 caractères, page de 1 à 50).")
			return
		}
		p := c.provider(input.Provider)
		if p == nil || !supportsProvider(p, input.Kind) {
			problem(w, 400, "Fournisseur incompatible.")
			return
		}
		dp, ok := p.(DiscoveryProvider)
		if !ok {
			problem(w, 400, "Ce fournisseur ne propose pas la découverte.")
			return
		}
		q := DiscoveryQuery{Kind: input.Kind, Query: input.Query, Page: input.Page}
		reason := "Résultats de recherche"
		if input.Seed != "" {
			seed, err := c.work(r.Context(), input.Seed)
			if err != nil || seed.Kind != input.Kind || seed.Metadata.Provider != input.Provider {
				problem(w, 400, "Point de départ invalide.")
				return
			}
			q.Seed = seed.Metadata.ProviderID
			reason = "À partir de « " + seed.Metadata.Title + " »"
			if seed.Favorite {
				reason = "Parce que vous aimez « " + seed.Metadata.Title + " »"
			} else {
				items, e := c.items(r.Context())
				if e != nil {
					problem(w, 500, "Bibliothèque indisponible.")
					return
				}
				attachAvailability(&seed, items)
				if seed.Watched {
					reason = "Parce que vous avez regardé « " + seed.Metadata.Title + " »"
				}
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		ctx, done, e := beginProvider(ctx, p)
		if e != nil {
			problem(w, 409, e.Error())
			return
		}
		defer done()
		page, err := dp.Discover(ctx, c.credential(ctx, input.Provider), q)
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		items, err := c.items(ctx)
		if err != nil {
			problem(w, 500, "Bibliothèque indisponible.")
			return
		}
		result := []discoveryWork{}
		seen := map[string]bool{}
		for _, md := range page.Items {
			if len(result) >= 20 {
				break
			}
			md.Provider = input.Provider
			md.ImageID = ""
			id, e := c.rememberWork(ctx, input.Kind, md, false)
			if e != nil {
				problem(w, 500, "Résultats non enregistrés.")
				return
			}
			if seen[id] || id == input.Seed {
				continue
			}
			seen[id] = true
			item, e := c.work(ctx, id)
			if e != nil {
				problem(w, 500, "Résultats indisponibles.")
				return
			}
			attachAvailability(&item, items)
			result = append(result, item)
		}
		pages := max(1, min(50, page.Pages))
		sendJSON(w, 200, map[string]any{"items": result, "page": input.Page, "pages": pages, "reason": reason})
	}))
	mux.HandleFunc("PATCH /api/v1/discovery/works/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		item, ok := resolve(w, r)
		if !ok {
			return
		}
		var input struct {
			Favorite  *bool `json:"favorite"`
			Watchlist *bool `json:"watchlist"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		if input.Favorite == nil && input.Watchlist == nil {
			problem(w, 400, "Choisissez une liste.")
			return
		}
		c.discoveryMu.Lock()
		// Patch independent fields atomically: toggling one list cannot overwrite the other.
		_, err := c.db.ExecContext(r.Context(), `INSERT INTO user_works(user_id,work_id,favorite,watchlist) VALUES(?,?,COALESCE(?,0),COALESCE(?,0)) ON CONFLICT(user_id,work_id) DO UPDATE SET favorite=COALESCE(?,favorite),watchlist=COALESCE(?,watchlist)`, userID(r.Context()), item.ID, input.Favorite, input.Watchlist, input.Favorite, input.Watchlist)
		c.discoveryMu.Unlock()
		if err != nil {
			problem(w, 500, "Liste non enregistrée.")
			return
		}
		item, err = c.work(r.Context(), item.ID)
		if err != nil {
			problem(w, 404, "Œuvre introuvable.")
			return
		}
		items, err := c.items(r.Context())
		if err != nil {
			problem(w, 500, "Bibliothèque indisponible.")
			return
		}
		attachAvailability(&item, items)
		sendJSON(w, 200, item)
	}))
	mux.HandleFunc("GET /api/v1/discovery/works/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		item, ok := resolve(w, r)
		if !ok {
			return
		}
		warning := ""
		if !item.detailed || ((item.Metadata.Provider == "tmdb" || item.Metadata.Provider == "musicbrainz") && item.Metadata.DetailVersion < 2) {
			p := c.provider(item.Metadata.Provider)
			if p != nil && providerEnabled(p) && supportsProvider(p, item.Kind) {
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				defer cancel()
				ctx, done, e := beginProvider(ctx, p)
				if e != nil {
					problem(w, 409, e.Error())
					return
				}
				defer done()
				md, err := p.Details(ctx, c.credential(ctx, p.Info().ID), MetadataSelection{Kind: item.Kind, ID: item.Metadata.ProviderID})
				if err != nil {
					warning = err.Error()
				} else {
					md.Provider = item.Metadata.Provider
					md.ProviderID = item.Metadata.ProviderID
					md.ImageID = item.Metadata.ImageID
					if _, err = c.rememberWork(ctx, item.Kind, md, true); err != nil {
						problem(w, 500, "Fiche non enregistrée.")
						return
					}
					item, err = c.work(ctx, item.ID)
					if err != nil {
						problem(w, 500, "Fiche indisponible.")
						return
					}
				}
			} else {
				warning = "Fournisseur indisponible. Les informations déjà enregistrées restent accessibles."
			}
		}
		items, err := c.items(r.Context())
		if err != nil {
			problem(w, 500, "Bibliothèque indisponible.")
			return
		}
		attachAvailability(&item, items)
		sendJSON(w, 200, map[string]any{"work": item, "warning": warning})
	}))
	mux.HandleFunc("GET /api/v1/discovery/works/{id}/poster", guard(func(w http.ResponseWriter, r *http.Request) {
		// Serialize image decoding to bound peak RAM; queued requests can be cancelled.
		item, ok := resolve(w, r)
		if !ok {
			return
		}
		var data []byte
		if r.URL.Query().Get("automatic") != "1" {
			_ = c.db.QueryRowContext(r.Context(), `SELECT a.data FROM custom_artwork a JOIN poster_choices p ON p.image_id=a.id WHERE p.target=?`, "work:"+item.ID).Scan(&data)
		}
		if len(data) == 0 && item.Metadata.ImageID != "" {
			_ = c.db.QueryRowContext(r.Context(), "SELECT data FROM artwork WHERE id=?", item.Metadata.ImageID).Scan(&data)
		}
		if len(data) == 0 && item.ref != "" {
			if p := c.provider(item.Metadata.Provider); p != nil && supportsProvider(p, item.Kind) {
				data, _ = c.visualImage(r.Context(), p, item.ref)
			}
		}

		if len(data) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, no-cache")
		http.ServeContent(w, r, item.ID+".jpg", time.Time{}, bytes.NewReader(data))
	}))
}
