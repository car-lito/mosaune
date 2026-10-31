package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"net/http"
	"slices"
)

// Supplied group manifests describe composition, not executable code or trust.
// External bundle archives and version constraints remain a separate installer step.
//
//go:embed extensions/media-*/extension.json
var suppliedMediaBundles embed.FS

type bundleComponent struct {
	Slot        string `json:"slot"`
	Composition string `json:"composition"`
	Shared      bool   `json:"shared"`
}
type mediaBundleManifest struct {
	Schema     int               `json:"schema"`
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Kind       string            `json:"kind"`
	Components []bundleComponent `json:"components"`
	Screen     string            `json:"screen"`
}

func mediaBundleDefinitions() ([]mediaBundleManifest, error) {
	dirs, e := suppliedMediaBundles.ReadDir("extensions")
	if e != nil {
		return nil, e
	}
	result := []mediaBundleManifest{}
	for _, dir := range dirs {
		raw, e := suppliedMediaBundles.ReadFile("extensions/" + dir.Name() + "/extension.json")
		if e != nil {
			return nil, e
		}
		var m mediaBundleManifest
		if protocol.DecodeOne(raw, &m) != nil || m.Schema != 1 || !extensionID.MatchString(m.ID) || !extensionVersion.MatchString(m.Version) || !slices.Contains(recognitionKinds, m.Kind) || m.Name == "" || !extensionID.MatchString(m.Screen) || len(m.Components) == 0 {
			return nil, errors.New("Manifeste de regroupement invalide.")
		}
		seen := map[string]bool{}
		for _, c := range m.Components {
			if !slices.Contains(extractionKinds, c.Slot) || seen[c.Slot] || c.Composition != "exclusive" {
				return nil, errors.New("Composant de regroupement invalide.")
			}
			seen[c.Slot] = true
		}
		for _, old := range result {
			if old.ID == m.ID || old.Kind == m.Kind {
				return nil, errors.New("Regroupement en double.")
			}
		}
		result = append(result, m)
	}
	return result, nil
}
func bundleSets(ms []mediaBundleManifest) []mediaSet {
	out := []mediaSet{}
	for _, m := range ms {
		s := mediaSet{ID: m.Kind, Name: m.Name}
		for _, c := range m.Components {
			s.Slots = append(s.Slots, c.Slot)
		}
		out = append(out, s)
	}
	return out
}

type bundleScreen struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Checksum string            `json:"checksum"`
	Enabled  bool              `json:"enabled"`
	Manifest extensionManifest `json:"-"`
}

func bundleScreens(ctx context.Context, tx *sql.Tx) ([]bundleScreen, error) {
	rows, e := tx.QueryContext(ctx, "SELECT id,manifest,checksum,enabled FROM extensions ORDER BY id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []bundleScreen{}
	for rows.Next() {
		var s bundleScreen
		var raw string
		if e = rows.Scan(&s.ID, &raw, &s.Checksum, &s.Enabled); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(raw), &s.Manifest); e != nil {
			return nil, e
		}
		if s.Manifest.API != 3 {
			continue
		}
		s.Name = s.Manifest.Name
		s.Version = s.Manifest.Version
		out = append(out, s)
	}
	return out, rows.Err()
}
func ownsBundleScreen(s bundleScreen, kind string) bool {
	for _, p := range s.Manifest.Pages {
		if p.Screen == kind {
			return true
		}
	}
	return false
}

// Adoption records only what already exists. It neither installs nor enables a component.
func migrateMediaBundles(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 30 {
		return nil
	}
	ctx := context.Background()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`CREATE TABLE IF NOT EXISTS media_bundles(id TEXT PRIMARY KEY,manifest TEXT NOT NULL);CREATE TABLE IF NOT EXISTS media_bundle_screens(kind TEXT PRIMARY KEY,id TEXT NOT NULL);`); e != nil {
		return e
	}
	ms, e := mediaBundleDefinitions()
	if e != nil {
		return e
	}
	screens, e := bundleScreens(ctx, tx)
	if e != nil {
		return e
	}
	for _, m := range ms {
		screen := m.Screen
		var present bool
		for _, s := range screens {
			if s.ID == m.Screen {
				present = true
			}
			if s.Enabled && ownsBundleScreen(s, m.Kind) {
				screen = s.ID
				present = true
			}
		}
		var n int
		if e = tx.QueryRow("SELECT count(*) FROM recognition_packages WHERE kind=? OR kind=?", m.Kind, "catalogue-"+m.Kind).Scan(&n); e != nil {
			return e
		}
		if present || n > 0 {
			raw, _ := json.Marshal(m)
			if _, e = tx.Exec("INSERT OR IGNORE INTO media_bundles VALUES(?,?)", m.ID, string(raw)); e != nil {
				return e
			}
		}
		if _, e = tx.Exec("INSERT OR IGNORE INTO media_bundle_screens VALUES(?,?)", m.Kind, screen); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=30"); e != nil {
		return e
	}
	return tx.Commit()
}

// Enabling a screen explicitly also chooses it for its media family. Disabling
// or deleting it keeps that choice; preparing a bundle never silently replaces it.
func (c *catalog) setUIScreenState(ctx context.Context, p extensionPackage, enabled bool) error {
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE extensions SET enabled=? WHERE id=?", enabled, p.Manifest.ID); e != nil {
		return e
	}
	if enabled {
		for _, page := range p.Manifest.Pages {
			if slices.Contains(recognitionKinds, page.Screen) {
				if _, e = tx.ExecContext(ctx, "INSERT INTO media_bundle_screens VALUES(?,?) ON CONFLICT(kind) DO UPDATE SET id=excluded.id", page.Screen, p.Manifest.ID); e != nil {
					return e
				}
			}
		}
	}
	return tx.Commit()
}

type mediaBundlePlan struct {
	mediaSetPlan
	Manifest         mediaBundleManifest  `json:"manifest"`
	InstalledVersion string               `json:"installedVersion"`
	ScreenID         string               `json:"screenId"`
	ScreenState      string               `json:"screenState"`
	State            string               `json:"state"`
	Snapshot         string               `json:"snapshot"`
	Archive          *bundleArchiveRecord `json:"archive,omitempty"`
}

func planBundle(ctx context.Context, tx *sql.Tx, ms []mediaBundleManifest, m mediaBundleManifest, op string, packages []recognitionPackageView, selected map[string]string, revision int64, safe bool, screens []bundleScreen) (mediaBundlePlan, error) {
	base, e := planMediaSet(bundleSets(ms), m.Kind, op, packages, selected, revision, safe)
	p := mediaBundlePlan{mediaSetPlan: base, Manifest: m, State: "partial"}
	if e != nil {
		return p, e
	}
	var stored string
	e = tx.QueryRowContext(ctx, "SELECT manifest FROM media_bundles WHERE id=?", m.ID).Scan(&stored)
	if e != nil && e != sql.ErrNoRows {
		return p, e
	}
	if stored != "" {
		var old mediaBundleManifest
		if json.Unmarshal([]byte(stored), &old) != nil {
			return p, errors.New("Regroupement enregistré illisible.")
		}
		p.InstalledVersion = old.Version
	}
	p.ScreenID = m.Screen
	e = tx.QueryRowContext(ctx, "SELECT id FROM media_bundle_screens WHERE kind=?", m.Kind).Scan(&p.ScreenID)
	if e != nil && e != sql.ErrNoRows {
		return p, e
	}
	var chosen *bundleScreen
	for i := range screens {
		if screens[i].ID == p.ScreenID {
			chosen = &screens[i]
		}
	}
	p.ScreenState = "absent"
	row := mediaSetStep{Slot: "screen", ID: p.ScreenID, Name: "Écran · " + m.Name}
	if chosen != nil {
		row.Name = chosen.Name
		row.Version = chosen.Version
		p.ScreenState = "disabled"
		if chosen.Enabled {
			p.ScreenState = "enabled"
		}
		if !ownsBundleScreen(*chosen, m.Kind) {
			p.Blockers = append(p.Blockers, "Le paquet d’écran choisi ne fournit plus cette rubrique.")
		}
	}
	add := func(action string) { s := row; s.Operation = action; p.Steps = append(p.Steps, s) }
	if op == "prepare" {
		if chosen == nil {
			if p.ScreenID != m.Screen {
				p.Blockers = append(p.Blockers, "L’écran de remplacement choisi est absent. Réinstallez-le ou activez explicitement un autre écran.")
			} else {
				add("install")
				add("enable")
			}
		} else if !chosen.Enabled {
			add("enable")
		} else {
			row.Operation = "ready"
			p.Kept = append(p.Kept, row)
		}
		claims := []string{m.Kind}
		if chosen != nil {
			claims = nil
			for _, page := range chosen.Manifest.Pages {
				if page.Screen != "" {
					claims = append(claims, page.Screen)
				}
			}
		}
		for _, s := range screens {
			if !s.Enabled || s.ID == p.ScreenID {
				continue
			}
			for _, claim := range claims {
				if ownsBundleScreen(s, claim) {
					p.Blockers = append(p.Blockers, "Un autre écran fournit déjà la rubrique "+claim+" : "+s.Name)
				}
			}
		}
	} else if chosen != nil {
		if len(chosen.Manifest.Pages) != 1 {
			p.Blockers = append(p.Blockers, "Cet écran fournit plusieurs pages. Gérez son retrait séparément pour préserver les autres rubriques.")
		} else if op == "remove" {
			add("remove")
		} else if chosen.Enabled {
			add("disable")
		} else {
			row.Operation = "unchanged"
			p.Kept = append(p.Kept, row)
		}
	}
	// Hash every relevant page state, manifest, and account grant. The registry
	// revision alone does not change when screens or their grants are edited.
	rows, e := tx.QueryContext(ctx, "SELECT user_id,extension_id FROM extension_grants ORDER BY user_id,extension_id")
	if e != nil {
		return p, e
	}
	grants := [][2]string{}
	for rows.Next() {
		var g [2]string
		if e = rows.Scan(&g[0], &g[1]); e != nil {
			rows.Close()
			return p, e
		}
		grants = append(grants, g)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, e
	}
	raw, _ := json.Marshal(struct {
		Stored  string
		Screens []bundleScreen
		Grants  [][2]string
	}{stored, screens, grants})
	p.Snapshot = recognitionHash(raw)
	if p.InstalledVersion == "" {
		p.State = "absent"
	} else {
		ready := p.ScreenState == "enabled"
		running := false
		for _, comp := range m.Components {
			found := false
			for _, v := range packages {
				if v.ID == selected[comp.Slot] && v.Installed && v.Enabled {
					if !comp.Shared {
						running = true
					}
					found = true
					break
				}
			}
			ready = ready && found
		}
		if ready {
			p.State = "ready"
		} else if !running && p.ScreenState != "enabled" {
			p.State = "disabled"
		}
	}
	p.Archive, e = readBundleRecord(ctx, tx, m.Kind)
	if e != nil {
		return p, e
	}
	p.Checksum = ""
	raw, _ = json.Marshal(p)
	p.Checksum = recognitionHash(raw)
	return p, nil
}
func applyBundleScreen(ctx context.Context, tx *sql.Tx, m mediaBundleManifest, step mediaSetStep) error {
	switch step.Operation {
	case "install":
		for _, p := range screenPackages() {
			if p.Manifest.ID == m.Screen && step.ID == m.Screen {
				var n int
				if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM extensions").Scan(&n); e != nil {
					return e
				}
				if n >= 32 {
					return errors.New("Limite de paquets d’interface atteinte.")
				}
				raw, _ := json.Marshal(p.Manifest)
				if _, e := tx.ExecContext(ctx, "INSERT INTO extensions VALUES(?,?,?,0)", p.Manifest.ID, string(raw), p.Checksum); e != nil {
					return e
				}
				_, e := tx.ExecContext(ctx, "INSERT INTO extension_grants(user_id,extension_id) SELECT id,? FROM accounts", p.Manifest.ID)
				return e
			}
		}
		return errors.New("Écran fourni introuvable.")
	case "enable", "disable":
		_, e := tx.ExecContext(ctx, "UPDATE extensions SET enabled=? WHERE id=?", step.Operation == "enable", step.ID)
		return e
	case "remove":
		_, e := tx.ExecContext(ctx, "DELETE FROM extensions WHERE id=?", step.ID)
		return e
	}
	return errors.New("Action d’écran invalide.")
}
func (c *catalog) mediaBundlePlans(ctx context.Context, id, op string, revision *int64, checksum string, apply bool) ([]mediaBundlePlan, error) {
	c.extensionMu.Lock()
	defer c.extensionMu.Unlock()
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	ms, e := mediaBundleDefinitions()
	if e != nil {
		return nil, e
	}
	pk, chosen, rev, e := c.mediaSetInputs(ctx, tx)
	if e != nil {
		return nil, e
	}
	if revision != nil && *revision != rev {
		return nil, registry.ErrConflict
	}
	screens, e := bundleScreens(ctx, tx)
	if e != nil {
		return nil, e
	}
	plans := []mediaBundlePlan{}
	for _, m := range ms {
		if id != "" && m.Kind != id {
			continue
		}
		p, e := planBundle(ctx, tx, ms, m, op, pk, chosen, rev, c.extensionSafeMode, screens)
		if e != nil {
			return nil, e
		}
		plans = append(plans, p)
	}
	if len(plans) == 0 {
		return nil, errors.New("Famille inconnue.")
	}
	if !apply {
		return plans, nil
	}
	p := plans[0]
	if id == "" || revision == nil || checksum == "" || checksum != p.Checksum {
		return nil, errors.New("Aperçu périmé. Vérifiez de nouveau les changements.")
	}
	if len(p.Blockers) > 0 {
		return nil, errors.New("Résolvez les conflits avant de continuer.")
	}
	for _, step := range p.Steps {
		if step.Slot == "screen" {
			e = applyBundleScreen(ctx, tx, p.Manifest, step)
		} else {
			_, e = changeRecognitionTx(ctx, tx, step.ID, step.Operation, rev)
			if e == nil {
				var s registry.Snapshot
				s, e = (registry.SQLStore{DB: tx}).Load(ctx)
				rev = s.Revision
			}
		}
		if e != nil {
			return nil, e
		}
	}
	if op == "prepare" {
		raw, _ := json.Marshal(p.Manifest)
		_, e = tx.ExecContext(ctx, "INSERT INTO media_bundles VALUES(?,?) ON CONFLICT(id) DO UPDATE SET manifest=excluded.manifest", p.Manifest.ID, string(raw))
	} else if op == "remove" {
		_, e = tx.ExecContext(ctx, "DELETE FROM media_bundles WHERE id=?", p.Manifest.ID)
		if e == nil {
			_, e = tx.ExecContext(ctx, "DELETE FROM media_bundle_archives WHERE kind=?", p.Manifest.Kind)
		}
	}
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	for _, step := range p.Steps {
		if step.Slot == "screen" {
			continue
		}
		if step.Operation == "remove" || step.Operation == "disable" {
			c.cancelRecognitionID(step.ID)
		}
		if step.Operation == "select" {
			for _, call := range c.recognition.calls {
				if call.kind == step.Slot {
					call.cancel(errRecognitionUnavailable)
				}
			}
		}
	}
	return plans, nil
}
func (c *catalog) mediaBundleRoutes(mux *secureMux) {
	c.bundleArchiveRoutes(mux)
	mux.HandleFunc("GET /api/v1/media-bundles", func(w http.ResponseWriter, r *http.Request) {
		p, e := c.mediaBundlePlans(r.Context(), "", "prepare", nil, "", false)
		if e != nil {
			problem(w, 500, "Regroupements indisponibles.")
			return
		}
		sendJSON(w, 200, map[string]any{"bundles": p})
	})
	for _, action := range []string{"preview", "apply"} {
		mux.HandleFunc("POST /api/v1/media-bundles/{id}/"+action, func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Operation string `json:"operation"`
				Revision  *int64 `json:"revision"`
				Checksum  string `json:"checksum"`
			}
			if !readJSON(w, r, &in) {
				return
			}
			if in.Revision == nil {
				problem(w, 400, "Actualisez les regroupements.")
				return
			}
			p, e := c.mediaBundlePlans(r.Context(), r.PathValue("id"), in.Operation, in.Revision, in.Checksum, action == "apply")
			if e != nil {
				problem(w, 409, "Aucune modification appliquée. "+e.Error())
				return
			}
			sendJSON(w, 200, p[0])
		})
	}
}
