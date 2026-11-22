package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mosaune/extensionkit/configuration"
)

// Persist the initial composition before schema migration. An interrupted first
// launch must resume that choice, not the defaults of a later executable.
func prepareProviderBootstrap(ctx context.Context, db *sql.DB, oldVersion int, factories []metadataProviderFactory) error {
	if err := validateProviderFactories(factories); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS provider_bootstrap(id INTEGER PRIMARY KEY CHECK(id=1),providers TEXT NOT NULL,completed INTEGER NOT NULL CHECK(completed IN(0,1)))`); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM provider_bootstrap WHERE id=1)").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		var used bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM libraries) OR EXISTS(SELECT 1 FROM media)").Scan(&used); err != nil {
			return err
		}
		completed := oldVersion != 0 || used
		ids := []string{}
		if !completed {
			for _, f := range factories {
				if f.InstallDefault != nil {
					ids = append(ids, f.ID)
				}
			}
		}
		raw, _ := json.Marshal(ids)
		if _, err = tx.ExecContext(ctx, "INSERT INTO provider_bootstrap(id,providers,completed) VALUES(1,?,?)", string(raw), completed); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Historical databases without this table retain their original migration
// path. Only a recorded, unfinished first launch defers provider provisioning.
func pendingProviderBootstrap(ctx context.Context, db configuration.Connection) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='provider_bootstrap')").Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	var completed bool
	err := db.QueryRowContext(ctx, "SELECT completed FROM provider_bootstrap WHERE id=1").Scan(&completed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return !completed, err
}

func deferProviderSeed(ctx context.Context, tx *sql.Tx, schemaVersion int) (bool, error) {
	pending, err := pendingProviderBootstrap(ctx, tx)
	if err != nil || !pending {
		return false, err
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion))
	return true, err
}

// All first-install callbacks and their completion marker commit together. A
// completed composition is never replayed, even after a provider is removed.
func finishProviderBootstrap(ctx context.Context, db *sql.DB, factories []metadataProviderFactory) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw string
	var completed bool
	if err = tx.QueryRowContext(ctx, "SELECT providers,completed FROM provider_bootstrap WHERE id=1").Scan(&raw, &completed); err != nil {
		return err
	}
	if !completed {
		var ids []string
		if len(raw) > 65536 || json.Unmarshal([]byte(raw), &ids) != nil || ids == nil || len(ids) > 128 {
			return errors.New("Composition initiale des fournisseurs illisible.")
		}
		byID := map[string]metadataProviderFactory{}
		for _, f := range factories {
			byID[f.ID] = f
		}
		seen := map[string]bool{}
		for _, id := range ids {
			f, ok := byID[id]
			if !ok || f.InstallDefault == nil || seen[id] {
				return errors.New("Un fournisseur prévu pour la première installation est indisponible.")
			}
			seen[id] = true
		}
		for _, id := range ids {
			if err = byID[id].InstallDefault(ctx, tx); err != nil {
				return fmt.Errorf("Installation initiale du fournisseur %s : %w", id, err)
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE provider_bootstrap SET completed=1 WHERE id=1"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=51"); err != nil {
		return err
	}
	return tx.Commit()
}
