package main

import (
	"context"
	"database/sql"
	"mosaune/extensionkit/packageformat"
)

// Migrate the former native choice once. Provider disable and service denials
// are independent and remain unchanged. Existing signed packages are retained.
func migrateTMDBModuleOnly(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 28 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO provider_runtime(id,enabled) VALUES('tmdb',1) ON CONFLICT(id) DO UPDATE SET enabled=1"); err != nil {
		return err
	}
	if err = setRegisteredProviderWanted(ctx, tx, true); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=28"); err != nil {
		return err
	}
	return tx.Commit()
}

// Provision once, through the same signed archive and registry as later installs.
// A missing, paused module in v28 may mean an earlier removal: never resurrect it.
func migrateTMDBInstalledPackage(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 29 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists, wanted bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM provider_packages WHERE id='tmdb')").Scan(&exists); err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, "SELECT enabled FROM provider_runtime WHERE id='tmdb'").Scan(&wanted)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if !exists && wanted {
		pack, e := packageformat.Parse(providerCatalogueExample, trustedProviderKey(), version)
		if e != nil {
			return e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO provider_packages(id,archive,checksum) VALUES('tmdb',?,?)", pack.Archive, pack.Checksum); err != nil {
			return err
		}
		if err = registerProviderPackage(ctx, tx, pack, true, false); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=29"); err != nil {
		return err
	}
	return tx.Commit()
}
