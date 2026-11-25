package main

import (
	"context"
	"net/http"
)

// Optional trusted-adapter contract for a separately controllable execution
// module. Presence of a package or a manifest alone does not grant this control.
type providerRuntimeManager interface {
	providerRuntime() map[string]bool
	setProviderRuntime(context.Context, bool) error
}
type providerManagement struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Enabled            bool            `json:"enabled"`
	Available          bool            `json:"available"`
	Activation         bool            `json:"activation"`
	Packages           bool            `json:"packages"`
	RuntimeControl     bool            `json:"runtimeControl"`
	ConfigurationOwner string          `json:"configurationOwner,omitempty"`
	Runtime            map[string]bool `json:"runtime,omitempty"`
}

func (c *catalog) providerManagementRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/metadata/provider-management", func(w http.ResponseWriter, r *http.Request) {
		result := []providerManagement{}
		for _, p := range c.providers {
			info := p.Info()
			item := providerManagement{ID: info.ID, Name: info.Name, Enabled: providerEnabled(p), Available: providerAvailable(p)}
			_, item.Activation = p.(providerStateManager)
			_, item.Packages = p.(metadataPackageProvider)
			if configurable, ok := p.(MetadataConfigurationProvider); ok {
				item.ConfigurationOwner = configurable.ConfigurationOwner()
			}
			if runtime, ok := p.(providerRuntimeManager); ok {
				item.RuntimeControl = true
				item.Runtime = runtime.providerRuntime()
			}
			result = append(result, item)
		}
		sendJSON(w, 200, result)
	})
	mux.HandleFunc("PUT /api/v1/metadata/providers/{provider}/runtime", c.changeProviderRuntime)
}
func (c *catalog) changeProviderRuntime(w http.ResponseWriter, r *http.Request) {
	p, ok := c.provider(r.PathValue("provider")).(providerRuntimeManager)
	if !ok {
		problem(w, 404, "Ce fournisseur ne propose pas ce réglage de module.")
		return
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	if input.Enabled == nil {
		problem(w, 400, "État requis.")
		return
	}
	if err := p.setProviderRuntime(r.Context(), *input.Enabled); err != nil {
		problem(w, 400, err.Error())
		return
	}
	sendJSON(w, 200, p.providerRuntime())
}
func (p *tmdbProvider) providerRuntime() map[string]bool { return p.extensionStatus() }
func (p *tmdbProvider) setProviderRuntime(ctx context.Context, enabled bool) error {
	return p.setExtension(ctx, enabled)
}
