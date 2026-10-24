package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

func migrateRecognitionAlternatives(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 19 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`ALTER TABLE recognition_packages RENAME TO recognition_packages_old;
CREATE TABLE recognition_packages(id TEXT PRIMARY KEY,kind TEXT NOT NULL,code BLOB NOT NULL,manifest BLOB NOT NULL DEFAULT '',archive BLOB NOT NULL DEFAULT '',previous BLOB NOT NULL DEFAULT '',previous_version TEXT NOT NULL DEFAULT '');
INSERT INTO recognition_packages(id,kind,code) SELECT id,kind,code FROM recognition_packages_old;
DROP TABLE recognition_packages_old;
CREATE TABLE recognition_selection(kind TEXT PRIMARY KEY,id TEXT NOT NULL);`)
	if err != nil {
		return err
	}
	// Remember the existing choice, including an absent factory package. Never
	// substitute a different installed module when the selected one disappears.
	for _, kind := range recognitionKinds {
		id, err := recognitionID(kind)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO recognition_selection(kind,id) VALUES(?,?)`, kind, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`PRAGMA user_version=19`); err != nil {
		return err
	}
	return tx.Commit()
}

func validateRecognitionImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	d := m.Module
	// Reserve factory identities. A local author's label is not an endorsement.
	if strings.HasPrefix(d.ID, "mosaune.") || len(m.Contributions) != 1 || m.Contributions[0].Role != recognition.Capability || !slices.Contains(recognitionKinds, m.Contributions[0].Target) || len(d.Capabilities) != 1 || d.Capabilities[0] != recognition.Capability || len(d.Services) != 0 || len(d.Dependencies) != 0 {
		return "", errors.New("Ce serveur accepte ici les reconnaisseurs films, séries ou musique, sans service ni dépendance. Les identifiants mosaune.* sont réservés.")
	}
	return m.Contributions[0].Target, nil
}

// Preview is static: untrusted code only executes after the administrator's
// installation confirmation, inside the same bounded engine used by scans.
func (c *catalog) previewRecognition(ctx context.Context, p packageformat.Executable) (registry.Plan, error) {
	kind, err := validateRecognitionImport(p)
	if err != nil {
		return registry.Plan{}, err
	}
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		return registry.Plan{}, err
	}
	op := "install"
	if _, ok := registryEntryFor(s, p.Manifest.Module.ID); ok {
		op = "update"
		var oldKind string
		if err = c.db.QueryRowContext(ctx, `SELECT kind FROM recognition_packages WHERE id=?`, p.Manifest.Module.ID).Scan(&oldKind); err != nil || oldKind != kind {
			return registry.Plan{}, errors.New("Une mise à jour doit conserver le rôle et le type de média du module installé.")
		}
	}
	e := registry.Entry{Descriptor: p.Manifest.Module, CodeSHA256: p.Manifest.CodeSHA256}
	return registry.Preview(s, registry.Change{Operation: op, ID: e.Descriptor.ID, Entry: &e})
}

func (c *catalog) importRecognition(ctx context.Context, raw []byte, checksum string, expected int64) error {
	if c.extensionSafeMode {
		return errors.New("Import indisponible en mode de secours.")
	}
	p, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		return err
	}
	if p.Checksum != checksum {
		return errors.New("Le fichier a changé. Vérifiez-le de nouveau.")
	}
	if _, err = validateRecognitionImport(p); err != nil {
		return err
	}
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	// Reject stale/invalid intent before spending resources on the executable.
	plan, err := c.previewRecognition(ctx, p)
	if err != nil {
		return err
	}
	if plan.BeforeRevision != expected {
		return registry.ErrConflict
	}
	reader, err := loadMediaRecognizer(ctx, p.Manifest.Module, p.Code)
	if err != nil {
		return err
	}
	reader.close()
	return c.commitRecognitionImport(ctx, p, raw, expected, false)
}

// Caller holds the lifecycle lock. Bytes, manifest, previous version and
// registry intent share one SQL transaction; no active session is invalidated
// until that transaction succeeds.
func (c *catalog) commitRecognitionImport(ctx context.Context, p packageformat.Executable, raw []byte, expected int64, restore bool) error {
	kind, err := validateRecognitionImport(p)
	if err != nil {
		return err
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := registry.SQLStore{DB: tx}
	s, err := store.Load(ctx)
	if err != nil {
		return err
	}
	if s.Revision != expected {
		return registry.ErrConflict
	}
	op := "install"
	previousVersion := ""
	if old, ok := registryEntryFor(s, p.Manifest.Module.ID); ok {
		previousVersion = old.Descriptor.Version
		op = "update"
		if restore {
			op = "restore"
		}
		var oldKind string
		if err = tx.QueryRowContext(ctx, `SELECT kind FROM recognition_packages WHERE id=?`, p.Manifest.Module.ID).Scan(&oldKind); err != nil || oldKind != kind {
			return errors.New("Type de média incompatible avec le module installé.")
		}
	} else if restore {
		return errors.New("Module désinstallé.")
	}
	e := registry.Entry{Descriptor: p.Manifest.Module, CodeSHA256: p.Manifest.CodeSHA256}
	if _, err = store.Commit(ctx, expected, registry.Change{Operation: op, ID: e.Descriptor.ID, Entry: &e}); err != nil {
		return err
	}
	m, err := json.Marshal(p.Manifest)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO recognition_packages(id,kind,code,manifest,archive,previous_version) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET previous=recognition_packages.archive,previous_version=excluded.previous_version,code=excluded.code,manifest=excluded.manifest,archive=excluded.archive`, e.Descriptor.ID, kind, p.Code, m, raw, previousVersion)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	c.cancelRecognitionID(e.Descriptor.ID)
	return nil
}

func (c *catalog) restoreRecognition(ctx context.Context, id string, expected int64) error {
	if c.extensionSafeMode {
		return errors.New("Restauration indisponible en mode de secours.")
	}
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	var raw []byte
	if err := c.db.QueryRowContext(ctx, `SELECT previous FROM recognition_packages WHERE id=?`, id).Scan(&raw); err != nil || len(raw) == 0 {
		return errors.New("Version précédente absente.")
	}
	p, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		return err
	}
	if p.Manifest.Module.ID != id {
		return errors.New("Version précédente incohérente.")
	}
	reader, err := loadMediaRecognizer(ctx, p.Manifest.Module, p.Code)
	if err != nil {
		return err
	}
	reader.close()
	return c.commitRecognitionImport(ctx, p, raw, expected, true)
}

func (c *catalog) recognitionImportRoutes(mux *secureMux) {
	read := func(w http.ResponseWriter, r *http.Request) ([]byte, packageformat.Executable, bool) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, packageformat.MaxArchive))
		if err != nil {
			problem(w, 400, "Archive illisible (9 Mio maximum).")
			return nil, packageformat.Executable{}, false
		}
		p, err := packageformat.ParseExecutable(raw, version)
		if err != nil {
			problem(w, 400, err.Error())
			return nil, p, false
		}
		return raw, p, true
	}
	mux.HandleFunc("POST /api/v1/recognition-packages/preview", func(w http.ResponseWriter, r *http.Request) {
		_, p, ok := read(w, r)
		if !ok {
			return
		}
		plan, err := c.previewRecognition(r.Context(), p)
		if err != nil {
			problem(w, 409, err.Error())
			return
		}
		sendJSON(w, 200, map[string]any{"package": p, "revision": plan.BeforeRevision, "source": "local-unverified"})
	})
	mux.HandleFunc("POST /api/v1/recognition-packages/import", func(w http.ResponseWriter, r *http.Request) {
		raw, p, ok := read(w, r)
		if !ok {
			return
		}
		revision, err := strconv.ParseInt(r.Header.Get("X-Mosaune-Registry-Revision"), 10, 64)
		if err != nil || revision < 0 {
			problem(w, 400, "Révision requise. Vérifiez le paquet avant de l’installer.")
			return
		}
		if err = c.importRecognition(r.Context(), raw, r.Header.Get("X-Mosaune-Package-Checksum"), revision); err != nil {
			problem(w, 409, fmt.Sprintf("Installation non appliquée : %s", err))
			return
		}
		sendJSON(w, 201, map[string]any{"id": p.Manifest.Module.ID})
	})
}
