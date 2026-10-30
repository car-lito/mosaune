package main

import (
	"context"
	"database/sql"
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
