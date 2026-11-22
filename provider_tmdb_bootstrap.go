package main

import (
	"context"
	"database/sql"
	"mosaune/extensionkit/authorization"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
)

// This default belongs to the official composition, not the shared schema or
// the arbitrary-package installer. Its historical first-party consent is kept.
func installInitialTMDB(ctx context.Context, tx *sql.Tx) error {
	pack, err := packageformat.Parse(providerIdentifiersExample, trustedProviderKey(), version)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO provider_packages(id,archive,checksum) VALUES('tmdb',?,?)", pack.Archive, pack.Checksum); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO provider_runtime(id,enabled) VALUES('tmdb',1)"); err != nil {
		return err
	}
	if err = registerProviderPackage(ctx, tx, pack, true, false); err != nil {
		return err
	}
	if err = registerProviderSettings(ctx, configuration.SQLStore{DB: tx}, pack.Manifest.Settings); err != nil {
		return err
	}
	store := authorization.SQLStore{DB: tx}
	_, err = store.Read(ctx, tmdbConfigurationOwner)
	newOwner := err == sql.ErrNoRows
	if err != nil && !newOwner {
		return err
	}
	if err = store.Register(ctx, pack.Manifest.Descriptor); err != nil {
		return err
	}
	if newOwner {
		state, e := store.Read(ctx, tmdbConfigurationOwner)
		if e != nil {
			return e
		}
		return store.Apply(ctx, tmdbConfigurationOwner, authorization.Patch{Revision: &state.Revision, Granted: state.Requested})
	}
	return nil
}
