package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	"net/http"
)

// Schemas cannot select endpoints, tables, SQL or credentials. Trusted host
// adapters bind a module ID to its own configuration and storage ownership.
type configurationBinding struct {
	schema     configuration.Schema
	descriptor protocol.Descriptor
	read       func(context.Context) (configuration.State, error)
	write      func(context.Context, configuration.Patch) error
}

func (c *catalog) configurationFor(id string) (configurationBinding, bool) {
	if id != bundled.Descriptor().ID || c.provider("tmdb") == nil {
		return configurationBinding{}, false
	}
	schema := bundled.Settings()
	descriptor := bundled.Descriptor()
	if p, ok := c.provider("tmdb").(*tmdbProvider); ok {
		if installed, err := p.installedPackage(context.Background()); err == nil {
			schema = installed.Manifest.Settings
			descriptor = installed.Manifest.Descriptor
		}
	}
	return configurationBinding{schema: schema, descriptor: descriptor, read: func(ctx context.Context) (configuration.State, error) {
		var configured bool
		err := c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM provider_settings WHERE id='tmdb' AND secret<>'')`).Scan(&configured)
		return configuration.State{Schema: schema, Values: map[string]json.RawMessage{}, SecretsConfigured: map[string]bool{"apiToken": configured}}, err
	}, write: func(ctx context.Context, patch configuration.Patch) error {
		tx, err := c.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if raw, ok := patch.Values["apiToken"]; ok {
			var secret string
			if json.Unmarshal(raw, &secret) != nil {
				return errors.New("invalid credential")
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO provider_settings(id,secret) VALUES('tmdb',?) ON CONFLICT(id) DO UPDATE SET secret=excluded.secret`, secret)
		}
		if err == nil && len(patch.ClearSecrets) > 0 {
			_, err = tx.ExecContext(ctx, `DELETE FROM provider_settings WHERE id='tmdb'`)
		}
		if err != nil {
			return err
		}
		return tx.Commit()
	}}, true
}

// The host describes the effective named services, not arbitrary manifest prose.
type extensionServiceView struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Operations  []string `json:"operations"`
}

func serviceViews(d protocol.Descriptor) []extensionServiceView {
	views := []extensionServiceView{}
	for _, s := range d.Services {
		if s.Name == "tmdb.api.v1" {
			views = append(views, extensionServiceView{Name: s.Name, Label: "Consulter TMDB", Description: "Recherche et fiches de l’œuvre demandée via api.themoviedb.org. La clé et le cache sont gérés par Mosaune. Aucun accès direct aux fichiers ou au catalogue.", Operations: s.Operations})
		}
	}
	return views
}
func (c *catalog) extensionConfigurationRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/extensions/bundled/{id}/configuration", func(w http.ResponseWriter, r *http.Request) {
		binding, ok := c.configurationFor(r.PathValue("id"))
		if !ok {
			problem(w, 404, "Réglages d’extension introuvables.")
			return
		}
		state, err := binding.read(r.Context())
		if err != nil {
			problem(w, 500, "Réglages indisponibles.")
			return
		}
		sendJSON(w, 200, struct {
			configuration.State
			Services []extensionServiceView `json:"services"`
		}{state, serviceViews(binding.descriptor)})
	})
	mux.HandleFunc("PUT /api/v1/extensions/bundled/{id}/configuration", func(w http.ResponseWriter, r *http.Request) {
		binding, ok := c.configurationFor(r.PathValue("id"))
		if !ok {
			problem(w, 404, "Réglages d’extension introuvables.")
			return
		}
		var input configuration.Patch
		if !readJSON(w, r, &input) {
			return
		}
		patch, err := binding.schema.NormalizePatch(input)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		if err = binding.write(r.Context(), patch); err != nil {
			problem(w, 500, "Réglages non enregistrés.")
			return
		}
		// Only acknowledge: no credential is ever serialized in this response.
		w.WriteHeader(http.StatusNoContent)
	})
}
