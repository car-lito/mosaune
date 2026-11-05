package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"mosaune/extensionkit/registry"
)

func migrateAcquisitionExtension(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 39 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx := context.Background()
	store := registry.SQLStore{DB: tx}
	state, err := store.Load(ctx)
	if err != nil {
		return err
	}
	present, enabled := false, false
	for _, kind := range []string{"films", "series"} {
		var id string
		err = tx.QueryRow("SELECT id FROM recognition_selection WHERE kind=?", kind).Scan(&id)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if entry, ok := registryEntryFor(state, id); ok {
			present = true
			enabled = enabled || entry.Enabled
		}
	}
	for _, connector := range []string{"radarr", "sonarr"} {
		var raw string
		err = tx.QueryRow("SELECT data FROM acquisition_settings WHERE id=?", connector).Scan(&raw)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		var cfg arrConfig
		if err == nil {
			if err = json.Unmarshal([]byte(raw), &cfg); err != nil {
				return err
			}
			present = true
			enabled = enabled || cfg.Enabled
		}
	}
	id, err := recognitionID("acquisition")
	if err != nil {
		return err
	}
	var selected bool
	if err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM recognition_selection WHERE kind='acquisition')").Scan(&selected); err != nil {
		return err
	}
	_, installed := registryEntryFor(state, id)
	if present && !selected && !installed {
		state, err = installRecognition(ctx, tx, "acquisition", state.Revision)
		if err != nil {
			return err
		}
		if err = registryWanted(ctx, store, state, id, enabled); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("INSERT OR IGNORE INTO recognition_selection(kind,id) VALUES('acquisition',?)", id); err != nil {
		return err
	}
	for _, kind := range []string{"films", "series"} {
		next, err := bundledFamily(kind)
		if err != nil {
			return err
		}
		var raw string
		err = tx.QueryRow("SELECT manifest FROM media_bundles WHERE id=?", next.ID).Scan(&raw)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		var old mediaBundleManifest
		if err = json.Unmarshal([]byte(raw), &old); err != nil {
			return err
		}
		legacy := next
		legacy.Version = "1.0.0"
		legacy.Components = append([]bundleComponent(nil), next.Components[:len(next.Components)-1]...)
		oldJSON, _ := json.Marshal(old)
		legacyJSON, _ := json.Marshal(legacy)
		if string(oldJSON) == string(legacyJSON) {
			nextJSON, _ := json.Marshal(next)
			if _, err = tx.Exec("UPDATE media_bundles SET manifest=? WHERE id=?", string(nextJSON), next.ID); err != nil {
				return err
			}
		}
	}
	if _, err = tx.Exec("PRAGMA user_version=39"); err != nil {
		return err
	}
	return tx.Commit()
}
