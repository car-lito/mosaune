package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/resources"
	"net/http"
)

// These signed archives are standalone deliverables. Version 0.8.0 is provisioned
// once for a fresh installation; runtime execution only reads the installed archive.
var providerExample = resources.ProviderExample

var providerUpdateExample = resources.ProviderUpdateExample

var providerCatalogueExample = resources.ProviderCatalogueExample

var providerDiscoveryExample = resources.ProviderDiscoveryExample

var providerTransportExample = resources.ProviderTransportExample

var providerIdentifiersExample = resources.ProviderIdentifiersExample

// Compatibility URLs remain adapters to the same authenticated handlers.
func (c *catalog) tmdbPackageCompatibilityRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/provider-packages/example", func(w http.ResponseWriter, r *http.Request) {
		raw, name := providerExample, "tmdb-0.4.0.mosaune.zip"
		if r.URL.Query().Get("update") == "1" {
			raw, name = providerUpdateExample, "tmdb-0.4.1.mosaune.zip"
		}
		if r.URL.Query().Get("catalogue") == "1" {
			raw, name = providerCatalogueExample, "tmdb-0.5.0.mosaune.zip"
		}
		if r.URL.Query().Get("discovery") == "1" {
			raw, name = providerDiscoveryExample, "tmdb-0.6.0.mosaune.zip"
		}
		if r.URL.Query().Get("transport") == "1" {
			raw, name = providerTransportExample, "tmdb-0.7.0.mosaune.zip"
		}
		if r.URL.Query().Get("identifiers") == "1" {
			raw, name = providerIdentifiersExample, "tmdb-0.8.0.mosaune.zip"
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		_, _ = w.Write(raw)
	})

	for _, action := range []string{"preview", "install"} {
		mux.HandleFunc("POST /api/v1/provider-packages/"+action, func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("provider", "tmdb")
			c.providerPackageUpload(w, r, action == "install")
		})
	}
}

func (p *tmdbProvider) parseProviderPackage(raw []byte) (packageformat.Package, error) {
	return packageformat.Parse(raw, p.packageKey(), version)
}
func (p *tmdbProvider) packageSigner() string          { return "Clé de développement Mosaune" }
func (p *tmdbProvider) repositoryPackageIDs() []string { return []string{tmdbConfigurationOwner} }
func (p *tmdbProvider) rollbackPackage(ctx context.Context, current, previous string) error {
	var raw []byte
	var checksum string
	if err := p.db.QueryRowContext(ctx, "SELECT previous,previous_checksum FROM provider_packages WHERE id='tmdb'").Scan(&raw, &checksum); err != nil || len(raw) == 0 || checksum != previous {
		return errors.New("Version précédente absente ou modifiée.")
	}
	pack, err := p.parseProviderPackage(raw)
	if err != nil {
		return errProviderRollbackCorrupt
	}
	return p.installPackage(ctx, pack, current, true)
}
