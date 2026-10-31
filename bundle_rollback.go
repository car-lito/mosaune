package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
)

// One durable undo point per family, containing programs, never preferences,
// account grants or media. Individual updates cannot overwrite this snapshot.
type bundleUndoMember struct {
	Step           bundleArchiveStep
	Before         []byte
	BeforeChecksum string
	AfterChecksum  string
}
type bundleUndo struct {
	Before        bundleArchiveRecord
	AfterChecksum string
	Members       []bundleUndoMember
}

func migrateBundleUndo(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 32 {
		return nil
	}
	_, e := db.Exec("CREATE TABLE IF NOT EXISTS media_bundle_undo(kind TEXT PRIMARY KEY,previous_version TEXT NOT NULL,current_checksum TEXT NOT NULL,record BLOB NOT NULL); PRAGMA user_version=32;")
	return e
}
func readBundleUndo(ctx context.Context, tx *sql.Tx, kind string) (*bundleUndo, error) {
	var raw []byte
	e := tx.QueryRowContext(ctx, "SELECT record FROM media_bundle_undo WHERE kind=?", kind).Scan(&raw)
	if e == sql.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var u bundleUndo
	e = json.Unmarshal(raw, &u)
	return &u, e
}

// Called before updating, within the import transaction. Same-version repairs
// retain the undo point; fresh installations have no previous family to restore.
func saveBundleUndo(ctx context.Context, tx *sql.Tx, p bundleArchive, plan bundleArchivePlan) error {
	old, e := readBundleRecord(ctx, tx, p.Manifest.Kind)
	if e != nil {
		return e
	}
	if old == nil || old.Checksum == p.Checksum {
		return nil
	}
	u := bundleUndo{Before: *old, AfterChecksum: p.Checksum}
	for _, s := range plan.Steps {
		if s.Operation != "update" && s.Operation != "install" {
			continue
		}
		m := bundleUndoMember{Step: s}
		if s.Slot == "screen" {
			m.AfterChecksum = p.Screen.Checksum
			e = tx.QueryRowContext(ctx, "SELECT manifest,checksum FROM extensions WHERE id=?", s.ID).Scan(&m.Before, &m.BeforeChecksum)
		} else {
			m.AfterChecksum = recognitionHash(p.Raw[s.Slot])
			e = tx.QueryRowContext(ctx, "SELECT archive FROM recognition_packages WHERE id=?", s.ID).Scan(&m.Before)
		}
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		u.Members = append(u.Members, m)
	}
	raw, e := json.Marshal(u)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO media_bundle_undo VALUES(?,?,?,?) ON CONFLICT(kind) DO UPDATE SET previous_version=excluded.previous_version,current_checksum=excluded.current_checksum,record=excluded.record", p.Manifest.Kind, old.Manifest.Version, p.Checksum, raw)
	return e
}
func planBundleUndo(ctx context.Context, tx *sql.Tx, p mediaBundlePlan, chosen map[string]string, safe bool) (mediaBundlePlan, error) {
	p.Operation = "restore"
	p.Steps = []mediaSetStep{}
	p.Kept = []mediaSetStep{}
	p.Blockers = []string{}
	if safe {
		p.Blockers = append(p.Blockers, "Quittez le mode de secours pour restaurer une famille.")
	}
	u, e := readBundleUndo(ctx, tx, p.Manifest.Kind)
	if e != nil {
		return p, e
	}
	if u == nil || p.Archive == nil {
		p.Blockers = append(p.Blockers, "Aucune mise à jour restaurable. L’historique commence avec les mises à jour effectuées depuis Mosaune 0.0.49.")
	} else if u.AfterChecksum != p.Archive.Checksum {
		p.Blockers = append(p.Blockers, "L’historique ne correspond plus à cette famille.")
	} else {
		p.PreviousVersion = u.Before.Manifest.Version
		for _, m := range u.Members {
			s := mediaSetStep{Slot: m.Step.Slot, ID: m.Step.ID, Name: m.Step.Name, Version: u.Before.Manifest.Version, Operation: "restore", Shared: m.Step.Shared}
			selected := chosen[s.Slot]
			if s.Slot == "screen" {
				selected = p.ScreenID
			}
			// Shared modules and user-selected alternatives are not family-owned.
			if s.Shared || selected != s.ID {
				s.Operation = "unchanged"
				p.Kept = append(p.Kept, s)
				continue
			}
			if len(m.Before) == 0 {
				p.Blockers = append(p.Blockers, "Version antérieure absente pour "+s.Name+". Ce composant a été réinstallé pendant la mise à jour.")
				continue
			}
			var currentHash string
			if s.Slot == "screen" {
				e = tx.QueryRowContext(ctx, "SELECT checksum FROM extensions WHERE id=?", s.ID).Scan(&currentHash)
				var manifest extensionManifest
				if json.Unmarshal(m.Before, &manifest) != nil || manifest.ID != s.ID {
					return p, errors.New("Écran sauvegardé invalide.")
				}
				s.Version = manifest.Version
			} else {
				var raw []byte
				e = tx.QueryRowContext(ctx, "SELECT archive FROM recognition_packages WHERE id=?", s.ID).Scan(&raw)
				currentHash = recognitionHash(raw)
				old, err := packageformat.ParseExecutable(m.Before, version)
				if err != nil {
					return p, err
				}
				slot, err := validateRecognitionImport(old)
				if err != nil || slot != s.Slot || old.Manifest.Module.ID != s.ID {
					return p, errors.New("Programme sauvegardé incompatible.")
				}
				s.Version = old.Manifest.Module.Version
			}
			if e != nil && e != sql.ErrNoRows {
				return p, e
			}
			if e == sql.ErrNoRows || currentHash != m.AfterChecksum {
				p.Blockers = append(p.Blockers, s.Name+" a été modifié ou retiré séparément. Aucun composant ne sera écrasé.")
			}
			p.Steps = append(p.Steps, s)
		}
	}
	// Include the prior base checksum (selections, permissions, registry) and the
	// saved point, so another update or permission change invalidates the preview.
	raw, _ := json.Marshal(struct {
		Base string
		Undo *bundleUndo
	}{p.Checksum, u})
	p.Snapshot = recognitionHash(raw)
	p.Checksum = ""
	raw, _ = json.Marshal(p)
	p.Checksum = recognitionHash(raw)
	return p, nil
}
func applyBundleUndo(ctx context.Context, tx *sql.Tx, p mediaBundlePlan) error {
	u, e := readBundleUndo(ctx, tx, p.Manifest.Kind)
	if e != nil {
		return e
	}
	if u == nil {
		return errors.New("Historique absent.")
	}
	store := registry.SQLStore{DB: tx}
	rev := p.Revision
	for _, s := range p.Steps {
		var saved *bundleUndoMember
		for i := range u.Members {
			if u.Members[i].Step.ID == s.ID {
				saved = &u.Members[i]
				break
			}
		}
		if saved == nil {
			return errors.New("Composant sauvegardé absent.")
		}
		if s.Slot == "screen" {
			_, e = tx.ExecContext(ctx, "UPDATE extensions SET manifest=?,checksum=? WHERE id=?", string(saved.Before), saved.BeforeChecksum, s.ID)
		} else {
			program, err := packageformat.ParseExecutable(saved.Before, version)
			if err != nil {
				return err
			}
			if err = validateExtractionCode(ctx, program); err != nil {
				return err
			}
			entry := registry.Entry{Descriptor: program.Manifest.Module, CodeSHA256: program.Manifest.CodeSHA256}
			result, err := store.Commit(ctx, rev, registry.Change{Operation: "restore", ID: s.ID, Entry: &entry})
			if err != nil {
				return err
			}
			rev = result.After.Revision
			manifest, _ := json.Marshal(program.Manifest)
			_, e = tx.ExecContext(ctx, `UPDATE recognition_packages SET previous=archive,previous_version=?,code=?,manifest=?,archive=? WHERE id=?`, saved.Step.Version, program.Code, manifest, saved.Before, s.ID)
		}
		if e != nil {
			return e
		}
	}
	raw, _ := json.Marshal(u.Before)
	if _, e = tx.ExecContext(ctx, "UPDATE media_bundle_archives SET record=? WHERE kind=?", string(raw), p.Manifest.Kind); e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "DELETE FROM media_bundle_undo WHERE kind=?", p.Manifest.Kind)
	return e
}
