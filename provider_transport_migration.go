package main

import (
	"bytes"
	"context"
	"database/sql"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
)

// Only the exact known factory archives are upgraded. Removals, unknown signed
// replacements, disabled runtime/provider and denied services stay untouched.
func migrateProviderTransport(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 37 {
		return nil
	}
	ctx := context.Background()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var raw []byte
	var sum string
	e = tx.QueryRowContext(ctx, "SELECT archive,checksum FROM provider_packages WHERE id='tmdb' AND previous_checksum=''").Scan(&raw, &sum)
	if e != nil && e != sql.ErrNoRows {
		return e
	}
	known := bytes.Equal(raw, providerDiscoveryExample)
	if known {
		old, err := packageformat.Parse(raw, trustedProviderKey(), version)
		if err != nil {
			return err
		}
		if sum != old.Checksum {
			return sql.ErrNoRows
		}
		pack, err := packageformat.Parse(providerTransportExample, trustedProviderKey(), version)
		if err != nil {
			return err
		}
		var enabled bool
		if err = tx.QueryRowContext(ctx, "SELECT enabled FROM provider_runtime WHERE id='tmdb'").Scan(&enabled); err != nil {
			return err
		}
		if err = registerProviderPackage(ctx, tx, pack, enabled, false); err != nil {
			return err
		}
		if err = registerProviderSettings(ctx, configuration.SQLStore{DB: tx}, pack.Manifest.Settings); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE provider_packages SET archive=?,checksum=?,previous=?,previous_checksum=? WHERE id='tmdb'", pack.Archive, pack.Checksum, raw, sum); err != nil {
			return err
		}
	}
	if _, e = tx.ExecContext(ctx, "PRAGMA user_version=37"); e != nil {
		return e
	}
	return tx.Commit()
}
