package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
)

const catalogSchemaVersion = 21

// TMDB packages and recognition packages share this registry. Other built-in
// functions/themes still have their existing lifecycles until extracted.
func migrateExtensionRegistry(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 15 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := registry.SQLStore{DB: tx}
	if err = store.Initialize(ctx); err != nil {
		return err
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT archive FROM provider_packages WHERE id='tmdb'`).Scan(&raw); err != nil && err != sql.ErrNoRows {
		return err
	}
	if len(raw) > 0 {
		// An unreadable package must not stop the whole server from starting.
		// Its existing package UI still reports it, without executing its code.
		if pack, parseErr := packageformat.Parse(raw, trustedProviderKey(), version); parseErr == nil {
			var wanted bool
			err = tx.QueryRowContext(ctx, `SELECT enabled FROM provider_runtime WHERE id='tmdb'`).Scan(&wanted)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err = registerProviderPackage(ctx, tx, pack, wanted, false); err != nil {
				return err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `PRAGMA user_version=15`); err != nil {
		return err
	}
	return tx.Commit()
}
func registryEntryFor(s registry.Snapshot, id string) (registry.Entry, bool) {
	for _, entry := range s.Entries {
		if entry.Descriptor.ID == id {
			return entry, true
		}
	}
	return registry.Entry{}, false
}

// Caller owns a SQL transaction shared with archive replacement. Registry and
// package bytes either commit together or both remain unchanged.
func registerProviderPackage(ctx context.Context, tx *sql.Tx, pack packageformat.Package, wanted, rollback bool) error {
	store := registry.SQLStore{DB: tx}
	s, err := store.Load(ctx)
	if err != nil {
		return err
	}
	entry := registry.Entry{Descriptor: pack.Manifest.Descriptor, CodeSHA256: pack.Manifest.CodeSHA256}
	old, exists := registryEntryFor(s, entry.Descriptor.ID)
	op := "install"
	if exists {
		op = "update"
		if rollback {
			op = "restore"
		}
	}
	// Idempotent legacy adoption is allowed only for identical authenticated data.
	a, _ := json.Marshal(old.Descriptor)
	b, _ := json.Marshal(entry.Descriptor)
	if !exists || old.CodeSHA256 != entry.CodeSHA256 || string(a) != string(b) {
		p, e := store.Commit(ctx, s.Revision, registry.Change{Operation: op, ID: entry.Descriptor.ID, Entry: &entry})
		if e != nil {
			return e
		}
		s = p.After
	}
	return registryWanted(ctx, store, s, entry.Descriptor.ID, wanted)
}
func registryWanted(ctx context.Context, store registry.SQLStore, s registry.Snapshot, id string, wanted bool) error {
	entry, ok := registryEntryFor(s, id)
	if !ok {
		return nil
	}
	if entry.Enabled == wanted {
		return nil
	}
	op := "disable"
	if wanted {
		op = "enable"
	}
	_, err := store.Commit(ctx, s.Revision, registry.Change{Operation: op, ID: id})
	return err
}
func setRegisteredProviderWanted(ctx context.Context, tx *sql.Tx, wanted bool) error {
	store := registry.SQLStore{DB: tx}
	s, err := store.Load(ctx)
	if err != nil {
		return err
	}
	return registryWanted(ctx, store, s, "mosaune.tmdb-search", wanted)
}
func removeRegisteredProvider(ctx context.Context, tx *sql.Tx) error {
	store := registry.SQLStore{DB: tx}
	s, err := store.Load(ctx)
	if err != nil {
		return err
	}
	if _, ok := registryEntryFor(s, "mosaune.tmdb-search"); !ok {
		return nil
	}
	_, err = store.Commit(ctx, s.Revision, registry.Change{Operation: "remove", ID: "mosaune.tmdb-search"})
	return err
}
func (p *tmdbProvider) registeredPackage(ctx context.Context, pack packageformat.Package) (registry.Entry, int64, error) {
	s, err := (registry.SQLStore{DB: p.db}).Load(ctx)
	if err != nil {
		return registry.Entry{}, 0, err
	}
	entry, ok := registryEntryFor(s, pack.Manifest.Descriptor.ID)
	a, _ := json.Marshal(entry.Descriptor)
	b, _ := json.Marshal(pack.Manifest.Descriptor)
	if !ok || entry.CodeSHA256 != pack.Manifest.CodeSHA256 || string(a) != string(b) {
		return registry.Entry{}, s.Revision, errors.New("Enregistrement de l’extension absent ou incohérent. Réinstallez une version vérifiée du paquet.")
	}
	return entry, s.Revision, nil
}
