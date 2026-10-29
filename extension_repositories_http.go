package main

import (
	"database/sql"
	"encoding/json"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"strings"
	"time"
)

func (c *catalog) extensionRepositoryRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/extension-repositories/example", func(w http.ResponseWriter, r *http.Request) {
		raw, err := bundledRepositoryArchive()
		if err != nil {
			problem(w, 500, "Exemple indisponible.")
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="mosaune-repository-example.zip"`)
		_, _ = w.Write(raw)
	})
	// All routes are admin-only under secureMux's default-deny policy.
	mux.HandleFunc("GET /api/v1/extension-repositories", func(w http.ResponseWriter, r *http.Request) {
		repos := []extensionRepository{bundledRepository()}
		official, err := c.repository(r.Context(), officialRepository().ID)
		if err != nil {
			problem(w, 500, "Catalogue officiel illisible.")
			return
		}
		repos = append(repos, official)
		rows, err := c.db.QueryContext(r.Context(), `SELECT id,url,ref,index_json,updated_at FROM extension_repositories ORDER BY url,ref`)
		if err != nil {
			problem(w, 500, "Dépôts indisponibles.")
			return
		}
		for rows.Next() {
			var repo extensionRepository
			var raw string
			if err = rows.Scan(&repo.ID, &repo.URL, &repo.Ref, &raw, &repo.UpdatedAt); err != nil {
				break
			}
			repo.Index, err = parseRepositoryIndex([]byte(raw))
			if err != nil {
				break
			}
			if repo.ID != official.ID {
				repos = append(repos, repo)
			}
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			problem(w, 500, "Catalogue illisible.")
			return
		}
		type installedVersion struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Kind    string `json:"kind"`
		}
		installed := []installedVersion{}
		rows, err = c.db.QueryContext(r.Context(), "SELECT manifest FROM extensions")
		if err != nil {
			problem(w, 500, "Extensions indisponibles.")
			return
		}
		for rows.Next() {
			var raw string
			var m extensionManifest
			if err = rows.Scan(&raw); err != nil {
				break
			}
			if err = json.Unmarshal([]byte(raw), &m); err != nil {
				break
			}
			installed = append(installed, installedVersion{m.ID, m.Version, uiPackageKind(m)})
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			problem(w, 500, "Extensions illisibles.")
			return
		}
		if p, ok := c.provider("tmdb").(*tmdbProvider); ok {
			pack, e := p.installedPackage(r.Context())
			if e == nil {
				installed = append(installed, installedVersion{pack.Manifest.Descriptor.ID, pack.Manifest.Descriptor.Version, "provider"})
			} else if e != sql.ErrNoRows {
				problem(w, 500, "Paquet fournisseur illisible.")
				return
			}
		}
		modules, _, err := c.recognitionPackages(r.Context())
		if err != nil {
			problem(w, 500, "Modules de reconnaissance illisibles.")
			return
		}
		for _, module := range modules {
			if module.Installed {
				kind := "recognition"
				if module.Kind == "detail-experience" {
					kind = "detail-experience"
				}
				if module.Kind == "detail-header" {
					kind = "detail-header"
				}
				if module.Kind == "detail-information" {
					kind = "detail-sections"
				}
				if module.Kind == "detail-series" {
					kind = "detail-list"
				}
				if strings.HasPrefix(module.Kind, "catalogue-") {
					kind = "catalogue"
				}
				if module.Kind == "audiotags" {
					kind = "audio-tags"
				}
				installed = append(installed, installedVersion{module.ID, module.Version, kind})
			}
		}
		sendJSON(w, 200, map[string]any{"repositories": repos, "installed": installed, "coreVersion": version})
	})
	mux.HandleFunc("POST /api/v1/extension-repositories", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			URL string `json:"url"`
			Ref string `json:"ref"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		input.Ref = strings.TrimSpace(input.Ref)
		canonical, err := normalizeRepository(input.URL, input.Ref)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		c.repositoryMu.Lock()
		defer c.repositoryMu.Unlock()
		var count int
		if c.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM extension_repositories").Scan(&count) != nil {
			problem(w, 500, "Dépôts indisponibles.")
			return
		}
		if count >= 20 {
			problem(w, 400, "Limite de 20 dépôts atteinte.")
			return
		}
		repo := extensionRepository{ID: packageformat.Hash([]byte(canonical + "\n" + input.Ref)), URL: canonical, Ref: input.Ref}
		if _, err = c.repository(r.Context(), repo.ID); err == nil {
			problem(w, 409, "Ce dépôt est déjà ajouté.")
			return
		} else if err != sql.ErrNoRows {
			problem(w, 500, "Dépôts indisponibles.")
			return
		}
		raw, err := c.fetchRepositoryFile(r.Context(), repo, repositoryIndexFile, repositoryIndexLimit)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		repo.Index, err = parseRepositoryIndex(raw)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		repo.UpdatedAt = time.Now().Unix()
		_, err = c.db.ExecContext(r.Context(), `INSERT INTO extension_repositories VALUES(?,?,?,?,?)`, repo.ID, repo.URL, repo.Ref, string(raw), repo.UpdatedAt)
		if err != nil {
			problem(w, 500, "Dépôt non enregistré.")
			return
		}
		sendJSON(w, 201, repo)
	})
	mux.HandleFunc("POST /api/v1/extension-repositories/{id}/refresh", func(w http.ResponseWriter, r *http.Request) {
		c.repositoryMu.Lock()
		defer c.repositoryMu.Unlock()
		repo, err := c.repository(r.Context(), r.PathValue("id"))
		if err != nil || repo.Bundled {
			problem(w, 400, "Ce dépôt ne peut pas être actualisé.")
			return
		}
		raw, err := c.fetchRepositoryFile(r.Context(), repo, repositoryIndexFile, repositoryIndexLimit)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		index, err := parseRepositoryIndex(raw)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		repo.Index = index
		repo.UpdatedAt = time.Now().Unix()
		if _, err = c.db.ExecContext(r.Context(), `INSERT INTO extension_repositories(id,url,ref,index_json,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET index_json=excluded.index_json,updated_at=excluded.updated_at`, repo.ID, repo.URL, repo.Ref, string(raw), repo.UpdatedAt); err != nil {
			problem(w, 500, "Catalogue non enregistré. Version précédente conservée.")
			return
		}
		sendJSON(w, 200, repo)
	})
	mux.HandleFunc("DELETE /api/v1/extension-repositories/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == officialRepository().ID {
			problem(w, 400, "Le dépôt officiel reste proposé. Aucune connexion n’est effectuée sans actualisation ou téléchargement demandé.")
			return
		}
		if r.PathValue("id") == bundledRepositoryID {
			problem(w, 400, "Les paquets fournis avec Mosaune restent disponibles.")
			return
		}
		c.repositoryMu.Lock()
		defer c.repositoryMu.Unlock()
		result, err := c.db.ExecContext(r.Context(), "DELETE FROM extension_repositories WHERE id=?", r.PathValue("id"))
		if err != nil {
			problem(w, 500, "Dépôt non retiré.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 404, "Dépôt absent.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("POST /api/v1/extension-repositories/{id}/package", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			SHA256  string `json:"sha256"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		// Serialize remote downloads with refreshes: bounded work on small servers.
		c.repositoryMu.Lock()
		defer c.repositoryMu.Unlock()
		repo, err := c.repository(r.Context(), r.PathValue("id"))
		if err != nil {
			problem(w, 404, "Dépôt absent.")
			return
		}
		matched := false
		for _, p := range repo.Index.Packages {
			if p.ID == input.ID && p.Version == input.Version && p.SHA256 == input.SHA256 {
				matched = true
				break
			}
		}
		if !matched {
			problem(w, 409, "La fiche du catalogue a changé. Actualisez la liste.")
			return
		}
		raw, err := c.repositoryPackage(r.Context(), repo, input.ID, input.Version)
		if err != nil {
			problem(w, 400, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(raw)
	})
}
