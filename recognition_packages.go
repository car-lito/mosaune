package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"net/http"
	"sync"
)

var errRecognitionUnavailable = errors.New("Reconnaissance désactivée ou désinstallée. Réactivez son extension dans les paramètres pour analyser ce type de média. Votre catalogue est conservé.")

// Factory offers share the lifecycle with administrator-imported packages.
var recognitionKinds = []string{"films", "series", "music"}

// Shared executable-package lifecycle; the legacy table/API names stay compatible.
var extractionKinds = []string{"films", "series", "music", "audiotags", "catalogue-films", "catalogue-series", "catalogue-music"}

// Legacy kind paths still address their factory package; all other operations
// use the stable module ID, never a publisher label supplied by an archive.
func recognitionID(id string) (string, error) {
	for _, kind := range extractionKinds {
		if id == kind {
			d, err := modules.Descriptor(kind)
			return d.ID, err
		}
	}
	if (protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{"media.recognition.v1"}}).Validate() != nil {
		return "", errors.New("Identifiant de module invalide.")
	}
	return id, nil
}

func recognitionHash(code []byte) string { h := sha256.Sum256(code); return hex.EncodeToString(h[:]) }

func installRecognition(ctx context.Context, tx *sql.Tx, kind string, revision int64) (registry.Snapshot, error) {
	d, code, err := modules.Lookup(kind)
	if err != nil {
		return registry.Snapshot{}, err
	}
	entry := registry.Entry{Descriptor: d, CodeSHA256: recognitionHash(code)}
	plan, err := (registry.SQLStore{DB: tx}).Commit(ctx, revision, registry.Change{Operation: "install", ID: d.ID, Entry: &entry})
	if err != nil {
		return registry.Snapshot{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO recognition_packages(kind,id,code) VALUES(?,?,?)`, kind, d.ID, code)
	return plan.After, err
}

func migrateRecognitionPackages(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= 18 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE recognition_packages(kind TEXT PRIMARY KEY,id TEXT NOT NULL UNIQUE,code BLOB NOT NULL);`); err != nil {
		return err
	}
	store := registry.SQLStore{DB: tx}
	for _, kind := range recognitionKinds {
		s, e := store.Load(ctx)
		if e != nil {
			return e
		}
		s, e = installRecognition(ctx, tx, kind, s.Revision)
		if e != nil {
			return e
		}
		d, _ := modules.Descriptor(kind)
		if e = registryWanted(ctx, store, s, d.ID, true); e != nil {
			return e
		}
	}
	if _, err = tx.ExecContext(ctx, `PRAGMA user_version=18`); err != nil {
		return err
	}
	return tx.Commit()
}

// The mutex orders state changes, session starts and catalogue commits. It is
// acquired before SQL, never from inside an already open SQL transaction.
type recognitionLifecycle struct {
	mu    sync.Mutex
	next  uint64
	calls map[uint64]recognitionCall
}
type recognitionCall struct {
	kind   string
	id     string
	cancel context.CancelCauseFunc
}

type recognitionPackageView struct {
	Kind            string `json:"kind"`
	ID              string `json:"id"`
	Version         string `json:"version"`
	Installed       bool   `json:"installed"`
	Enabled         bool   `json:"enabled"`
	Name            string `json:"name"`
	Publisher       string `json:"publisher"`
	Selected        bool   `json:"selected"`
	Factory         bool   `json:"factory"`
	PreviousVersion string `json:"previousVersion"`
}

func (c *catalog) recognitionPackages(ctx context.Context) ([]recognitionPackageView, int64, error) {
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		return nil, 0, err
	}
	list := []recognitionPackageView{}
	selected := map[string]string{}
	rows, err := c.db.QueryContext(ctx, `SELECT kind,id FROM recognition_selection`)
	if err != nil {
		return nil, 0, err
	}
	for rows.Next() {
		var kind, id string
		if err = rows.Scan(&kind, &id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		selected[kind] = id
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	seen := map[string]bool{}
	names := map[string]string{"films": "Reconnaissance des films", "series": "Reconnaissance des séries", "music": "Reconnaissance musicale", "audiotags": "Informations des fichiers musicaux", "catalogue-films": "Catalogue des films", "catalogue-series": "Catalogue des séries", "catalogue-music": "Catalogue musical"}
	for _, kind := range extractionKinds {
		d, err := modules.Descriptor(kind)
		if err != nil {
			return nil, 0, err
		}
		entry, installed := registryEntryFor(s, d.ID)
		v := d.Version
		if installed {
			v = entry.Descriptor.Version
		}
		list = append(list, recognitionPackageView{Kind: kind, ID: d.ID, Version: v, Installed: installed, Enabled: installed && entry.Enabled, Name: names[kind], Publisher: "Mosaune", Factory: true, Selected: selected[kind] == d.ID})
		seen[d.ID] = true
	}
	rows, err = c.db.QueryContext(ctx, `SELECT id,kind,manifest,previous_version FROM recognition_packages ORDER BY id`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, kind string
		var raw []byte
		var previousVersion string
		if err = rows.Scan(&id, &kind, &raw, &previousVersion); err != nil {
			return nil, 0, err
		}
		if seen[id] {
			continue
		}
		var m packageformat.ExecutableManifest
		if json.Unmarshal(raw, &m) != nil {
			return nil, 0, errors.New("Manifeste installé illisible.")
		}
		e, ok := registryEntryFor(s, id)
		if !ok {
			return nil, 0, errors.New("Module absent du registre.")
		}
		v := recognitionPackageView{Kind: kind, ID: id, Name: m.Name, Publisher: m.Publisher, Version: e.Descriptor.Version, Installed: true, Enabled: e.Enabled, Selected: selected[kind] == id}
		v.PreviousVersion = previousVersion
		list = append(list, v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, s.Revision, nil
}

func (c *catalog) beginRecognition(ctx context.Context, kind string) (context.Context, func(), protocol.Descriptor, []byte, error) {
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	var empty protocol.Descriptor
	if c.extensionSafeMode {
		return ctx, func() {}, empty, nil, errRecognitionUnavailable
	}
	if err := ctx.Err(); err != nil {
		return ctx, func() {}, empty, nil, err
	}
	var selected string
	if err := c.db.QueryRowContext(ctx, `SELECT id FROM recognition_selection WHERE kind=?`, kind).Scan(&selected); err != nil {
		return ctx, func() {}, empty, nil, errRecognitionUnavailable
	}
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		return ctx, func() {}, empty, nil, err
	}
	entry, ok := registryEntryFor(s, selected)
	if !ok || !entry.Enabled {
		return ctx, func() {}, empty, nil, errRecognitionUnavailable
	}
	var id string
	var code []byte
	err = c.db.QueryRowContext(ctx, `SELECT id,code FROM recognition_packages WHERE id=? AND kind=?`, selected, kind).Scan(&id, &code)
	if err != nil || id != entry.Descriptor.ID || recognitionHash(code) != entry.CodeSHA256 {
		return ctx, func() {}, empty, nil, errors.New("Paquet de reconnaissance absent ou endommagé. Réinstallez-le ; le catalogue est conservé.")
	}
	call, cancel := context.WithCancelCause(ctx)
	c.recognition.next++
	key := c.recognition.next
	if c.recognition.calls == nil {
		c.recognition.calls = map[uint64]recognitionCall{}
	}
	c.recognition.calls[key] = recognitionCall{kind: kind, id: id, cancel: cancel}
	done := func() {
		c.recognition.mu.Lock()
		delete(c.recognition.calls, key)
		c.recognition.mu.Unlock()
		cancel(context.Canceled)
	}
	return call, done, entry.Descriptor, code, nil
}

func (c *catalog) recognitionCommit(ctx context.Context) (func(), error) {
	c.recognition.mu.Lock()
	if err := context.Cause(ctx); err != nil {
		c.recognition.mu.Unlock()
		return func() {}, err
	}
	return c.recognition.mu.Unlock, nil
}

func (c *catalog) changeRecognition(ctx context.Context, kind, operation string, expected int64) error {
	id, err := recognitionID(kind)
	if err != nil {
		return errors.New("Module de reconnaissance inconnu.")
	}
	if operation == "restore" {
		return c.restoreRecognition(ctx, id, expected)
	}
	if operation != "install" && operation != "enable" && operation != "disable" && operation != "remove" && operation != "select" {
		return errors.New("Action inconnue.")
	}
	if c.extensionSafeMode && (operation == "enable" || operation == "install" || operation == "select") {
		return errors.New("Quittez le mode de secours pour installer ou activer un module.")
	}
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := registry.SQLStore{DB: tx}
	if operation == "install" {
		factoryKind := ""
		for _, k := range extractionKinds {
			d, _ := modules.Descriptor(k)
			if d.ID == id {
				factoryKind = k
			}
		}
		if factoryKind == "" {
			return errors.New("Réimportez le fichier pour réinstaller ce module externe.")
		}
		_, err = installRecognition(ctx, tx, factoryKind, expected)
	} else {
		if err = tx.QueryRowContext(ctx, `SELECT kind FROM recognition_packages WHERE id=?`, id).Scan(&kind); err != nil {
			return err
		}
		if operation == "select" {
			s, e := store.Load(ctx)
			if e != nil {
				return e
			}
			if s.Revision != expected {
				return registry.ErrConflict
			}
			entry, ok := registryEntryFor(s, id)
			if !ok || !entry.Enabled {
				return errors.New("Activez le module avant de le choisir pour les scans.")
			}
			// Advance the shared revision for a choice change too: an old tab
			// cannot subsequently overwrite this decision with a stale revision.
			_, err = store.Commit(ctx, expected, registry.Change{Operation: "enable", ID: id})
			if err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO recognition_selection(kind,id) VALUES(?,?) ON CONFLICT(kind) DO UPDATE SET id=excluded.id`, kind, id)
			}
		} else {
			_, err = store.Commit(ctx, expected, registry.Change{Operation: operation, ID: id})
		}
		if err == nil && operation == "remove" {
			_, err = tx.ExecContext(ctx, `DELETE FROM recognition_packages WHERE id=?`, id)
		}
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	// A failed/stale mutation never cancels a session. Old sessions remain
	// cancelled even if this module is immediately installed or enabled again.
	if operation == "disable" || operation == "remove" {
		c.cancelRecognitionID(id)
	}
	if operation == "select" {
		for _, call := range c.recognition.calls {
			if call.kind == kind {
				call.cancel(errRecognitionUnavailable)
			}
		}
	}
	return nil
}

func (c *catalog) cancelRecognitionID(id string) {
	for _, call := range c.recognition.calls {
		if call.id == id {
			call.cancel(errRecognitionUnavailable)
		}
	}
}

func (c *catalog) recognitionPackageRoutes(mux *secureMux) {
	c.recognitionImportRoutes(mux)
	c.audioTagsRoutes(mux)
	mux.HandleFunc("GET /api/v1/recognition-packages", func(w http.ResponseWriter, r *http.Request) {
		list, revision, err := c.recognitionPackages(r.Context())
		if err != nil {
			problem(w, 500, "Modules indisponibles.")
			return
		}
		sendJSON(w, 200, map[string]any{"packages": list, "revision": revision, "safeMode": c.extensionSafeMode})
	})
	mux.HandleFunc("POST /api/v1/recognition-packages/{kind}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Operation string `json:"operation"`
			Revision  *int64 `json:"revision"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if in.Revision == nil {
			problem(w, 400, "Actualisez la liste des extensions avant cette action.")
			return
		}
		if err := c.changeRecognition(r.Context(), r.PathValue("kind"), in.Operation, *in.Revision); err != nil {
			problem(w, 409, "Modification non appliquée. Actualisez la liste. "+err.Error())
			return
		}
		sendJSON(w, 200, map[string]bool{"ok": true})
	})
}
