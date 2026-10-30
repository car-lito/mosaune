package main

import (
	"database/sql"
	_ "embed"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
)

// These signed examples are also standalone deliverables. Installed bytes come
// from the administrator's upload, never from selecting an embedded version.
//
//go:embed packages/tmdb-0.4.0.mosaune.zip
var providerExample []byte

//go:embed packages/tmdb-0.4.1.mosaune.zip
var providerUpdateExample []byte

//go:embed packages/tmdb-0.5.0.mosaune.zip
var providerCatalogueExample []byte

func (c *catalog) providerPackageRoutes(mux *secureMux) {
	provider := func() *tmdbProvider { p, _ := c.provider("tmdb").(*tmdbProvider); return p }
	mux.HandleFunc("GET /api/v1/provider-packages/tmdb", func(w http.ResponseWriter, r *http.Request) {
		p := provider()
		if p == nil {
			problem(w, 404, "Fournisseur absent.")
			return
		}
		state, err := p.packageState(r.Context())
		if err != nil {
			problem(w, 500, "Paquet fournisseur illisible.")
			return
		}
		sendJSON(w, 200, state)
	})
	mux.HandleFunc("GET /api/v1/provider-packages/example", func(w http.ResponseWriter, r *http.Request) {
		raw, name := providerExample, "tmdb-0.4.0.mosaune.zip"
		if r.URL.Query().Get("update") == "1" {
			raw, name = providerUpdateExample, "tmdb-0.4.1.mosaune.zip"
		}
		if r.URL.Query().Get("catalogue") == "1" {
			raw, name = providerCatalogueExample, "tmdb-0.5.0.mosaune.zip"
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		_, _ = w.Write(raw)
	})
	upload := func(w http.ResponseWriter, r *http.Request) (packageformat.Package, bool) {
		p := provider()
		if p == nil {
			problem(w, 404, "Fournisseur absent.")
			return packageformat.Package{}, false
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, packageformat.MaxArchive))
		if err != nil {
			problem(w, 400, "Paquet illisible ou trop volumineux (9 Mio maximum).")
			return packageformat.Package{}, false
		}
		pack, err := packageformat.Parse(raw, p.packageKey(), version)
		if err != nil {
			problem(w, 400, err.Error())
			return pack, false
		}
		return pack, true
	}
	mux.HandleFunc("POST /api/v1/provider-packages/preview", func(w http.ResponseWriter, r *http.Request) {
		pack, ok := upload(w, r)
		if !ok {
			return
		}
		old, err := provider().installedPackage(r.Context())
		if err != nil && err != sql.ErrNoRows {
			problem(w, 500, "Installation actuelle illisible.")
			return
		}
		if err == nil && packageformat.Compare(pack.Manifest.Descriptor.Version, old.Manifest.Descriptor.Version) <= 0 {
			problem(w, 409, "Une mise à jour doit avoir une version supérieure. Pour revenir en arrière, utilisez Restaurer la version précédente.")
			return
		}
		sendJSON(w, 200, map[string]any{"package": pack, "previousChecksum": old.Checksum, "previousVersion": old.Manifest.Descriptor.Version, "signer": "Clé de développement Mosaune", "services": serviceViews(pack.Manifest.Descriptor)})
	})
	mux.HandleFunc("POST /api/v1/provider-packages/install", func(w http.ResponseWriter, r *http.Request) {
		pack, ok := upload(w, r)
		if !ok {
			return
		}
		if r.Header.Get("X-Mosaune-Package-Checksum") != pack.Checksum {
			problem(w, 409, "Le paquet a changé. Vérifiez-le de nouveau.")
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		if err := provider().installPackage(r.Context(), pack, r.Header.Get("X-Mosaune-Previous-Checksum"), false); err != nil {
			problem(w, 409, err.Error())
			return
		}
		sendJSON(w, 201, pack)
	})
	mux.HandleFunc("POST /api/v1/provider-packages/tmdb/rollback", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Checksum         string `json:"checksum"`
			PreviousChecksum string `json:"previousChecksum"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p := provider()
		if p == nil {
			problem(w, 404, "Fournisseur absent.")
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		var raw []byte
		var checksum string
		if err := c.db.QueryRowContext(r.Context(), `SELECT previous,previous_checksum FROM provider_packages WHERE id='tmdb'`).Scan(&raw, &checksum); err != nil || len(raw) == 0 || checksum != input.PreviousChecksum {
			problem(w, 409, "Version précédente absente ou modifiée.")
			return
		}
		pack, err := packageformat.Parse(raw, p.packageKey(), version)
		if err != nil {
			problem(w, 400, "Version précédente altérée.")
			return
		}
		if err = p.installPackage(r.Context(), pack, input.Checksum, true); err != nil {
			problem(w, 409, err.Error())
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("DELETE /api/v1/provider-packages/tmdb", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Checksum string `json:"checksum"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p := provider()
		if p == nil {
			problem(w, 404, "Fournisseur absent.")
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		if err := p.removePackage(r.Context(), input.Checksum); err != nil {
			problem(w, 409, err.Error())
			return
		}
		w.WriteHeader(204)
	})
}
