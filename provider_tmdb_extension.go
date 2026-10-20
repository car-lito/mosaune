package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"errors"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	runtime "mosaune/extensionkit/runtime"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// The bundled module and signed TMDB releases use this host adapter.
// Untrusted community code remains excluded until process budgets are ready.
type tmdbExtension struct {
	key     ed25519.PublicKey
	mu      sync.Mutex
	enabled bool
	engine  *runtime.Engine
	gate    chan struct{}
	blocked func() bool
}

func migrateProviderRuntime(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 11 {
		return nil
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS provider_runtime(id TEXT PRIMARY KEY,enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0,1))); PRAGMA user_version=11;`)
	return err
}
func (p *tmdbProvider) extensionStatus() map[string]bool {
	active := providerEnabled(p)
	available := providerAvailable(p)
	p.extension.mu.Lock()
	defer p.extension.mu.Unlock()
	blocked := p.extension.blocked != nil && p.extension.blocked()
	return map[string]bool{"enabled": p.extension.enabled, "providerEnabled": active, "effective": available && p.extension.enabled && !blocked, "loaded": p.extension.engine != nil, "safeMode": blocked, "serviceAccess": available}
}
func (p *tmdbProvider) setExtension(ctx context.Context, enabled bool) error {
	p.extension.mu.Lock()
	defer p.extension.mu.Unlock()
	if enabled && p.extension.blocked != nil && p.extension.blocked() {
		return errors.New("Le mode de secours empêche l’activation des extensions.")
	}
	if p.db != nil {
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return errors.New("Réglage non enregistré.")
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, `INSERT INTO provider_runtime(id,enabled) VALUES('tmdb',?) ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled`, enabled); err != nil {
			return errors.New("Réglage non enregistré.")
		}
		if err = setRegisteredProviderWanted(ctx, tx, enabled); err != nil {
			return errors.New("Choix de l’extension non enregistré.")
		}
		if err = tx.Commit(); err != nil {
			return errors.New("Réglage non enregistré.")
		}
	}
	p.extension.enabled = enabled
	if !enabled && p.extension.engine != nil {
		_ = p.extension.engine.Close()
		p.extension.engine = nil
	}
	return nil
}
func (p *tmdbProvider) closeExtension() {
	p.extension.mu.Lock()
	defer p.extension.mu.Unlock()
	p.extension.enabled = false
	if p.extension.engine != nil {
		_ = p.extension.engine.Close()
		p.extension.engine = nil
	}
}
func (p *tmdbProvider) activeSearch(ctx context.Context, secret string, q MetadataQuery) ([]MetadataCandidate, error) {
	p.extension.mu.Lock()
	enabled := p.extension.enabled && (p.extension.blocked == nil || !p.extension.blocked())
	p.extension.mu.Unlock()
	if !enabled {
		return p.searchNative(ctx, secret, q)
	}
	query := metadata.Query{Kind: q.Kind, Title: q.Title, Year: q.Year, Language: appLanguage}
	if err := query.Validate(); err != nil || (q.Kind != "films" && q.Kind != "series") {
		return nil, errors.New("Recherche invalide.")
	}
	engine, release, err := p.extensionEngine(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	if engine == nil {
		return p.searchNative(ctx, secret, q)
	}
	payload, _ := json.Marshal(query)
	var serviceErr error
	used := false
	result, err := engine.CallWithServices(ctx, bundled.Descriptor().ID, protocol.Request{Protocol: 1, ID: "tmdb-search", Capability: metadata.SearchCapability, Operation: "search", Payload: payload}, runtime.ServiceGrants{
		"tmdb.api.v1": func(call context.Context, operation string, raw json.RawMessage) (json.RawMessage, error) {
			if used {
				return nil, errors.New("request budget exhausted")
			}
			used = true
			params, path, e := tmdbSearchGrant(operation, raw, query)
			if e != nil {
				return nil, e
			}
			var body json.RawMessage
			serviceErr = p.get(call, secret, path, params, &body)
			return body, serviceErr
		},
	})
	if serviceErr != nil {
		return nil, serviceErr
	}
	if err != nil {
		return nil, errors.New("Recherche interrompue ou moteur TMDB expérimental indisponible. Réessayez ou désactivez-le dans les paramètres.")
	}
	if result.Response.Error != nil {
		return nil, errors.New("L’extension TMDB n’a pas pu terminer cette recherche.")
	}
	var search metadata.SearchResult
	if protocol.DecodeOne(result.Response.Result, &search) != nil || search.Validate() != nil {
		return nil, errors.New("Résultats de l’extension TMDB invalides.")
	}
	candidates := make([]MetadataCandidate, 0, len(search.Items))
	for _, item := range search.Items {
		id, e := strconv.Atoi(item.ID)
		if e != nil || id <= 0 || id > 2147483647 {
			return nil, errors.New("Identifiant TMDB invalide.")
		}
		candidates = append(candidates, MetadataCandidate{Presentation: item.Presentation, ID: item.ID, Title: item.Title, OriginalTitle: item.OriginalTitle, Year: item.Year, Overview: item.Overview})
	}
	return candidates, nil
}

// A guest cannot change the host, path, credentials or the user's search.
func tmdbSearchGrant(operation string, raw json.RawMessage, q metadata.Query) (url.Values, string, error) {
	var request struct {
		Path  string            `json:"path"`
		Query map[string]string `json:"query"`
	}
	deny := errors.New("TMDB service request denied")
	if operation != "get" || protocol.DecodeOne(raw, &request) != nil {
		return nil, "", deny
	}
	path := "/search/movie"
	yearKey := "year"
	if q.Kind == "series" {
		path = "/search/tv"
		yearKey = "first_air_date_year"
	} else if q.Kind != "films" {
		return nil, "", deny
	}
	expected := map[string]string{"query": q.Title, "include_adult": "false", "language": q.Language}
	if q.Year > 0 {
		expected[yearKey] = strconv.Itoa(q.Year)
	}
	if request.Path != path || len(request.Query) != len(expected) {
		return nil, "", deny
	}
	params := url.Values{}
	for key, value := range expected {
		if request.Query[key] != value {
			return nil, "", deny
		}
		params.Set(key, value)
	}
	return params, path, nil
}
func tmdbExtensionRoutes(mux *secureMux, c *catalog) {
	mux.HandleFunc("GET /api/v1/metadata/tmdb-runtime", func(w http.ResponseWriter, r *http.Request) {
		p, ok := c.provider("tmdb").(*tmdbProvider)
		if !ok {
			problem(w, 404, "Fournisseur absent.")
			return
		}
		sendJSON(w, 200, p.extensionStatus())
	})
	mux.HandleFunc("PUT /api/v1/metadata/tmdb-runtime", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Enabled bool `json:"enabled"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		p, ok := c.provider("tmdb").(*tmdbProvider)
		if !ok {
			problem(w, 404, "Fournisseur absent.")
			return
		}
		if err := p.setExtension(r.Context(), input.Enabled); err != nil {
			problem(w, 400, err.Error())
			return
		}
		sendJSON(w, 200, p.extensionStatus())
	})
}

// A shared lease serializes Search and Details and returns to native mode if disabled while waiting.
func (p *tmdbProvider) extensionEngine(ctx context.Context) (*runtime.Engine, func(), error) {
	select {
	case p.extension.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	release := func() { <-p.extension.gate }
	p.extension.mu.Lock()
	if !p.extension.enabled || (p.extension.blocked != nil && p.extension.blocked()) {
		p.extension.mu.Unlock()
		release()
		return nil, func() {}, nil
	}
	if p.extension.engine == nil {
		engine, err := p.effectiveEngine(ctx)
		if err != nil {
			if engine != nil {
				_ = engine.Close()
			}
			p.extension.mu.Unlock()
			release()
			return nil, nil, errors.New("Le moteur TMDB expérimental ne démarre pas. Vous pouvez le désactiver dans les paramètres.")
		}
		p.extension.engine = engine
	}
	engine := p.extension.engine
	p.extension.mu.Unlock()

	return engine, release, nil
}
