package main

import (
	"context"
	"database/sql"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
)

// Upgrade only the reserved factory package with intact stored bytes. Different
// Go toolchain patch versions can produce different official Wasm digests.
// Third-party choices,
// disabled state and removed packages are never replaced or recreated.
func migrateMusicRequests(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 35 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	ctx := context.Background()
	store := registry.SQLStore{DB: tx}
	s, e := store.Load(ctx)
	if e != nil {
		return e
	}
	entry, present := registryEntryFor(s, "mosaune.music.results")
	if present && entry.Descriptor.Version == "0.1.0" {
		var code []byte
		var manifest, archive []byte
		if e = tx.QueryRow("SELECT code,manifest,archive FROM recognition_packages WHERE id=? AND kind='music-results'", entry.Descriptor.ID).Scan(&code, &manifest, &archive); e != nil {
			return e
		}
		if len(manifest) == 0 && len(archive) == 0 && recognitionHash(code) == entry.CodeSHA256 {
			desc, next, e := modules.Lookup("music-results")
			if e != nil {
				return e
			}
			updated := registry.Entry{Descriptor: desc, CodeSHA256: recognitionHash(next)}
			if _, e = store.Commit(ctx, s.Revision, registry.Change{Operation: "update", ID: desc.ID, Entry: &updated}); e != nil {
				return e
			}
			if _, e = tx.Exec("UPDATE recognition_packages SET code=? WHERE id=?", next, desc.ID); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=35"); e != nil {
		return e
	}
	return tx.Commit()
}
