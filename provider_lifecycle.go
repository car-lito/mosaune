package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
)

var errProviderDisabled = errors.New("Fournisseur désactivé. Les informations déjà enregistrées restent disponibles.")

// Optional host-owned lifecycle, independent of the provider's execution engine.
// A lease covers a whole operation, including persistence of its result.
type providerLifecycle struct {
	changes  sync.Mutex
	mu       sync.Mutex
	disabled bool
	next     uint64
	calls    map[uint64]context.CancelCauseFunc
}
type lifecycleProvider interface{ lifecycle() *providerLifecycle }
type providerLease struct{ owner *providerLifecycle }
type providerLeaseKey struct{}

func providerEnabled(p MetadataProvider) bool {
	if managed, ok := p.(lifecycleProvider); ok {
		s := managed.lifecycle()
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.disabled
	}
	return true
}
func beginProvider(ctx context.Context, p MetadataProvider) (context.Context, func(), error) {
	managed, ok := p.(lifecycleProvider)
	if !ok {
		return ctx, func() {}, ctx.Err()
	}
	s := managed.lifecycle()
	if lease, ok := ctx.Value(providerLeaseKey{}).(providerLease); ok && lease.owner == s {
		return ctx, func() {}, context.Cause(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.disabled {
		return ctx, func() {}, errProviderDisabled
	}
	if ctx.Err() != nil {
		return ctx, func() {}, context.Cause(ctx)
	}
	call, cancel := context.WithCancelCause(ctx)
	s.next++
	id := s.next
	if s.calls == nil {
		s.calls = map[uint64]context.CancelCauseFunc{}
	}
	s.calls[id] = cancel
	return context.WithValue(call, providerLeaseKey{}, providerLease{s}), func() { s.mu.Lock(); delete(s.calls, id); s.mu.Unlock(); cancel(context.Canceled) }, nil
}

// Acquire before opening a write transaction, never while holding one. A disable
// cannot overtake a committing result, and old leases stay cancelled on re-enable.
func providerCommit(ctx context.Context) (func(), error) {
	if lease, ok := ctx.Value(providerLeaseKey{}).(providerLease); ok {
		lease.owner.mu.Lock()
		if err := context.Cause(ctx); err != nil {
			lease.owner.mu.Unlock()
			return func() {}, err
		}
		if lease.owner.disabled {
			lease.owner.mu.Unlock()
			return func() {}, errProviderDisabled
		}
		return lease.owner.mu.Unlock, nil
	}
	return func() {}, ctx.Err()
}
func providerResult[T any](ctx context.Context, p MetadataProvider, run func(context.Context) (T, error)) (T, error) {
	var zero T
	ctx, done, err := beginProvider(ctx, p)
	if err != nil {
		return zero, err
	}
	defer done()
	value, err := run(ctx)
	if cause := context.Cause(ctx); cause != nil {
		return zero, cause
	}
	return value, err
}
func (p *tmdbProvider) lifecycle() *providerLifecycle { return &p.state }
func (p *tmdbProvider) setProviderEnabled(ctx context.Context, enabled bool) error {
	p.state.changes.Lock()
	defer p.state.changes.Unlock()
	p.state.mu.Lock()
	if p.db != nil {
		if _, err := p.db.ExecContext(ctx, `INSERT INTO provider_states(id,enabled) VALUES('tmdb',?) ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled`, enabled); err != nil {
			p.state.mu.Unlock()
			return errors.New("État du fournisseur non enregistré.")
		}
	}
	p.state.disabled = !enabled
	if !enabled {
		for _, cancel := range p.state.calls {
			cancel(errProviderDisabled)
		}
	}
	p.state.mu.Unlock()
	// Release the guest's memory without losing the user's engine preference.
	if !enabled {
		p.extension.mu.Lock()
		if p.extension.engine != nil {
			_ = p.extension.engine.Close()
			p.extension.engine = nil
		}
		p.extension.mu.Unlock()
	}
	return nil
}
func migrateProviderStates(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 14 {
		return nil
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS provider_states(id TEXT PRIMARY KEY,enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1))); PRAGMA user_version=14;`)
	return err
}
func providerStateRoutes(mux *secureMux, c *catalog) {
	mux.HandleFunc("PUT /api/v1/metadata/providers/{provider}/state", func(w http.ResponseWriter, r *http.Request) {
		p, ok := c.provider(r.PathValue("provider")).(*tmdbProvider)
		if !ok {
			problem(w, 404, "Ce fournisseur ne propose pas encore ce réglage.")
			return
		}
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		if input.Enabled == nil {
			problem(w, 400, "État requis.")
			return
		}
		if err := p.setProviderEnabled(r.Context(), *input.Enabled); err != nil {
			problem(w, 500, err.Error())
			return
		}
		sendJSON(w, 200, map[string]bool{"enabled": providerEnabled(p)})
	})
}
