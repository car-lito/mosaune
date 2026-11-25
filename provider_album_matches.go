package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Album matching is optional. Providers validate their own identifiers and
// return metadata; the host owns permissions, lifecycle and shared persistence.
type AlbumMatchProvider interface {
	MatchAlbums(context.Context, string, AlbumMatchQuery) (AlbumMatchResult, error)
}
type AlbumMatchQuery struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	ID     string `json:"id"`
	Page   int    `json:"page"`
}
type AlbumMatchResult struct {
	Page     DiscoveryPage
	Detailed bool
}

var errAlbumMatchInput = errors.New("Identifiant ou lien d’album invalide pour ce fournisseur.")

func albumMatchRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/music/matches/{provider}", func(w http.ResponseWriter, r *http.Request) {
		c.matchAlbums(w, r, allowed, c.provider(r.PathValue("provider")))
	})
	legacyMusicBrainzMatchRoutes(mux, c, allowed)
}

func (c *catalog) matchAlbums(w http.ResponseWriter, r *http.Request, allowed func(string) bool, p MetadataProvider) {
	if !canKind(r.Context(), "music") {
		problem(w, 403, "Votre compte n’a pas accès à la musique.")
		return
	}
	if p == nil || !allowed("music") || !supportsProvider(p, "music") {
		http.NotFound(w, r)
		return
	}
	matcher, ok := p.(AlbumMatchProvider)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var input AlbumMatchQuery
	if !readJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Artist = strings.TrimSpace(input.Artist)
	input.ID = strings.TrimSpace(input.ID)
	if input.Page == 0 {
		input.Page = 1
	}
	if len(input.Title) > 250 || len(input.Artist) > 250 || len(input.ID) > 200 || input.Page < 1 || input.Page > 50 || (input.ID == "" && len(input.Title) < 2) {
		problem(w, 400, "Indiquez un album ou un identifiant valide.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	ctx, done, err := beginProvider(ctx, p)
	if err != nil {
		problem(w, 409, err.Error())
		return
	}
	defer done()
	result, err := matcher.MatchAlbums(ctx, c.credential(ctx, p.Info().ID), input)
	if errors.Is(err, errAlbumMatchInput) {
		problem(w, 400, err.Error())
		return
	}
	if err != nil {
		problem(w, 502, err.Error())
		return
	}
	page := result.Page
	if len(page.Items) > 20 || page.Page < 1 || page.Page > 50 || page.Pages < page.Page || page.Pages > 50 {
		problem(w, 502, "Réponse du fournisseur invalide.")
		return
	}
	// Direct lookup may replace a cached summary, but only for this provider.
	if result.Detailed {
		for _, md := range page.Items {
			md.Provider = p.Info().ID
			if _, err = c.rememberWork(ctx, "music", md, true); err != nil {
				problem(w, 409, "Fiche non enregistrée. Relancez la recherche.")
				return
			}
		}
	}
	works, err := c.rememberPage(ctx, p.Info().ID, "music", page, allowed)
	if err != nil {
		problem(w, 409, "Recherche interrompue. Relancez la recherche.")
		return
	}
	unlock, err := providerCommit(ctx)
	if err != nil {
		problem(w, 409, "Le fournisseur a changé. Relancez la recherche.")
		return
	}
	defer unlock()
	sendJSON(w, 200, map[string]any{"items": works, "page": page.Page, "pages": page.Pages})
}
