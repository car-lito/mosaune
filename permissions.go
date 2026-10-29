package main

import (
	"context"
	"net/http"
	"strings"
)

// All feature routes pass through this registry. Unclassified routes are admin-only.
// A future extension must declare its permission rather than inherit a wildcard.
type secureMux struct {
	*http.ServeMux
	c *catalog
}

func (m *secureMux) HandleFunc(pattern string, h http.HandlerFunc) {
	m.ServeMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if !m.c.authorize(pattern, r) {
			problem(w, 403, "Votre compte n’a pas accès à cette action ou à ce contenu.")
			return
		}
		h(w, r)
	})
}
func (c *catalog) mediaPermission(ctx context.Context, id, action string) bool {
	var library, kind string
	e := c.db.QueryRowContext(ctx, `SELECT m.library_id,l.kind FROM media m JOIN libraries l ON l.id=m.library_id WHERE m.id=?`, id).Scan(&library, &kind)
	return e == nil && canLibrary(ctx, library, kind, action)
}
func (c *catalog) authorize(pattern string, r *http.Request) bool {
	u := viewer(r.Context())
	if u == nil {
		return false
	}
	if strings.Contains(pattern, "/music/playlists/{id}") {
		var owner string
		return canKind(r.Context(), "music") && c.db.QueryRowContext(r.Context(), `SELECT owner FROM playlists WHERE id=?`, r.PathValue("id")).Scan(&owner) == nil && owner == u.ID
	}
	if u.Admin {
		return true
	}
	ctx := r.Context()
	id := r.PathValue("id")
	read := r.Method == "GET" || r.Method == "HEAD"
	switch pattern {
	case "GET /api/v1/ui/navigation", "GET /api/v1/ui/pages/{owner}/{page}", "GET /api/v1/extensions", "GET /api/v1/extensions/appearance", "GET /api/v1/extensions/theme.css", "PUT /api/v1/account/extension-theme":
		return true // The handlers scope packages, preferences and CSS to this account.
	case "GET /api/v1/status", "GET /api/v1/library", "GET /api/v1/metadata/providers", "GET /api/v1/discovery/library", "PUT /api/v1/account/preferences", "PUT /api/v1/account/password":
		return true
	case "POST /api/v1/detail-experience", "POST /api/v1/detail-header", "POST /api/v1/detail-sections", "POST /api/v1/detail-list", "POST /api/v1/discovery/browse", "POST /api/v1/shelves":
		return len(u.Kinds) > 0 // Body kind is checked by the handler.
	case "GET /api/v1/acquisition/requests":
		return true // Results are filtered by destination.
	}
	if strings.Contains(pattern, "/media/{id}/") {
		action := "edit"
		if strings.HasSuffix(pattern, "/stream") || strings.HasSuffix(pattern, "/progress") || strings.HasSuffix(pattern, "/state") {
			action = "play"
		}
		return c.mediaPermission(ctx, id, action)
	}
	if strings.Contains(pattern, "/music/playlists") {
		if !canKind(ctx, "music") {
			return false
		}
		if id == "" {
			return true
		}
		var owner string
		return c.db.QueryRowContext(ctx, `SELECT owner FROM playlists WHERE id=?`, id).Scan(&owner) == nil && owner == u.ID
	}
	if strings.Contains(pattern, "/discovery/works/{id}") {
		work, e := c.work(ctx, id)
		return e == nil && canKind(ctx, work.Kind)
	}
	if strings.Contains(pattern, "/posters/{scope}/{id}") {
		action := "view"
		if !read || strings.HasSuffix(pattern, "/options") {
			action = "edit"
		}
		if r.PathValue("scope") == "media" {
			return c.mediaPermission(ctx, id, action)
		}
		if r.PathValue("scope") != "works" {
			return false
		}
		work, e := c.work(ctx, id)
		if e != nil || !canKind(ctx, work.Kind) {
			return false
		}
		if action == "view" {
			return true
		}
		items, e := c.items(ctx)
		if e != nil {
			return false
		}
		attachAvailability(&work, items)
		for _, mid := range work.LocalIDs {
			if c.mediaPermission(ctx, mid, "edit") {
				return true
			}
		}
		return false
	}
	if pattern == "GET /api/v1/artwork/{id}" {
		rows, e := c.db.QueryContext(ctx, `SELECT m.library_id,l.kind FROM metadata d JOIN media m ON m.id=d.media_id JOIN libraries l ON l.id=m.library_id WHERE json_extract(d.data,'$.imageId')=? AND d.revision=m.revision AND m.available=1`, id)
		if e != nil {
			return false
		}
		defer rows.Close()
		for rows.Next() {
			var library, kind string
			if rows.Scan(&library, &kind) == nil && canLibrary(ctx, library, kind, "view") {
				return true
			}
		}
		return false
	}
	if strings.Contains(pattern, "/people/{provider}") || strings.Contains(pattern, "/metadata/images/{provider}") {
		p := c.provider(r.PathValue("provider"))
		if p == nil {
			return false
		}
		for _, k := range p.Info().Kinds {
			if canKind(ctx, k) {
				return true
			}
		}
		return false
	}
	if pattern == "POST /api/v1/music/matches" || pattern == "POST /api/v1/music/associate" {
		return canKind(ctx, "music")
	} // Association checks every affected media.
	if pattern == "POST /api/v1/acquisition/preview" || pattern == "POST /api/v1/acquisition/requests" {
		return true
	} // Destination is resolved and checked before contacting the service.
	return false
}
