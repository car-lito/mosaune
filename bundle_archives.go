package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"net/http"
	"slices"
	"strings"
)

const bundleArchiveLimit = 32 << 20
const bundleExpandedLimit = 64 << 20

type bundleArchiveMember struct {
	Slot   string `json:"slot"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}
type bundleArchiveManifest struct {
	Schema     int                   `json:"schema"`
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Publisher  string                `json:"publisher"`
	Version    string                `json:"version"`
	MinCore    string                `json:"minCore"`
	Kind       string                `json:"kind"`
	Components []bundleArchiveMember `json:"components"`
	Screen     bundleArchiveMember   `json:"screen"`
}
type bundleArchive struct {
	Manifest bundleArchiveManifest               `json:"manifest"`
	Checksum string                              `json:"checksum"`
	Modules  map[string]packageformat.Executable `json:"-"`
	Raw      map[string][]byte                   `json:"-"`
	Screen   extensionPackage                    `json:"-"`
}
type bundleArchiveRecord struct {
	Manifest bundleArchiveManifest `json:"manifest"`
	Checksum string                `json:"checksum"`
	IDs      map[string]string     `json:"ids"`
	ScreenID string                `json:"screenId"`
}

func migrateBundleArchives(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 31 {
		return nil
	}
	_, e := db.Exec("CREATE TABLE IF NOT EXISTS media_bundle_archives(kind TEXT PRIMARY KEY,record TEXT NOT NULL); PRAGMA user_version=31;")
	return e
}
func readBundleRecord(ctx context.Context, tx *sql.Tx, kind string) (*bundleArchiveRecord, error) {
	var raw string
	e := tx.QueryRowContext(ctx, "SELECT record FROM media_bundle_archives WHERE kind=?", kind).Scan(&raw)
	if e == sql.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var r bundleArchiveRecord
	e = json.Unmarshal([]byte(raw), &r)
	return &r, e
}
func bundledFamily(kind string) (mediaBundleManifest, error) {
	ms, e := mediaBundleDefinitions()
	if e != nil {
		return mediaBundleManifest{}, e
	}
	for _, m := range ms {
		if m.Kind == kind {
			return m, nil
		}
	}
	return mediaBundleManifest{}, errors.New("Famille non prise en charge.")
}
func parseBundleArchive(raw []byte) (bundleArchive, error) {
	p := bundleArchive{Modules: map[string]packageformat.Executable{}, Raw: map[string][]byte{}}
	fail := errors.New("Archive de famille invalide, incomplète ou trop volumineuse.")
	if len(raw) > bundleArchiveLimit {
		return p, fail
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil || len(z.File) < 3 || len(z.File) > 9 {
		return p, fail
	}
	files := map[string][]byte{}
	total := 0
	for _, f := range z.File {
		limit := packageformat.MaxArchive
		if f.Name == "family.json" {
			limit = 64 << 10
		}
		if !f.Mode().IsRegular() || f.Name == "" || strings.ContainsAny(f.Name, "/\\:") || strings.Contains(f.Name, "..") || files[f.Name] != nil || f.UncompressedSize64 > uint64(limit) {
			return p, fail
		}
		r, e := f.Open()
		if e != nil {
			return p, fail
		}
		b, e := io.ReadAll(io.LimitReader(r, int64(limit+1)))
		r.Close()
		total += len(b)
		if e != nil || len(b) > limit || total > bundleExpandedLimit {
			return p, fail
		}
		files[f.Name] = b
	}
	if protocol.DecodeOne(files["family.json"], &p.Manifest) != nil {
		return p, fail
	}
	m := p.Manifest
	if m.Schema != 1 || !extensionID.MatchString(m.ID) || strings.HasPrefix(m.ID, "mosaune.") || strings.HasPrefix(m.ID, "org.mosaune.media.") || strings.TrimSpace(m.Name) == "" || len(m.Name) > 80 || strings.TrimSpace(m.Publisher) == "" || len(m.Publisher) > 80 || !extensionVersion.MatchString(m.Version) || !extensionVersion.MatchString(m.MinCore) || compareExtensionVersions(m.MinCore, version) > 0 {
		return p, fail
	}
	family, e := bundledFamily(m.Kind)
	if e != nil || len(m.Components) != len(family.Components) || len(files) != len(m.Components)+2 {
		return p, fail
	}
	used := map[string]bool{"family.json": true}
	slots := map[string]bool{}
	ids := map[string]bool{}
	expanded := 0
	member := func(ref bundleArchiveMember) ([]byte, error) {
		b, ok := files[ref.File]
		if !ok || used[ref.File] || packageformat.Hash(b) != ref.SHA256 {
			return nil, fail
		}
		used[ref.File] = true
		return b, nil
	}
	for _, ref := range m.Components {
		if slots[ref.Slot] || !slices.ContainsFunc(family.Components, func(c bundleComponent) bool { return c.Slot == ref.Slot }) {
			return p, fail
		}
		slots[ref.Slot] = true
		b, e := member(ref)
		if e != nil {
			return p, e
		}
		program, e := packageformat.ParseExecutable(b, version)
		if e != nil {
			return p, e
		}
		kind, e := validateRecognitionImport(program)
		if e != nil || kind != ref.Slot || ids[program.Manifest.Module.ID] {
			return p, fail
		}
		ids[program.Manifest.Module.ID] = true
		expanded += len(program.Code)
		if expanded > bundleExpandedLimit {
			return p, fail
		}
		p.Modules[ref.Slot] = program
		p.Raw[ref.Slot] = b
	}
	b, e := member(m.Screen)
	if e != nil || m.Screen.Slot != "screen" {
		return p, fail
	}
	p.Screen, e = parseExtensionPackage(b)
	if e != nil {
		return p, e
	}
	s := p.Screen.Manifest
	if s.API != 3 || len(s.Pages) != 1 || s.Pages[0].Screen != m.Kind || ids[s.ID] || strings.HasPrefix(s.ID, "org.mosaune.screens.") {
		return p, fail
	}
	p.Checksum = packageformat.Hash(raw)
	return p, nil
}

type bundleArchiveStep struct {
	Slot       string `json:"slot"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Operation  string `json:"operation"`
	Select     bool   `json:"select"`
	PreviousID string `json:"previousId"`
	Shared     bool   `json:"shared"`
}
type bundleArchivePlan struct {
	Manifest        bundleArchiveManifest `json:"manifest"`
	ArchiveChecksum string                `json:"archiveChecksum"`
	PreviousVersion string                `json:"previousVersion"`
	Revision        int64                 `json:"revision"`
	Steps           []bundleArchiveStep   `json:"steps"`
	Checksum        string                `json:"checksum"`
	Snapshot        string                `json:"snapshot"`
}

// Caller holds both lifecycle locks. Preview reads only; code validation occurs at import.
func planBundleArchive(ctx context.Context, c *catalog, tx *sql.Tx, p bundleArchive) (bundleArchivePlan, error) {
	out := bundleArchivePlan{Manifest: p.Manifest, ArchiveChecksum: p.Checksum, Steps: []bundleArchiveStep{}}
	if c.extensionSafeMode {
		return out, errors.New("Import indisponible en mode de secours.")
	}
	ms, e := mediaBundleDefinitions()
	if e != nil {
		return out, e
	}
	m, e := bundledFamily(p.Manifest.Kind)
	if e != nil {
		return out, e
	}
	pk, chosen, rev, e := c.mediaSetInputs(ctx, tx)
	if e != nil {
		return out, e
	}
	out.Revision = rev
	screens, e := bundleScreens(ctx, tx)
	if e != nil {
		return out, e
	}
	base, e := planBundle(ctx, tx, ms, m, "prepare", pk, chosen, rev, false, screens)
	if e != nil {
		return out, e
	}
	previous, e := readBundleRecord(ctx, tx, m.Kind)
	if e != nil {
		return out, e
	}
	if previous != nil {
		out.PreviousVersion = previous.Manifest.Version
		if previous.Manifest.ID != p.Manifest.ID {
			return out, errors.New("Une autre archive gère cette famille. Retirez la famille avant de la remplacer.")
		}
		cmp := compareExtensionVersions(p.Manifest.Version, previous.Manifest.Version)
		if cmp < 0 || cmp == 0 && p.Checksum != previous.Checksum {
			return out, errors.New("Version de famille plus récente requise ; une même version doit garder exactement le même contenu.")
		}
		for slot, program := range p.Modules {
			if previous.IDs[slot] != program.Manifest.Module.ID {
				return out, errors.New("Une mise à jour doit conserver les identifiants et rôles de ses composants.")
			}
		}
		if previous.ScreenID != p.Screen.Manifest.ID {
			return out, errors.New("Une mise à jour doit conserver l’identité de son écran.")
		}
	}
	snap, _ := json.Marshal(struct {
		Base     string
		Previous *bundleArchiveRecord
	}{base.Checksum, previous})
	out.Snapshot = recognitionHash(snap)
	registryState, e := (registry.SQLStore{DB: tx}).Load(ctx)
	if e != nil {
		return out, e
	}
	for _, comp := range m.Components {
		program := p.Modules[comp.Slot]
		d := program.Manifest.Module
		factory, _ := modules.Descriptor(comp.Slot)
		step := bundleArchiveStep{Slot: comp.Slot, ID: d.ID, Name: program.Manifest.Name, Version: d.Version, PreviousID: chosen[comp.Slot], Shared: comp.Shared}
		// An explicit alternative, even missing, is never replaced. Shared defaults
		// already installed are reused rather than duplicated for each family.
		existingSelected := slices.ContainsFunc(pk, func(v recognitionPackageView) bool { return v.ID == chosen[comp.Slot] && v.Installed })
		if previous != nil && chosen[comp.Slot] != previous.IDs[comp.Slot] || chosen[comp.Slot] != "" && chosen[comp.Slot] != factory.ID && chosen[comp.Slot] != d.ID || comp.Shared && chosen[comp.Slot] == factory.ID && existingSelected {
			step.Operation = "keep"
			out.Steps = append(out.Steps, step)
			continue
		}
		step.Select = chosen[comp.Slot] != d.ID
		old, exists := registryEntryFor(registryState, d.ID)
		step.Operation = "install"
		if exists {
			var oldKind string
			if e = tx.QueryRowContext(ctx, "SELECT kind FROM recognition_packages WHERE id=?", d.ID).Scan(&oldKind); e != nil || oldKind != comp.Slot {
				return out, errors.New("Identité de composant déjà utilisée pour une autre fonction.")
			}
			cmp := compareExtensionVersions(d.Version, old.Descriptor.Version)
			if cmp < 0 {
				return out, errors.New("Le paquet ferait reculer un composant installé : " + d.ID)
			}
			step.Operation = "update"
			if cmp == 0 {
				if old.CodeSHA256 != program.Manifest.CodeSHA256 {
					return out, errors.New("Même version avec un programme différent : " + d.ID)
				}
				oldJSON, _ := json.Marshal(old.Descriptor)
				newJSON, _ := json.Marshal(d)
				if !bytes.Equal(oldJSON, newJSON) {
					return out, errors.New("Même version avec un contrat différent.")
				}
				step.Operation = "reuse"
			}
		}
		out.Steps = append(out.Steps, step)
	}
	screen := p.Screen
	step := bundleArchiveStep{Slot: "screen", ID: screen.Manifest.ID, Name: screen.Manifest.Name, Version: screen.Manifest.Version, PreviousID: base.ScreenID}
	if previous != nil && base.ScreenID != previous.ScreenID || base.ScreenID != m.Screen && base.ScreenID != step.ID {
		step.Operation = "keep"
	} else {
		step.Select = base.ScreenID != step.ID
		step.Operation = "install"
		var oldRaw, oldHash string
		e = tx.QueryRowContext(ctx, "SELECT manifest,checksum FROM extensions WHERE id=?", step.ID).Scan(&oldRaw, &oldHash)
		if e != nil && e != sql.ErrNoRows {
			return out, e
		}
		if e == nil {
			var old extensionManifest
			if json.Unmarshal([]byte(oldRaw), &old) != nil {
				return out, errors.New("Écran installé illisible.")
			}
			if old.API != 3 || len(old.Pages) != 1 || old.Pages[0].Screen != m.Kind {
				return out, errors.New("Identité d’écran déjà utilisée pour une autre rubrique.")
			}
			cmp := compareExtensionVersions(step.Version, old.Version)
			if cmp < 0 || cmp == 0 && oldHash != screen.Checksum {
				return out, errors.New("Version d’écran plus récente requise.")
			}
			step.Operation = "update"
			if cmp == 0 {
				step.Operation = "reuse"
			}
		} else {
			var n int
			if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM extensions").Scan(&n); e != nil {
				return out, e
			}
			if n >= 32 {
				return out, errors.New("Limite de paquets d’interface atteinte.")
			}
		}
	}
	out.Steps = append(out.Steps, step)
	raw, _ := json.Marshal(out)
	out.Checksum = recognitionHash(raw)
	return out, nil
}
func (c *catalog) processBundleArchive(ctx context.Context, p bundleArchive, expected string, apply bool) (bundleArchivePlan, error) {
	c.extensionMu.Lock()
	defer c.extensionMu.Unlock()
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return bundleArchivePlan{}, e
	}
	defer tx.Rollback()
	plan, e := planBundleArchive(ctx, c, tx, p)
	if e != nil || !apply {
		return plan, e
	}
	if expected == "" || plan.Checksum != expected {
		return plan, errors.New("Aperçu périmé. Vérifiez de nouveau cette archive.")
	}
	// Bounded WASI validation, never run while previewing or parsing an archive.
	for _, step := range plan.Steps {
		if step.Slot != "screen" && (step.Operation == "install" || step.Operation == "update") {
			if e = validateExtractionCode(ctx, p.Modules[step.Slot]); e != nil {
				return plan, e
			}
		}
	}
	store := registry.SQLStore{DB: tx}
	rev := plan.Revision
	for _, step := range plan.Steps {
		if step.Operation == "keep" {
			continue
		}
		if step.Slot == "screen" {
			raw, _ := json.Marshal(p.Screen.Manifest)
			if step.Operation == "install" {
				_, e = tx.ExecContext(ctx, "INSERT INTO extensions VALUES(?,?,?,0)", step.ID, string(raw), p.Screen.Checksum)
			} else if step.Operation == "update" {
				_, e = tx.ExecContext(ctx, "UPDATE extensions SET manifest=?,checksum=? WHERE id=?", string(raw), p.Screen.Checksum, step.ID)
			}
			if e != nil {
				return plan, e
			}
			if step.Select {
				// New screens start paused and without new account grants. The supplied
				// screen is paused explicitly; all of its grants remain unchanged.
				if _, e = tx.ExecContext(ctx, "UPDATE extensions SET enabled=0 WHERE id=?", step.PreviousID); e != nil {
					return plan, e
				}
				if _, e = tx.ExecContext(ctx, "INSERT INTO media_bundle_screens VALUES(?,?) ON CONFLICT(kind) DO UPDATE SET id=excluded.id", p.Manifest.Kind, step.ID); e != nil {
					return plan, e
				}
			}
			continue
		}
		program := p.Modules[step.Slot]
		if step.Operation == "install" || step.Operation == "update" {
			before, e := store.Load(ctx)
			if e != nil {
				return plan, e
			}
			previousVersion := ""
			if old, ok := registryEntryFor(before, step.ID); ok {
				previousVersion = old.Descriptor.Version
			}
			entry := registry.Entry{Descriptor: program.Manifest.Module, CodeSHA256: program.Manifest.CodeSHA256}
			result, e := store.Commit(ctx, rev, registry.Change{Operation: step.Operation, ID: step.ID, Entry: &entry})
			if e != nil {
				return plan, e
			}
			rev = result.After.Revision
			raw, _ := json.Marshal(program.Manifest)
			if _, e = tx.ExecContext(ctx, `INSERT INTO recognition_packages(id,kind,code,manifest,archive,previous_version) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET previous=recognition_packages.archive,previous_version=excluded.previous_version,code=excluded.code,manifest=excluded.manifest,archive=excluded.archive`, step.ID, step.Slot, program.Code, raw, p.Raw[step.Slot], previousVersion); e != nil {
				return plan, e
			}
		}
		if step.Select {
			if _, e = tx.ExecContext(ctx, "INSERT INTO recognition_selection VALUES(?,?) ON CONFLICT(kind) DO UPDATE SET id=excluded.id", step.Slot, step.ID); e != nil {
				return plan, e
			}
		}
	}
	record := bundleArchiveRecord{Manifest: p.Manifest, Checksum: p.Checksum, IDs: map[string]string{}, ScreenID: p.Screen.Manifest.ID}
	for slot, program := range p.Modules {
		record.IDs[slot] = program.Manifest.Module.ID
	}
	raw, _ := json.Marshal(record)
	if _, e = tx.ExecContext(ctx, "INSERT INTO media_bundle_archives VALUES(?,?) ON CONFLICT(kind) DO UPDATE SET record=excluded.record", p.Manifest.Kind, string(raw)); e != nil {
		return plan, e
	}
	m, _ := bundledFamily(p.Manifest.Kind)
	raw, _ = json.Marshal(m)
	if _, e = tx.ExecContext(ctx, "INSERT OR IGNORE INTO media_bundles VALUES(?,?)", m.ID, string(raw)); e != nil {
		return plan, e
	}
	if e = tx.Commit(); e != nil {
		return plan, e
	}
	for _, step := range plan.Steps {
		if step.Operation == "update" && step.Slot != "screen" {
			c.cancelRecognitionID(step.ID)
		}
		if step.Select && step.Slot != "screen" {
			for _, call := range c.recognition.calls {
				if call.kind == step.Slot {
					call.cancel(errRecognitionUnavailable)
				}
			}
		}
	}
	return plan, nil
}
func (c *catalog) bundleArchiveRoutes(mux *secureMux) {
	for _, action := range []string{"preview", "import"} {
		mux.HandleFunc("POST /api/v1/media-bundle-archives/"+action, func(w http.ResponseWriter, r *http.Request) {
			raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, bundleArchiveLimit))
			if e != nil {
				problem(w, 400, "Archive illisible (32 Mio maximum).")
				return
			}
			p, e := parseBundleArchive(raw)
			if e != nil {
				problem(w, 400, e.Error())
				return
			}
			plan, e := c.processBundleArchive(r.Context(), p, r.Header.Get("X-Mosaune-Plan-Checksum"), action == "import")
			if e != nil {
				problem(w, 409, "Aucune modification appliquée. "+e.Error())
				return
			}
			sendJSON(w, 200, plan)
		})
	}
	mux.HandleFunc("GET /api/v1/media-bundle-archives/example/{kind}", func(w http.ResponseWriter, r *http.Request) {
		release := r.URL.Query().Get("version")
		if release == "" {
			release = "1.0.0"
		}
		if release != "1.0.0" && release != "1.1.0" {
			problem(w, 400, "Version d’exemple inconnue.")
			return
		}
		raw, e := exampleBundleArchive(r.PathValue("kind"), release)
		if e != nil {
			problem(w, 400, "Famille inconnue.")
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="mosaune-family-%s-example.zip"`, r.PathValue("kind")))
		w.Write(raw)
	})
}

// A real multi-component archive for contributors; it uses distinct example
// identities and the existing contracts, not the reserved factory identities.
func exampleBundleArchive(kind, release string) ([]byte, error) {
	m, e := bundledFamily(kind)
	if e != nil {
		return nil, e
	}
	manifest := bundleArchiveManifest{Schema: 1, ID: "org.mosaune.examples.family." + kind, Name: "Famille " + m.Name + " · exemple", Publisher: "Mosaune · exemple local", Version: release, MinCore: "0.0.47", Kind: kind}
	files := map[string][]byte{}
	for _, comp := range m.Components {
		d, code, e := modules.Lookup(comp.Slot)
		if e != nil {
			return nil, e
		}
		d.ID = "org.mosaune.examples.component." + comp.Slot
		d.Version = release
		target := kind
		switch comp.Slot {
		case "detail-information", "detail-header", "detail-experience":
			target = "media"
		case "detail-series":
			target = "series"
		}
		em := packageformat.ExecutableManifest{Schema: 2, Name: mediaSlotName(comp.Slot), Publisher: manifest.Publisher, MinCore: "0.0.47", Module: d, Contributions: []packageformat.Contribution{{Role: d.Capabilities[0], Target: target}}}
		b, e := packageformat.CreateExecutable(em, code)
		if e != nil {
			return nil, e
		}
		file := comp.Slot + ".zip"
		files[file] = b
		manifest.Components = append(manifest.Components, bundleArchiveMember{Slot: comp.Slot, File: file, SHA256: packageformat.Hash(b)})
	}
	for _, screen := range screenPackages() {
		if screen.Manifest.ID == m.Screen {
			screen.Manifest.ID = manifest.ID + ".screen"
			screen.Manifest.Version = release
			screen.Manifest.Name = "Écran " + m.Name + " · exemple"
			screen.Manifest.MinCore = "0.0.47"
			files["screen.zip"] = screenArchive(screen)
		}
	}
	manifest.Screen = bundleArchiveMember{Slot: "screen", File: "screen.zip", SHA256: packageformat.Hash(files["screen.zip"])}
	raw, e := json.Marshal(manifest)
	if e != nil {
		return nil, e
	}
	files["family.json"] = raw
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		f, e := z.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = f.Write(files[name]); e != nil {
			return nil, e
		}
	}
	if e = z.Close(); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
