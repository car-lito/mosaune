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
	if c.db.QueryRowContext(ctx, `SELECT body FROM provider_cache WHERE provider=? AND key=? AND expires>?`, p.Info().ID, key, time.Now().Unix()).Scan(&id) == nil {
		if len(id) == 0 {
			return nil, errors.New("Image absente.")
		}
		var data []byte
		if c.db.QueryRowContext(ctx, `SELECT data FROM artwork WHERE id=?`, string(id)).Scan(&data) == nil {
			return data, nil
		}
	}
	raw, err := p.Artwork(ctx, c.credential(ctx, p.Info().ID), ref)
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
	_, _ = c.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<?`, time.Now().Unix())
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
	mux.HandleFunc("POST /api/v1/music/matches", func(w http.ResponseWriter, r *http.Request) {
		p, ok := c.provider("musicbrainz").(*musicBrainzProvider)
		if !ok || !allowed("music") {
			http.NotFound(w, r)
			return
		}
		var input struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
			ID     string `json:"id"`
			Page   int    `json:"page"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		input.Title = strings.TrimSpace(input.Title)
		input.Artist = strings.TrimSpace(input.Artist)
		input.ID = strings.TrimSpace(input.ID)
		if input.Page == 0 {
			input.Page = 1
		}
		if len(input.Title) > 250 || len(input.Artist) > 250 || len(input.ID) > 200 || input.Page < 1 || input.Page > 50 {
			problem(w, 400, "Recherche invalide.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		page := DiscoveryPage{Page: input.Page, Pages: 1}
		var err error
		if input.ID != "" {
			id := input.ID
			if u, e := url.Parse(id); e == nil && u.Scheme == "https" && u.Host == "musicbrainz.org" && strings.HasPrefix(u.Path, "/release/") {
				id = strings.TrimPrefix(strings.TrimSuffix(u.Path, "/"), "/release/")
			}
			if !mbID.MatchString(id) {
				problem(w, 400, "Collez un identifiant ou un lien d’édition MusicBrainz /release/.")
				return
			}
			var md Metadata
			md, err = p.Details(ctx, "", MetadataSelection{Kind: "music", ID: id})
			if err == nil {
				// A direct edition lookup is authoritative, including when a light
				// search result for the same ID was already cached.
				if _, saveErr := c.rememberWork(ctx, "music", md, true); saveErr != nil {
					problem(w, 500, "Fiche non enregistrée.")
					return
				}
				page.Items = []Metadata{md}
			}
		} else {
			if len(input.Title) < 2 {
				problem(w, 400, "Indiquez au moins deux caractères pour le titre.")
				return
			}
			q := "release:" + mbQuote(input.Title)
			if input.Artist != "" {
				q += " AND artist:" + mbQuote(input.Artist)
			}
			page, err = p.searchReleases(ctx, q, input.Page)
		}
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		works, err := c.rememberPage(ctx, "musicbrainz", "music", page, allowed)
		if err != nil {
			problem(w, 500, "Recherche indisponible.")
			return
		}
		sendJSON(w, 200, map[string]any{"items": works, "page": page.Page, "pages": page.Pages})
	})
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
		people, err := resolver.ResolveArtist(ctx, input.Name)
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		sendJSON(w, 200, people)
	})
}
