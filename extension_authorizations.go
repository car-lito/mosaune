package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"mosaune/extensionkit/authorization"
	bundled "mosaune/extensionkit/providers/tmdb"
)

var errProviderAccessDenied = errors.New("L’accès en ligne de TMDB est retiré. Un administrateur peut le rétablir dans les accès de l’extension. Les fiches enregistrées restent disponibles.")

func (p *tmdbProvider) loadServiceAuthorization(ctx context.Context) error {
	state, err := (authorization.SQLStore{DB: p.db}).Read(ctx, tmdbConfigurationOwner)
	if err != nil {
		return err
	}
	p.state.accessDenied = !state.Allows("tmdb.api.v1", "get")
	return nil
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

// Serialize against providerCommit before starting a DB write. A revoked lease
// stays cancelled even if the administrator immediately grants access again.
func (p *tmdbProvider) setServiceAuthorization(ctx context.Context, patch authorization.Patch) error {
	for _, s := range patch.Granted {
		if s.Name != "tmdb.api.v1" {
			return authorization.ErrInvalid
		}
		for _, op := range s.Operations {
			if op != "get" {
				return authorization.ErrInvalid
			}
		}
	}
	p.state.changes.Lock()
	defer p.state.changes.Unlock()
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	s := authorization.SQLStore{DB: tx}
	if err = s.Apply(ctx, tmdbConfigurationOwner, patch); err != nil {
		return err
	}
	state, err := s.Read(ctx, tmdbConfigurationOwner)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	p.state.accessDenied = !state.Allows("tmdb.api.v1", "get")
	if p.state.accessDenied {
		for _, cancel := range p.state.calls {
			cancel(errProviderAccessDenied)
		}
	}
	return nil
}

func (c *catalog) extensionAuthorizationRoutes(mux *secureMux) {
	// Only host adapters with a complete revocation/commit lifecycle may expose
	// mutable grants. A registry row alone must not enable an unenforced switch.
	provider := func(w http.ResponseWriter, r *http.Request) *tmdbProvider {
		p, ok := c.provider("tmdb").(*tmdbProvider)
		if !ok || r.PathValue("id") != tmdbConfigurationOwner {
			problem(w, 404, "Accès de cette extension non disponibles.")
			return nil
		}
		return p
	}
	mux.HandleFunc("GET /api/v1/extensions/{id}/authorizations", func(w http.ResponseWriter, r *http.Request) {
		if provider(w, r) == nil {
			return
		}
		state, err := (authorization.SQLStore{DB: c.db}).Read(r.Context(), tmdbConfigurationOwner)
		if err != nil {
			problem(w, 500, "Autorisations indisponibles.")
			return
		}
		sendJSON(w, 200, state)
	})
	mux.HandleFunc("PUT /api/v1/extensions/{id}/authorizations", func(w http.ResponseWriter, r *http.Request) {
		p := provider(w, r)
		if p == nil {
			return
		}
		var patch authorization.Patch
		if !readJSON(w, r, &patch) {
			return
		}
		if err := p.setServiceAuthorization(r.Context(), patch); err != nil {
			switch {
			case errors.Is(err, authorization.ErrConflict):
				problem(w, 409, err.Error())
			case errors.Is(err, authorization.ErrInvalid):
				problem(w, 400, err.Error())
			default:
				problem(w, 500, "Autorisations non enregistrées.")
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
