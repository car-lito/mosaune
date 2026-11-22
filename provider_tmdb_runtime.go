package main

import (
	"context"
	"database/sql"
)

func (p *tmdbProvider) StartMetadataProvider(ctx context.Context) error {
	active := true
	if err := p.db.QueryRowContext(ctx, "SELECT enabled FROM provider_states WHERE id=?", p.Info().ID).Scan(&active); err != nil && err != sql.ErrNoRows {
		return err
	}
	p.state.disabled = !active
	if err := p.loadServiceAuthorization(ctx); err != nil {
		return err
	}
	// A missing historical row keeps the constructor's default. Invalid stored
	// state is an error, never a reason to silently reactivate a provider.
	if err := p.db.QueryRowContext(ctx, "SELECT enabled FROM provider_runtime WHERE id=?", p.Info().ID).Scan(&p.extension.enabled); err != nil && err != sql.ErrNoRows {
		return err
	}
	p.extension.enabled = p.extension.enabled && p.extension.installed
	p.setModuleAvailable(p.extension.enabled)
	return ctx.Err()
}

func (p *tmdbProvider) CloseMetadataProvider() error {
	p.state.stop()
	err := p.closeExtensionWithError()
	p.client.CloseIdleConnections()
	return err
}
