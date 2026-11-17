package main

import (
	"context"
	"database/sql"
	"errors"
	"mosaune/extensionkit/authorization"
	bundled "mosaune/extensionkit/providers/tmdb"
)

func (p *tmdbProvider) ServiceAccess() providerServiceAccess {
	return providerServiceAccess{db: p.db, owner: tmdbConfigurationOwner, lifecycle: &p.state, services: []extensionServiceView{{Name: "tmdb.api.v1", Label: "Consulter TMDB", Description: "Recherches, fiches et images en ligne de TMDB. La clé reste gérée par Mosaune. Retirer cet accès arrête aussi les recherches en cours ; les fiches et images déjà conservées restent disponibles.", Operations: []string{"get"}}}}
}
func (p *tmdbProvider) loadServiceAuthorization(ctx context.Context) error {
	return p.ServiceAccess().load(ctx)
}
func (p *tmdbProvider) setServiceAuthorization(ctx context.Context, patch authorization.Patch) error {
	return p.ServiceAccess().apply(ctx, patch)
}

func migrateExtensionAuthorizations(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 17 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	s := authorization.SQLStore{DB: tx}
	if err = s.Initialize(ctx); err != nil {
		return err
	}
	_, err = s.Read(ctx, tmdbConfigurationOwner)
	newOwner := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newOwner {
		return err
	}
	// TMDB is a first-party service already shipped by the host. Preserve its
	// historical default once; Register never grants access to a new module.
	// Native and Wasm implementations share the same authorization owner.
	if err = s.Register(ctx, bundled.Descriptor()); err != nil {
		return err
	}
	if newOwner {
		state, e := s.Read(ctx, tmdbConfigurationOwner)
		if e != nil {
			return e
		}
		if err = s.Apply(ctx, tmdbConfigurationOwner, authorization.Patch{Revision: &state.Revision, Granted: state.Requested}); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`PRAGMA user_version=17`); err != nil {
		return err
	}
	return tx.Commit()
}
