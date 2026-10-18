package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (c *catalog) provider(id string) MetadataProvider {
	for _, p := range c.providers {
		if p.Info().ID == id {
			return p
		}
	}
	return nil
}
func (c *catalog) applyProvider(ctx context.Context, p MetadataProvider, secret string, m media, id string, manual bool) error {
	ctx, done, e := beginProvider(ctx, p)
	if e != nil {
		return e
	}
	defer done()
	if m.Metadata.EpisodeEnd > m.Metadata.Episode {
		return errors.New("Cet épisode multiple garde sa fiche locale pour le moment.")
	}
	md, err := p.Details(ctx, secret, MetadataSelection{Kind: m.Kind, ID: id, Season: m.Metadata.Season, Episode: m.Metadata.Episode})
	if err != nil {
		return err
	}
	md.Manual = manual
	if md.ArtworkRef != "" {
		data, imageErr := p.Artwork(ctx, secret, md.ArtworkRef)
		if imageErr == nil {
			unlock, e := providerCommit(ctx)
			if e != nil {
				return e
			}
			md.ImageID, err = storeArtwork(ctx, c.db, data)
			unlock()
			if err != nil {
				return err
			}
		}
	}
	return c.saveMetadata(ctx, m, md)
}

func metadataRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	tmdbExtensionRoutes(mux, c)
	providerStateRoutes(mux, c)
	metadataQueueRoutes(mux, c, allowed)
	mux.HandleFunc("GET /api/v1/metadata/providers", func(w http.ResponseWriter, r *http.Request) { sendJSON(w, 200, c.providerInfo(r.Context())) })
	mux.HandleFunc("PUT /api/v1/metadata/providers/{provider}", func(w http.ResponseWriter, r *http.Request) {
		p := c.provider(r.PathValue("provider"))
		if p == nil {
			problem(w, 404, "Fournisseur inconnu.")
			return
		}
		var input struct {
			Secret string `json:"secret"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		input.Secret = strings.TrimSpace(input.Secret)
		if len(input.Secret) < 20 || len(input.Secret) > 2048 || strings.ContainsAny(input.Secret, " \r\n\t") {
			problem(w, 400, "Copiez le jeton de lecture ou la clé API, sans préfixe Bearer.")
			return
		}
		_, err := c.db.ExecContext(r.Context(), `INSERT INTO provider_settings(id,secret) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET secret=excluded.secret`, p.Info().ID, input.Secret)
		if err != nil {
			problem(w, 500, "Configuration non enregistrée.")
			return
		}
		sendJSON(w, 200, map[string]bool{"configured": true})
	})
	mux.HandleFunc("DELETE /api/v1/metadata/providers/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if c.provider(r.PathValue("provider")) == nil {
			problem(w, 404, "Fournisseur inconnu.")
			return
		}
		if _, err := c.db.ExecContext(r.Context(), `DELETE FROM provider_settings WHERE id=?`, r.PathValue("provider")); err != nil {
			problem(w, 500, "Configuration non modifiée.")
			return
		}
		w.WriteHeader(204)
	})

	mux.HandleFunc("POST /api/v1/media/{id}/search-metadata", func(w http.ResponseWriter, r *http.Request) {
		m, err := c.mediaMetadata(r.Context(), r.PathValue("id"))
		if err != nil || !allowed(m.Kind) {
			problem(w, 404, "Média inconnu.")
			return
		}
		var input struct {
			Provider string `json:"provider"`
			Query    string `json:"query"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p := c.provider(input.Provider)
		if p == nil || !supportsProvider(p, m.Kind) {
			problem(w, 400, "Fournisseur incompatible.")
			return
		}
		input.Query = strings.TrimSpace(input.Query)
		if len(input.Query) < 2 || len(input.Query) > 250 {
			problem(w, 400, "Indiquez un titre entre 2 et 250 caractères.")
			return
		}
		result, err := p.Search(r.Context(), c.credential(r.Context(), p.Info().ID), MetadataQuery{Kind: m.Kind, Title: input.Query})
		if err != nil {
			problem(w, 502, err.Error())
			return
		}
		sendJSON(w, 200, result)
	})
	mux.HandleFunc("PUT /api/v1/media/{id}/identify", func(w http.ResponseWriter, r *http.Request) {
		if !c.metadataMu.TryLock() {
			problem(w, 409, "Une identification est déjà en cours.")
			return
		}
		defer c.metadataMu.Unlock()
		m, err := c.mediaMetadata(r.Context(), r.PathValue("id"))
		if err != nil || !allowed(m.Kind) {
			problem(w, 404, "Média inconnu.")
			return
		}
		var input struct {
			Provider string `json:"provider"`
			ID       string `json:"id"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p := c.provider(input.Provider)
		if p == nil || !supportsProvider(p, m.Kind) || input.ID == "" || len(input.ID) > 200 {
			problem(w, 400, "Identification invalide.")
			return
		}
		if err = c.applyProvider(r.Context(), p, c.credential(r.Context(), p.Info().ID), m, input.ID, true); err != nil {
			problem(w, 502, err.Error())
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/v1/artwork/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := hex.DecodeString(id); len(id) != 64 || err != nil {
			http.NotFound(w, r)
			return
		}
		// Do not expose images belonging only to disabled modules.
		rows, err := c.db.QueryContext(r.Context(), `SELECT DISTINCT l.kind FROM metadata d JOIN media m ON m.id=d.media_id JOIN libraries l ON l.id=m.library_id WHERE json_extract(d.data,'$.imageId')=? AND m.available=1`, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		permitted := false
		for rows.Next() {
			var kind string
			if rows.Scan(&kind) == nil && allowed(kind) {
				permitted = true
			}
		}
		rows.Close()
		if !permitted {
			http.NotFound(w, r)
			return
		}
		var data []byte
		if c.db.QueryRowContext(r.Context(), `SELECT data FROM artwork WHERE id=?`, id).Scan(&data) != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=86400")
		http.ServeContent(w, r, id+".jpg", time.Time{}, bytes.NewReader(data))
	})
}
