package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/acquisitionconfig"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
)

func (v arrConfig) document() a.Configuration {
	return a.Configuration{ID: v.ID, URL: v.URL, Key: v.Key, Configured: v.Configured, Enabled: v.Enabled, Profile: v.Profile, Root: v.Root, Library: v.Library, Revision: v.revision}
}
func (c *catalog) configFromDocument(v a.Configuration) arrConfig {
	return arrConfig{owner: c, ID: v.ID, URL: v.URL, Key: v.Key, Configured: v.Configured, Enabled: v.Enabled, Profile: v.Profile, Root: v.Root, Library: v.Library, revision: v.Revision}
}
func (c *catalog) arrConfig(ctx context.Context, id string) (arrConfig, error) {
	v, err := (a.ConfigurationStore{DB: c.db}).Load(ctx, id)
	return c.configFromDocument(v), err
}
func (c *catalog) configInput(ctx context.Context, v arrConfig) (arrConfig, error) {
	old, err := c.arrConfig(ctx, v.ID)
	if err != nil {
		return v, err
	}
	next, err := a.PrepareConfiguration(v.document(), old.document())
	return c.configFromDocument(next), err
}
func (c *catalog) saveAcquisitionConfiguration(ctx context.Context, v *arrConfig) error {
	if err := (a.ConfigurationStore{DB: c.db}).Save(ctx, v.document()); err != nil {
		return err
	}
	v.revision++
	return nil
}

// Move both connectors atomically. A conflicting destination or malformed legacy
// value rolls back the migration; never silently normalize or discard a secret.
func migrateAcquisitionConfiguration(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 46 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx := context.Background()
	store := a.ConfigurationStore{DB: tx}
	for _, id := range []string{"radarr", "sonarr"} {
		if err = store.Register(ctx, id); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,data FROM acquisition_settings")
	if err != nil {
		return err
	}
	var legacy []a.Configuration
	for rows.Next() {
		var id, raw string
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		var v a.Configuration
		if len(raw) > 256<<10 || protocol.DecodeOne([]byte(raw), &v) != nil || v.ID != id {
			rows.Close()
			return configuration.ErrStored
		}
		if _, err = a.ConfigurationOwner(id); err != nil {
			rows.Close()
			return err
		}
		legacy = append(legacy, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, v := range legacy {
		b, err := store.Bound(v.ID)
		if err != nil {
			return err
		}
		state, err := b.Read(ctx)
		if err != nil {
			return err
		}
		if len(state.Values) > 0 || state.SecretsConfigured["key"] {
			current, err := store.Load(ctx, v.ID)
			if err != nil {
				return err
			}
			current.Revision = 0
			v.Revision = 0
			current.Configured = false
			v.Configured = false
			x, _ := json.Marshal(current)
			y, _ := json.Marshal(v)
			if string(x) != string(y) {
				return errors.New("Deux configurations différentes empêchent la migration du connecteur. Les réglages sont conservés.")
			}
		} else {
			if err = store.Save(ctx, v); err != nil {
				return err
			}
		}
	}
	// Keep an empty legacy table for old schema migrations; it is never read by
	// runtime configuration code, and no duplicate credential is retained there.
	if _, err = tx.Exec("DELETE FROM acquisition_settings; PRAGMA user_version=46;"); err != nil {
		return err
	}
	return tx.Commit()
}
