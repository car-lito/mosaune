package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
)

//go:embed examples/ambre/extension.json
var exampleTheme []byte

//go:embed examples/ambre-v2/extension.json
var exampleThemeV2 []byte

func readExtensionUpload(w http.ResponseWriter, r *http.Request) (extensionPackage, bool) {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, extensionLimit))
	if e != nil {
		problem(w, 400, "Paquet illisible ou trop volumineux (256 Ko maximum).")
		return extensionPackage{}, false
	}
	p, e := parseExtensionPackage(raw)
	if e != nil {
		problem(w, 400, e.Error())
		return p, false
	}
	return p, true
}
func (c *catalog) extensionRoutes(mux *secureMux) {
	c.extensionRepositoryRoutes(mux)
	c.extensionConfigurationRoutes(mux)
	c.providerPackageRoutes(mux)
	mux.HandleFunc("GET /api/v1/extensions/theme-template", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="extension.json"`)
		_, _ = w.Write(exampleThemeV2)
	})
	mux.HandleFunc("GET /api/v1/extensions/theme-contract", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="mosaune-theme-contract-v2.json"`)
		_, _ = w.Write(themeContractJSON)
	})

	mux.HandleFunc("GET /api/v1/extensions", func(w http.ResponseWriter, r *http.Request) {
		rows, e := c.db.QueryContext(r.Context(), `SELECT manifest,checksum,enabled FROM extensions WHERE ? OR id IN (SELECT extension_id FROM extension_grants WHERE user_id=?) ORDER BY id`, viewer(r.Context()).Admin, userID(r.Context()))
		if e != nil {
			problem(w, 500, "Extensions indisponibles.")
			return
		}
		list := []extensionPackage{}
		for rows.Next() {
			var p extensionPackage
			var raw string
			if e = rows.Scan(&raw, &p.Checksum, &p.Enabled); e != nil {
				break
			}
			if e = json.Unmarshal([]byte(raw), &p.Manifest); e != nil {
				break
			}
			p.Users = []string{}
			p.Preview = themePreviewFor(p.Manifest)
			list = append(list, p)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			problem(w, 500, "Extensions illisibles.")
			return
		}
		if viewer(r.Context()).Admin {
			for i := range list {
				grants, err := c.db.QueryContext(r.Context(), "SELECT user_id FROM extension_grants WHERE extension_id=? ORDER BY user_id", list[i].Manifest.ID)
				if err != nil {
					problem(w, 500, "Permissions indisponibles.")
					return
				}
				for grants.Next() {
					var id string
					if err = grants.Scan(&id); err != nil {
						break
					}
					list[i].Users = append(list[i].Users, id)
				}
				if err == nil {
					err = grants.Err()
				}
				grants.Close()
				if err != nil {
					problem(w, 500, "Permissions indisponibles.")
					return
				}
			}
		}
		sendJSON(w, 200, map[string]any{"packages": list, "safeMode": c.extensionSafeMode})
	})
	mux.HandleFunc("GET /api/v1/extensions/example", func(w http.ResponseWriter, r *http.Request) {
		var buffer bytes.Buffer
		z := zip.NewWriter(&buffer)
		f, e := z.Create("extension.json")
		if e == nil {
			_, e = f.Write(exampleThemeV2)
		}
		if e == nil {
			e = z.Close()
		}
		if e != nil {
			problem(w, 500, "Exemple indisponible.")
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="atelier-ambre-2.0.0.mosaune.zip"`)
		_, _ = w.Write(buffer.Bytes())
	})
	mux.HandleFunc("POST /api/v1/extensions/preview", func(w http.ResponseWriter, r *http.Request) {
		p, ok := readExtensionUpload(w, r)
		if !ok {
			return
		}
		old, e := c.extension(r.Context(), p.Manifest.ID)
		if e != nil && e != sql.ErrNoRows {
			problem(w, 500, "Catalogue des extensions indisponible.")
			return
		}
		if e == nil && (old.Manifest.Publisher != p.Manifest.Publisher || compareExtensionVersions(p.Manifest.Version, old.Manifest.Version) <= 0) {
			problem(w, 409, "Cet identifiant est déjà installé. Une mise à jour doit conserver l’éditeur déclaré et augmenter la version.")
			return
		}
		sendJSON(w, 200, map[string]any{"package": p, "previousChecksum": old.Checksum, "previousVersion": old.Manifest.Version})
	})
	mux.HandleFunc("POST /api/v1/extensions/install", func(w http.ResponseWriter, r *http.Request) {
		p, ok := readExtensionUpload(w, r)
		if !ok {
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		old, e := c.extension(r.Context(), p.Manifest.ID)
		if e != nil && e != sql.ErrNoRows {
			problem(w, 500, "Catalogue indisponible.")
			return
		}
		if r.Header.Get("X-Mosaune-Package-Checksum") != p.Checksum || r.Header.Get("X-Mosaune-Previous-Checksum") != old.Checksum {
			problem(w, 409, "Le paquet ou l’installation a changé. Vérifiez à nouveau le paquet.")
			return
		}
		if e == nil && (old.Manifest.Publisher != p.Manifest.Publisher || compareExtensionVersions(p.Manifest.Version, old.Manifest.Version) <= 0) {
			problem(w, 409, "Une mise à jour doit conserver l’éditeur déclaré et augmenter la version.")
			return
		}
		var count int
		if c.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM extensions").Scan(&count) != nil {
			problem(w, 500, "Catalogue indisponible.")
			return
		}
		if e == sql.ErrNoRows && count >= 32 {
			problem(w, 400, "Limite de 32 paquets atteinte.")
			return
		}
		raw, _ := json.Marshal(p.Manifest)
		// One atomic write, no extraction or partially active installation. Updates
		// preserve selections/grants but require an explicit activation of the new version.
		_, e = c.db.ExecContext(r.Context(), `INSERT INTO extensions(id,manifest,checksum,enabled) VALUES(?,?,?,0) ON CONFLICT(id) DO UPDATE SET manifest=excluded.manifest,checksum=excluded.checksum,enabled=0`, p.Manifest.ID, string(raw), p.Checksum)
		if e != nil {
			problem(w, 500, "Installation non enregistrée.")
			return
		}
		sendJSON(w, 201, p)
	})
	mux.HandleFunc("PUT /api/v1/extensions/{id}/state", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Enabled  bool   `json:"enabled"`
			Checksum string `json:"checksum"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		p, e := c.extension(r.Context(), r.PathValue("id"))
		if e != nil || p.Checksum != in.Checksum {
			problem(w, 409, "Extension absente ou modifiée. Actualisez la liste.")
			return
		}
		if in.Enabled && c.extensionSafeMode {
			problem(w, 409, "Le mode de secours empêche l’activation. Relancez Mosaune normalement pour le quitter.")
			return
		}
		if in.Enabled {
			if e = validateExtension(p.Manifest); e != nil {
				problem(w, 400, e.Error())
				return
			}
		}
		if _, e = c.db.ExecContext(r.Context(), "UPDATE extensions SET enabled=? WHERE id=?", in.Enabled, p.Manifest.ID); e != nil {
			problem(w, 500, "État non enregistré.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("PUT /api/v1/extensions/{id}/access", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Users    []string `json:"users"`
			Checksum string   `json:"checksum"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if len(in.Users) > 100 {
			problem(w, 400, "Trop de comptes.")
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		p, e := c.extension(r.Context(), r.PathValue("id"))
		if e != nil || p.Checksum != in.Checksum {
			problem(w, 409, "Extension absente ou modifiée.")
			return
		}
		tx, e := c.db.BeginTx(r.Context(), nil)
		if e != nil {
			problem(w, 500, "Permissions non enregistrées.")
			return
		}
		defer tx.Rollback()
		seen := map[string]bool{}
		for _, id := range in.Users {
			var n int
			if seen[id] || tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM accounts WHERE id=?", id).Scan(&n) != nil || n != 1 {
				problem(w, 400, "Compte inconnu ou en double.")
				return
			}
			seen[id] = true
		}
		_, e = tx.ExecContext(r.Context(), "DELETE FROM extension_grants WHERE extension_id=?", p.Manifest.ID)
		if e == nil {
			for _, id := range in.Users {
				if _, e = tx.ExecContext(r.Context(), "INSERT INTO extension_grants VALUES(?,?)", id, p.Manifest.ID); e != nil {
					break
				}
			}
		}
		if e != nil || tx.Commit() != nil {
			problem(w, 500, "Permissions non enregistrées.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("DELETE /api/v1/extensions/{id}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Checksum string `json:"checksum"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		result, e := c.db.ExecContext(r.Context(), "DELETE FROM extensions WHERE id=? AND checksum=?", r.PathValue("id"), in.Checksum)
		if e != nil {
			problem(w, 500, "Désinstallation non enregistrée.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 409, "Extension absente ou modifiée. Actualisez la liste.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("PUT /api/v1/account/extension-theme", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ID string `json:"id"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		c.extensionMu.Lock()
		defer c.extensionMu.Unlock()
		if in.ID != "" {
			if !c.canUseExtension(r.Context(), in.ID) {
				problem(w, 403, "Ce compte n’est pas autorisé à utiliser cette extension.")
				return
			}
			p, e := c.extension(r.Context(), in.ID)
			if e != nil || !p.Enabled || c.extensionSafeMode {
				problem(w, 409, "Ce thème n’est pas actif.")
				return
			}
		}
		_, e := c.db.ExecContext(r.Context(), `INSERT INTO extension_preferences VALUES(?,NULLIF(?,'')) ON CONFLICT(user_id) DO UPDATE SET theme_id=excluded.theme_id`, userID(r.Context()), in.ID)
		if e != nil {
			problem(w, 500, "Choix non enregistré.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/v1/extensions/appearance", func(w http.ResponseWriter, r *http.Request) {
		a, _, e := c.extensionAppearance(r.Context())
		if e != nil {
			problem(w, 500, "Apparence indisponible.")
			return
		}
		sendJSON(w, 200, a)
	})
	mux.HandleFunc("GET /api/v1/extensions/theme.css", func(w http.ResponseWriter, r *http.Request) {
		a, p, e := c.extensionAppearance(r.Context())
		if e != nil {
			problem(w, 500, "Thème indisponible.")
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if a.Effective != "" && r.URL.Query().Get("stamp") == a.Stamp {
			_, _ = io.WriteString(w, extensionCSS(p, a.Stamp))
		}
	})
}
