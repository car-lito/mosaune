package main

import (
	"context"
	"database/sql"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
)

// Upgrade only installed factory versions. Removed/paused packages and explicit
// alternative selections remain untouched. New installations already have 0.2.
func migrateCatalogueSearch(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 33 {
		return nil
	}
	ctx := context.Background()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	store := registry.SQLStore{DB: tx}
	for _, kind := range catalogueKinds {
		d, code, e := modules.Lookup("catalogue-" + kind)
		if e != nil {
			return e
		}
		snapshot, e := store.Load(ctx)
		if e != nil {
			return e
		}
		old, ok := registryEntryFor(snapshot, d.ID)
		if !ok || old.Descriptor.Version != "0.1.0" {
			continue
		}
		var present bool
		if e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM recognition_packages WHERE id=?)", d.ID).Scan(&present); e != nil {
			return e
		}
		if !present {
			continue
		}
		entry := registry.Entry{Descriptor: d, CodeSHA256: recognitionHash(code)}
		if _, e = store.Commit(ctx, snapshot.Revision, registry.Change{Operation: "update", ID: d.ID, Entry: &entry}); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE recognition_packages SET code=? WHERE id=?", code, d.ID); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "PRAGMA user_version=33"); e != nil {
		return e
	}
	return tx.Commit()
}
