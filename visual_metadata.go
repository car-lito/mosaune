package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func assetURL(provider, ref string) string {
	if ref == "" {
		return ""
	}
	return "/api/v1/metadata/images/" + url.PathEscape(provider) + "?ref=" + url.QueryEscape(ref)
}

// Image bytes stay in the bounded shared artwork cache. Small references and
// negative results expire, so a missing remote image isn't requested on every view.
// Positive references may be reused offline while a provider is disabled.
func (c *catalog) visualImage(ctx context.Context, p MetadataProvider, ref string) ([]byte, error) {
	c.posterMu.Lock()
	defer c.posterMu.Unlock()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	key := "visual-image:v2:" + ref
	var id []byte
	var expires int64
	var fallback []byte
	if c.db.QueryRowContext(ctx, `SELECT body,expires FROM provider_cache WHERE provider=? AND key=? AND (expires>? OR length(body)>0)`, p.Info().ID, key, time.Now().Unix()).Scan(&id, &expires) == nil {
		if len(id) == 0 {
			return nil, errors.New("Image absente.")
		}
		var data []byte
		if c.db.QueryRowContext(ctx, `SELECT data FROM artwork WHERE id=?`, string(id)).Scan(&data) == nil {
			if expires > time.Now().Unix() || !providerAvailable(p) {
				return data, nil
			}
			fallback = data
		}
	}
	ctx, done, err := beginProvider(ctx, p)
	if err != nil {
		return nil, err
	}
	defer done()
	raw, err := p.Artwork(ctx, c.credential(ctx, p.Info().ID), ref)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil && len(fallback) > 0 {
		return fallback, nil
	}
	if errors.Is(err, errRecognitionUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	unlock, e := providerCommit(ctx)
	if e != nil {
		return nil, e
	}
	defer unlock()
	imageID := ""
	if err == nil {
		imageID, err = storeArtwork(ctx, c.db, raw)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ttl := 7 * 24 * time.Hour
	if imageID == "" {
		ttl = 10 * time.Minute
	}
	_, _ = c.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<? AND NOT (key LIKE 'visual-image:v2:%' AND length(body)>0)`, time.Now().Unix())
	_, _ = c.db.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT ?,?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+?<16777216`, p.Info().ID, key, []byte(imageID), time.Now().Add(ttl).Unix(), len(imageID))
	if err != nil || imageID == "" {
		return nil, errors.New("Image indisponible.")
	}
	var data []byte
	err = c.db.QueryRowContext(ctx, `SELECT data FROM artwork WHERE id=?`, imageID).Scan(&data)
	return data, err
}

type ArtistResolver interface {
	ResolveArtist(context.Context, string) ([]Contributor, error)
}

func visualRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	albumMatchRoutes(mux, c, allowed)
	mux.HandleFunc("GET /api/v1/metadata/images/{provider}", func(w http.ResponseWriter, r *http.Request) {
		p := c.provider(r.PathValue("provider"))
		ref := r.URL.Query().Get("ref")
		enabled := false
		if p != nil {
			for _, kind := range p.Info().Kinds {
				enabled = enabled || allowed(kind)
			}
		}
		if !enabled || len(ref) == 0 || len(ref) > 600 {
			http.NotFound(w, r)
			return
		}
		// Only built-in providers' strictly validated references are accepted; never URLs.
		if p.Info().ID != "tmdb" && p.Info().ID != "musicbrainz" {
			http.NotFound(w, r)
			return
		}
		data, err := c.visualImage(r.Context(), p, ref)
		if err != nil {
			w.Header().Set("Cache-Control", "private, max-age=300")
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=86400")
		http.ServeContent(w, r, "image.jpg", time.Time{}, bytes.NewReader(data))
	})
	mux.HandleFunc("POST /api/v1/people/{provider}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if !canKind(r.Context(), "music") {
			problem(w, 403, "Votre compte n’a pas accès à la musique.")
			return
		}
		p := c.provider(r.PathValue("provider"))
		if p == nil || !allowed("music") || !supportsProvider(p, "music") {
			http.NotFound(w, r)
			return
		}
		resolver, ok := p.(ArtistResolver)
		if !ok {
			http.NotFound(w, r)
			return
		}
		var input struct {
			Name string `json:"name"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		if len(input.Name) < 2 || len(input.Name) > 250 {
			problem(w, 400, "Nom d’artiste invalide.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		ctx, done, err := beginProvider(ctx, p)
		if err != nil {
			problem(w, 409, err.Error())
			return
		}
		defer done()
		people, err := resolver.ResolveArtist(ctx, input.Name)
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		unlock, err := providerCommit(ctx)
		if err != nil {
			problem(w, 409, "Le fournisseur a changé. Relancez la recherche.")
			return
		}
		defer unlock()
		sendJSON(w, 200, people)
	})
}
