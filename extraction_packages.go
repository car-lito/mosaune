package main

import (
	"context"
	"database/sql"
	"errors"
	tags "mosaune/extensionkit/capabilities/audiotags/v1"
	"mosaune/extensionkit/packageformat"
)

// Fold the existing audio-tag package into the shared executable lifecycle.
// No registry intent is changed, including disabled or already removed packages.
func migrateExtractionPackages(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 21 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO recognition_packages(id,kind,code) SELECT id,'audiotags',code FROM local_metadata_packages;
 INSERT INTO recognition_selection(kind,id) VALUES('audiotags','mosaune.music.tags') ON CONFLICT(kind) DO NOTHING;
 DROP TABLE local_metadata_packages;
 PRAGMA user_version=21;`); err != nil {
		return err
	}
	return tx.Commit()
}

// Returns the host slot, not an arbitrary module-provided database/table name.
func audioTagsImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	d := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != tags.Capability || m.Contributions[0].Target != "music" || len(d.Capabilities) != 1 || d.Capabilities[0] != tags.Capability || len(d.Dependencies) != 0 || len(d.Services) != 1 || d.Services[0].Name != tags.Service || len(d.Services[0].Operations) != 1 || d.Services[0].Operations[0] != "range" {
		return "", errors.New("Lecteur musical incompatible : seul l’accès limité au fichier en cours est accepté, sans réseau ni dépendance.")
	}
	return "audiotags", nil
}

func validateExtractionCode(ctx context.Context, p packageformat.Executable) error {
	kind, err := validateRecognitionImport(p)
	if err != nil {
		return err
	}
	if kind == "audiotags" {
		r, e := loadAudioTags(ctx, p.Manifest.Module, p.Code)
		if e != nil {
			return e
		}
		r.engine.Close()
		return nil
	}
	r, err := loadMediaRecognizer(ctx, p.Manifest.Module, p.Code)
	if err != nil {
		return err
	}
	r.close()
	return nil
}
