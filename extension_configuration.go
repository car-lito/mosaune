package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	"mosaune/extensionkit/registry"
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
	if id != bundled.Descriptor().ID {
		// Only a host-registered installed module may expose its settings. A
		// guessed owner ID or a row alone cannot create an accessible extension.
		s, err := (registry.SQLStore{DB: c.db}).Load(context.Background())
		if err != nil {
			return configurationBinding{}, false
		}
		entry, ok := registryEntryFor(s, id)
		if !ok {
			return configurationBinding{}, false
		}
		bound, err := (configuration.SQLStore{DB: c.db}).For(id)
		if err != nil {
			return configurationBinding{}, false
		}
		state, err := bound.Read(context.Background())
		if err != nil {
			return configurationBinding{}, false
		}
		return configurationBinding{schema: state.Schema, descriptor: entry.Descriptor, read: bound.Read, write: bound.Apply}, true
	}
	if c.provider("tmdb") == nil {
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
	bound := tmdbConfiguration(c.db)
	return configurationBinding{schema: schema, descriptor: descriptor, read: bound.Read, write: bound.Apply}, true
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
	read := func(w http.ResponseWriter, r *http.Request) {
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
	}
	write := func(w http.ResponseWriter, r *http.Request) {
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
			if errors.Is(err, configuration.ErrConflict) {
				problem(w, 409, err.Error())
				return
			}
			problem(w, 500, "Réglages non enregistrés.")
			return
		}
		// Only acknowledge: no credential is ever serialized in this response.
		w.WriteHeader(http.StatusNoContent)
	}
	for _, path := range []string{"/api/v1/extensions/{id}/configuration", "/api/v1/extensions/bundled/{id}/configuration"} {
		mux.HandleFunc("GET "+path, read)
		mux.HandleFunc("PUT "+path, write)
	}
}
