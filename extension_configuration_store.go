package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	bundled "mosaune/extensionkit/providers/tmdb"
)

const tmdbConfigurationOwner = "mosaune.tmdb-search"

func tmdbConfiguration(db configuration.Connection) configuration.Bound {
	b, _ := (configuration.SQLStore{DB: db}).For(tmdbConfigurationOwner)
	return b
}
func migrateExtensionConfiguration(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 16 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := configuration.SQLStore{DB: tx}
	if err = store.Initialize(ctx); err != nil {
		return err
	}
	schema := bundled.Settings()
	var archive []byte
	if err = tx.QueryRow(`SELECT archive FROM provider_packages WHERE id='tmdb'`).Scan(&archive); err != nil && err != sql.ErrNoRows {
		return err
	}
	if len(archive) > 0 {
		if pack, e := packageformat.Parse(archive, trustedProviderKey(), version); e == nil {
			schema = pack.Manifest.Settings
		}
	}
	if err = store.Register(ctx, tmdbConfigurationOwner, schema); err != nil {
		return err
	}
	var old string
	if err = tx.QueryRow(`SELECT secret FROM provider_settings WHERE id='tmdb'`).Scan(&old); err != nil && err != sql.ErrNoRows {
		return err
	}
	if old != "" {
		bound := tmdbConfiguration(tx)
		existing, e := bound.Secret(ctx, "apiToken")
		if e != nil {
			return e
		}
		if existing != "" && existing != old {
			return errors.New("Deux configurations TMDB différentes empêchent la migration. Les anciennes clés sont conservées.")
		}
		raw, _ := json.Marshal(old)
		if err = bound.Apply(ctx, configuration.Patch{Values: map[string]json.RawMessage{"apiToken": raw}}); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`DELETE FROM provider_settings WHERE id='tmdb';PRAGMA user_version=16;`); err != nil {
		return err
	}
	return tx.Commit()
}
func (c *catalog) setProviderCredential(ctx context.Context, id, secret string) error {
	if id == "tmdb" {
		patch := configuration.Patch{}
		if secret == "" {
			patch.ClearSecrets = []string{"apiToken"}
		} else {
			raw, _ := json.Marshal(secret)
			patch.Values = map[string]json.RawMessage{"apiToken": raw}
		}
		return tmdbConfiguration(c.db).Apply(ctx, patch)
	}
	if secret == "" {
		_, err := c.db.ExecContext(ctx, `DELETE FROM provider_settings WHERE id=?`, id)
		return err
	}
	_, err := c.db.ExecContext(ctx, `INSERT INTO provider_settings(id,secret) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET secret=excluded.secret`, id, secret)
	return err
}
