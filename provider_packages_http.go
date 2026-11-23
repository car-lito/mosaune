package main

import (
	"database/sql"
	"errors"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
)

func (c *catalog) packageProvider(w http.ResponseWriter, r *http.Request) metadataPackageProvider {
	p, ok := c.provider(r.PathValue("provider")).(metadataPackageProvider)
	if !ok {
		problem(w, 404, "Ce fournisseur ne propose pas de gestion de paquets.")
		return nil
	}
	return p
}

func (c *catalog) providerPackageRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/provider-packages", func(w http.ResponseWriter, r *http.Request) {
		// Discovery is based on implemented contracts, never a manifest claim.
		managers := []map[string]string{}
		for _, p := range c.providers {
			if _, ok := p.(metadataPackageProvider); ok {
				info := p.Info()
				managers = append(managers, map[string]string{"id": info.ID, "name": info.Name})
			}
		}
		sendJSON(w, 200, managers)
	})
	mux.HandleFunc("GET /api/v1/provider-packages/{provider}", func(w http.ResponseWriter, r *http.Request) {
		p := c.packageProvider(w, r)
		if p == nil {
			return
		}
		state, err := p.packageState(r.Context())
		if err != nil {
			problem(w, 500, "Paquet fournisseur illisible.")
			return
		}
		sendJSON(w, 200, state)
	})
	for _, action := range []string{"preview", "install"} {
		mux.HandleFunc("POST /api/v1/provider-packages/{provider}/"+action, func(w http.ResponseWriter, r *http.Request) {
			c.providerPackageUpload(w, r, action == "install")
		})
	}
	mux.HandleFunc("POST /api/v1/provider-packages/{provider}/rollback", func(w http.ResponseWriter, r *http.Request) {
		p := c.packageProvider(w, r)
		if p == nil {
			return
		}
		var input struct {
			Checksum         string `json:"checksum"`
			PreviousChecksum string `json:"previousChecksum"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		if err := p.rollbackPackage(r.Context(), input.Checksum, input.PreviousChecksum); err != nil {
			status := 409
			if errors.Is(err, errProviderRollbackCorrupt) {
				status = 400
			}
			problem(w, status, err.Error())
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("DELETE /api/v1/provider-packages/{provider}", func(w http.ResponseWriter, r *http.Request) {
		p := c.packageProvider(w, r)
		if p == nil {
			return
		}
		var input struct {
			Checksum string `json:"checksum"`
		}
		if !readJSON(w, r, &input) {
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

func (c *catalog) providerPackageUpload(w http.ResponseWriter, r *http.Request, install bool) {
	p := c.packageProvider(w, r)
	if p == nil {
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, packageformat.MaxArchive))
	if err != nil {
		problem(w, 400, "Paquet illisible ou trop volumineux (9 Mio maximum).")
		return
	}
	pack, err := p.parseProviderPackage(raw)
	if err != nil {
		problem(w, 400, err.Error())
		return
	}
	if !install {
		old, err := p.installedPackage(r.Context())
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			problem(w, 500, "Installation actuelle illisible.")
			return
		}
		if err == nil && packageformat.Compare(pack.Manifest.Descriptor.Version, old.Manifest.Descriptor.Version) <= 0 {
			problem(w, 409, "Une mise à jour doit avoir une version supérieure. Pour revenir en arrière, utilisez Restaurer la version précédente.")
			return
		}
		sendJSON(w, 200, map[string]any{"package": pack, "previousChecksum": old.Checksum, "previousVersion": old.Manifest.Descriptor.Version, "signer": p.packageSigner(), "services": c.serviceViews(pack.Manifest.Descriptor)})
		return
	}
	if r.Header.Get("X-Mosaune-Package-Checksum") != pack.Checksum {
		problem(w, 409, "Le paquet a changé. Vérifiez-le de nouveau.")
		return
	}
	c.extensionMu.Lock()
	defer c.extensionMu.Unlock()
	if err := p.installPackage(r.Context(), pack, r.Header.Get("X-Mosaune-Previous-Checksum"), false); err != nil {
		problem(w, 409, err.Error())
		return
	}
	sendJSON(w, 201, pack)
}
