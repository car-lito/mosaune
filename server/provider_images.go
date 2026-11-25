package main

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

// ImageReferenceProvider opts a trusted adapter into the public image route.
// It validates opaque references without I/O and returns their permitted media
// kinds. An empty result rejects the reference, including before cache lookup.
// Artwork remains responsible for bounded fetches through authorized transports.
type ImageReferenceProvider interface {
	ImageReferenceKinds(string) []string
}

func providerImageAllowed(ctx context.Context, p MetadataProvider, ref string, allowed func(string) bool) bool {
	if p == nil || len(ref) == 0 || len(ref) > 600 {
		return false
	}
	policy, ok := p.(ImageReferenceProvider)
	if !ok {
		return false
	}
	kinds := policy.ImageReferenceKinds(ref)
	if len(kinds) == 0 || len(kinds) > 16 {
		return false
	}
	seen := map[string]bool{}
	permitted := false
	for _, kind := range kinds {
		if kind == "" || seen[kind] || !supportsProvider(p, kind) {
			return false
		}
		seen[kind] = true
		permitted = permitted || (allowed(kind) && canKind(ctx, kind))
	}
	return permitted
}

func providerImageRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("GET /api/v1/metadata/images/{provider}", func(w http.ResponseWriter, r *http.Request) {
		p := c.provider(r.PathValue("provider"))
		ref := r.URL.Query().Get("ref")
		if !providerImageAllowed(r.Context(), p, ref, allowed) {
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
}
