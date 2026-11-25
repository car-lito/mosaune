package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"
)

// Optional provider capability. Choices belong to Mosaune, never the provider cache.
type PosterProvider interface {
	Posters(context.Context, string, MetadataSelection, string) ([]PosterOption, error)
}
type PosterOption struct {
	Ref      string `json:"ref"`
	Image    string `json:"image"`
	Language string `json:"language"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}
type posterTarget struct{ key, kind, provider, id, scope, routeID string }

func migratePosters(db *sql.DB) error {
	var v int
	if e := db.QueryRow(`PRAGMA user_version`).Scan(&v); e != nil {
		return e
	}
	if v >= 8 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS custom_artwork(id TEXT PRIMARY KEY,data BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS poster_choices(target TEXT PRIMARY KEY,kind TEXT NOT NULL,image_id TEXT NOT NULL REFERENCES custom_artwork(id),source TEXT NOT NULL);
 PRAGMA user_version=8;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (c *catalog) posterTarget(ctx context.Context, scope, id string) (posterTarget, error) {
	t := posterTarget{scope: scope, routeID: id}
	if len(id) > 200 {
		return t, errors.New("Cible invalide.")
	}
	if scope == "works" {
		w, e := c.work(ctx, id)
		if e != nil {
			return t, e
		}
		t.key = "work:" + id
		t.kind = w.Kind
		t.provider = w.Metadata.Provider
		t.id = w.Metadata.ProviderID
	} else if scope == "media" {
		m, e := c.mediaMetadata(ctx, id)
		if e != nil {
			return t, e
		}
		t.key = "media:" + id
		t.kind = m.Kind
		t.provider = m.Metadata.Provider
		t.id = m.Metadata.ProviderID
		var existing int
		_ = c.db.QueryRowContext(ctx, `SELECT count(*) FROM poster_choices WHERE target=?`, t.key).Scan(&existing)
		if existing == 0 && t.id != "" {
			t.key = "work:" + workID(m.Kind, m.Metadata.Provider, m.Metadata.ProviderID)
		}
	} else {
		return t, errors.New("Cible invalide.")
	}
	// First delivery exposes movie/series artwork; the storage/provider contract is reusable.
	if !videoKind(t.kind) {
		return t, errors.New("Type de média non pris en charge.")
	}
	return t, nil
}
func (c *catalog) posterChoiceURL(ctx context.Context, key, scope, id string) string {
	var image string
	if c.db.QueryRowContext(ctx, `SELECT image_id FROM poster_choices WHERE target=?`, key).Scan(&image) != nil {
		return ""
	}
	return "/api/v1/posters/" + scope + "/" + id + "?v=" + image
}
func (c *catalog) posterOptions(ctx context.Context, t posterTarget) ([]PosterOption, error) {
	p := c.provider(t.provider)
	if p == nil || t.id == "" {
		return []PosterOption{}, nil
	}
	pp, ok := p.(PosterProvider)
	if !ok {
		return []PosterOption{}, nil
	}
	options, e := pp.Posters(ctx, c.credential(ctx, t.provider), MetadataSelection{Kind: t.kind, ID: t.id}, appLanguage)
	if e != nil {
		return nil, e
	}
	if len(options) > 100 {
		options = options[:100]
	}
	for i := range options {
		options[i].Image = assetURL(t.provider, options[i].Ref)
	}
	return options, nil
}
func (c *catalog) savePoster(ctx context.Context, t posterTarget, raw []byte, source string) error {
	c.customPosterMu.Lock()
	defer c.customPosterMu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	data, e := normalizeArtwork(raw, 1200)
	if e != nil {
		return e
	}
	if len(data) == 0 {
		return errors.New("Image invalide. Choisissez un JPEG ou PNG de 8 Mo et 16 millions de pixels maximum.")
	}
	unlock, err := providerCommit(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	sum := sha256.Sum256(data)
	id := hex.EncodeToString(sum[:])
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Bound permanent storage without ever evicting a user's selection.
	var size int64
	if e = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(length(data)),0) FROM custom_artwork`).Scan(&size); e != nil {
		return e
	}
	var exists int
	_ = tx.QueryRowContext(ctx, `SELECT count(*) FROM custom_artwork WHERE id=?`, id).Scan(&exists)
	if exists == 0 && size+int64(len(data)) > 256<<20 {
		return errors.New("Le stockage des affiches personnalisées est plein (256 Mo). Retirez des choix devenus inutiles.")
	}
	if _, e = tx.ExecContext(ctx, `INSERT OR IGNORE INTO custom_artwork VALUES(?,?)`, id, data); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO poster_choices(target,kind,image_id,source) VALUES(?,?,?,?) ON CONFLICT(target) DO UPDATE SET image_id=excluded.image_id,source=excluded.source`, t.key, t.kind, id, source); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `DELETE FROM custom_artwork WHERE id NOT IN(SELECT image_id FROM poster_choices)`); e != nil {
		return e
	}
	return tx.Commit()
}
func posterRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	resolve := func(w http.ResponseWriter, r *http.Request) (posterTarget, bool) {
		t, e := c.posterTarget(r.Context(), r.PathValue("scope"), r.PathValue("id"))
		if e != nil || !allowed(t.kind) {
			problem(w, 404, "Média introuvable.")
			return t, false
		}
		return t, true
	}
	mux.HandleFunc("GET /api/v1/posters/{scope}/{id}/options", func(w http.ResponseWriter, r *http.Request) {
		t, ok := resolve(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		options, e := c.posterOptions(ctx, t)
		warning := ""
		if e != nil {
			warning = e.Error()
			options = []PosterOption{}
		}
		sendJSON(w, 200, map[string]any{"items": options, "custom": c.posterChoiceURL(r.Context(), t.key, t.scope, t.routeID), "warning": warning})
	})
	mux.HandleFunc("GET /api/v1/posters/{scope}/{id}", func(w http.ResponseWriter, r *http.Request) {
		t, ok := resolve(w, r)
		if !ok {
			return
		}
		var data []byte
		if c.db.QueryRowContext(r.Context(), `SELECT a.data FROM custom_artwork a JOIN poster_choices p ON p.image_id=a.id WHERE p.target=?`, t.key).Scan(&data) != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, no-cache")
		http.ServeContent(w, r, "poster.jpg", time.Time{}, bytes.NewReader(data))
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		t, ok := resolve(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		var raw []byte
		var e error
		source := "upload"
		if r.Method == "POST" {
			var in struct {
				Ref string `json:"ref"`
			}
			if !readJSON(w, r, &in) {
				return
			}
			ctx, done, err := beginProvider(ctx, c.provider(t.provider))
			if err != nil {
				problem(w, 409, err.Error())
				return
			}
			defer done()
			options, err := c.posterOptions(ctx, t)
			if err != nil {
				problem(w, 502, err.Error())
				return
			}
			found := false
			for _, o := range options {
				found = found || o.Ref == in.Ref
			}
			if !found {
				problem(w, 400, "Cette affiche ne fait pas partie des propositions de ce média.")
				return
			}
			p := c.provider(t.provider)
			raw, e = p.Artwork(ctx, c.credential(ctx, t.provider), in.Ref)
			source = t.provider
		} else {
			r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
			raw, e = readBounded(r.Body, 8<<20)
		}
		if e != nil {
			problem(w, 400, "Impossible de charger cette image (8 Mo maximum).")
			return
		}
		if e = c.savePoster(ctx, t, raw, source); e != nil {
			problem(w, 400, e.Error())
			return
		}
		sendJSON(w, 200, map[string]string{"url": c.posterChoiceURL(ctx, t.key, t.scope, t.routeID)})
	}
	mux.HandleFunc("PUT /api/v1/posters/{scope}/{id}", save)
	mux.HandleFunc("POST /api/v1/posters/{scope}/{id}", save)
	mux.HandleFunc("DELETE /api/v1/posters/{scope}/{id}", func(w http.ResponseWriter, r *http.Request) {
		t, ok := resolve(w, r)
		if !ok {
			return
		}
		c.customPosterMu.Lock()
		defer c.customPosterMu.Unlock()
		tx, e := c.db.BeginTx(r.Context(), nil)
		if e != nil {
			problem(w, 500, "Affiche non rétablie.")
			return
		}
		defer tx.Rollback()
		_, e = tx.ExecContext(r.Context(), `DELETE FROM poster_choices WHERE target=?; DELETE FROM custom_artwork WHERE id NOT IN(SELECT image_id FROM poster_choices)`, t.key)
		if e == nil {
			e = tx.Commit()
		}
		if e != nil {
			problem(w, 500, "Affiche non rétablie.")
			return
		}
		w.WriteHeader(204)
	})
}
