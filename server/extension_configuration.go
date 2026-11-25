package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"net/http"
)

// Schemas cannot select endpoints, tables, SQL or credentials. Trusted host
// adapters bind a module ID to its own configuration and storage ownership.
type configurationBinding struct {
	authorizationsManaged bool
	schema                configuration.Schema
	descriptor            protocol.Descriptor
	read                  func(context.Context) (configuration.State, error)
	write                 func(context.Context, configuration.Patch) error
}

func (c *catalog) configurationFor(id string) (configurationBinding, bool) {
	return c.configurationForContext(context.Background(), id)
}

func (c *catalog) configurationForContext(ctx context.Context, id string) (configurationBinding, bool) {
	// Connector settings belong to a shared capability. They must pass through
	// its destination checks, never the generic module settings write endpoint.
	if id == "mosaune.acquisition.radarr" || id == "mosaune.acquisition.sonarr" {
		return configurationBinding{}, false
	}
	for _, provider := range c.providers {
		if configurable, ok := provider.(MetadataConfigurationProvider); ok && configurable.ConfigurationOwner() == id {
			binding, err := configurable.Configuration(ctx)
			if err == nil {
				_, binding.authorizationsManaged = c.authorizationFor(id)
			}
			return binding, err == nil
		}
	}
	// Only a host-registered installed module may expose its settings.
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
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
	state, err := bound.Read(ctx)
	if err != nil {
		return configurationBinding{}, false
	}
	return configurationBinding{schema: state.Schema, descriptor: entry.Descriptor, read: bound.Read, write: bound.Apply}, true
}

// The host describes the effective named services, not arbitrary manifest prose.
type extensionServiceView struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Operations  []string `json:"operations"`
}

func (c *catalog) extensionConfigurationRoutes(mux *secureMux) {
	c.extensionAuthorizationRoutes(mux)
	read := func(w http.ResponseWriter, r *http.Request) {
		binding, ok := c.configurationForContext(r.Context(), r.PathValue("id"))
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
			Services              []extensionServiceView `json:"services"`
			AuthorizationsManaged bool                   `json:"authorizationsManaged"`
		}{state, c.serviceViews(binding.descriptor), binding.authorizationsManaged})
	}
	write := func(w http.ResponseWriter, r *http.Request) {
		binding, ok := c.configurationForContext(r.Context(), r.PathValue("id"))
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
