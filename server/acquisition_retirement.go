package main

import (
	"context"
	"database/sql"
	a "mosaune/extensionkit/acquisitionconfig"
	"mosaune/extensionkit/configuration"
)

// The installed package set is authoritative, not the active selection: an
// inactive replacement still owns the shared capability configuration.
func reconcileAcquisitionConfiguration(ctx context.Context, tx *sql.Tx) error {
	if err := (configuration.SQLStore{DB: tx}).Initialize(ctx); err != nil {
		return err
	}
	var installed int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM recognition_packages WHERE kind='acquisition'").Scan(&installed); err != nil {
		return err
	}
	return (a.ConfigurationStore{DB: tx}).Reconcile(ctx, installed > 0)
}
func migrateAcquisitionRetirement(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 50 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx := context.Background()
	if err = (configuration.SQLStore{DB: tx}).Initialize(ctx); err != nil {
		return err
	}
	if err = reconcileAcquisitionConfiguration(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.Exec("PRAGMA user_version=50"); err != nil {
		return err
	}
	return tx.Commit()
}
