package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/authorization"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	runtime "mosaune/extensionkit/runtime"
	"strings"
	"time"
)

//go:embed provider-package-key.pub
var providerPublicKey string

func trustedProviderKey() ed25519.PublicKey {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(providerPublicKey))
	if err != nil || len(key) != ed25519.PublicKeySize {
		panic("invalid provider trust key")
	}
	return key
}
func migrateProviderPackages(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 12 {
		return nil
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS provider_packages(id TEXT PRIMARY KEY,archive BLOB NOT NULL,checksum TEXT NOT NULL,previous BLOB,previous_checksum TEXT NOT NULL DEFAULT ''); PRAGMA user_version=12;`)
	return err
}

type providerPackageState struct {
	Registration        *providerRegistration  `json:"registration,omitempty"`
	RegistrationWarning string                 `json:"registrationWarning,omitempty"`
	Package             *packageformat.Package `json:"package"`
	PreviousVersion     string                 `json:"previousVersion"`
	PreviousChecksum    string                 `json:"previousChecksum"`
	Runtime             map[string]bool        `json:"runtime"`
}

type providerRegistration struct {
	Revision     int64             `json:"revision"`
	Wanted       bool              `json:"wanted"`
	Dependencies []string          `json:"dependencies"`
	Choices      map[string]string `json:"choices"`
}

func (p *tmdbProvider) packageKey() ed25519.PublicKey {
	if p.extension.key != nil {
		return p.extension.key
	}
	return trustedProviderKey()
}
func (p *tmdbProvider) installedPackage(ctx context.Context) (packageformat.Package, error) {
	if p.db == nil {
		return packageformat.Package{}, sql.ErrNoRows
	}
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT archive FROM provider_packages WHERE id='tmdb'`).Scan(&raw)
	if err != nil {
		return packageformat.Package{}, err
	}
	return packageformat.Parse(raw, p.packageKey(), version)
}
func (p *tmdbProvider) packageState(ctx context.Context) (providerPackageState, error) {
	state := providerPackageState{Runtime: p.extensionStatus()}
	current, err := p.installedPackage(ctx)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.Package = &current
	entry, revision, registeredErr := p.registeredPackage(ctx, current)
	if registeredErr != nil {
		state.RegistrationWarning = "L’enregistrement de ce paquet doit être réparé avant son démarrage. Installez une version vérifiée plus récente."
	} else {
		deps := append([]string{}, entry.Descriptor.Dependencies...)
		state.Registration = &providerRegistration{Revision: revision, Wanted: entry.Enabled, Dependencies: deps, Choices: entry.Choices}
	}
	var raw []byte
	err = p.db.QueryRowContext(ctx, `SELECT previous,previous_checksum FROM provider_packages WHERE id='tmdb'`).Scan(&raw, &state.PreviousChecksum)
	if err != nil {
		return state, err
	}
	if len(raw) > 0 {
		previous, e := packageformat.Parse(raw, p.packageKey(), version)
		if e != nil {
			return state, e
		}
		state.PreviousVersion = previous.Manifest.Descriptor.Version
	}
	return state, nil
}
func prepareProviderEngine(ctx context.Context, d protocol.Descriptor, code []byte, health bool, savedChoices ...map[string]string) (*runtime.Engine, error) {
	limits := runtime.DefaultLimits()
	limits.Modules = 1
	limits.Timeout = 15 * time.Second
	engine, err := runtime.New(context.Background(), limits)
	if err == nil {
		err = engine.Load(ctx, d, code)
	}
	if err == nil {
		var choices map[string]string
		if len(savedChoices) > 0 {
			choices = savedChoices[0]
		}
		err = engine.EnableWithProviders(d.ID, choices)
	}
	if err == nil && health {
		// Health checks have no service grants, network, credentials or catalogue.
		check, cancel := context.WithTimeout(ctx, 3*time.Second)
		var result runtime.Result
		result, err = engine.Call(check, d.ID, protocol.Request{Protocol: 1, ID: "package-check", Capability: packageformat.HealthCapability, Operation: "check", Payload: json.RawMessage(`{}`)})
		cancel()
		if err == nil {
			var reply struct {
				Version  string `json:"version"`
				Provider string `json:"provider"`
			}
			if result.Response.Error != nil || protocol.DecodeOne(result.Response.Result, &reply) != nil || reply.Version != d.Version || reply.Provider != "tmdb" {
				err = errors.New("test de démarrage du fournisseur non concluant")
			}
		}
	}
	if err != nil {
		if engine != nil {
			_ = engine.Close()
		}
		return nil, errors.New("Le nouveau module ne démarre pas correctement. La version précédente reste inchangée.")
	}
	return engine, nil
}
func (p *tmdbProvider) effectiveEngine(ctx context.Context) (*runtime.Engine, error) {
	pack, err := p.installedPackage(ctx)
	if err == nil {
		entry, _, e := p.registeredPackage(ctx, pack)
		if e != nil {
			return nil, e
		}
		if !entry.Enabled {
			return nil, errors.New("Le paquet n’est pas activé dans le registre. Actualisez son choix de moteur.")
		}
		return prepareProviderEngine(ctx, pack.Manifest.Descriptor, pack.Code, true, entry.Choices)
	}
	if err != sql.ErrNoRows {
		return nil, errors.New("Paquet installé illisible. Utilisez le mode de secours ou restaurez le fournisseur.")
	}
	return nil, errors.New("Le paquet TMDB est absent. Réinstallez-le depuis le catalogue des extensions.")
}

// Installation is serialized with activation and calls. Preparation happens
// before the single atomic replacement; no failed candidate changes the DB.
func (p *tmdbProvider) installPackage(ctx context.Context, pack packageformat.Package, expected string, rollback bool) error {
	select {
	case p.extension.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.extension.gate }()
	p.extension.mu.Lock()
	defer p.extension.mu.Unlock()
	if p.extension.blocked != nil && p.extension.blocked() {
		return errors.New("Quittez le mode de secours pour installer ou restaurer un module.")
	}
	current, err := p.installedPackage(ctx)
	exists := err == nil
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if current.Checksum != expected {
		return errors.New("L’installation a changé. Vérifiez de nouveau le paquet.")
	}
	if exists && !rollback && packageformat.Compare(pack.Manifest.Descriptor.Version, current.Manifest.Descriptor.Version) <= 0 {
		return errors.New("La mise à jour doit avoir une version supérieure.")
	}
	if rollback {
		var checksum string
		if err = p.db.QueryRowContext(ctx, `SELECT previous_checksum FROM provider_packages WHERE id='tmdb'`).Scan(&checksum); err != nil || checksum != pack.Checksum {
			return errors.New("Version précédente modifiée ou absente.")
		}
	}
	candidate, err := prepareProviderEngine(ctx, pack.Manifest.Descriptor, pack.Code, true)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = candidate.Close()
		}
	}()
	enabled := exists && p.extension.enabled
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO provider_packages(id,archive,checksum,previous,previous_checksum) VALUES('tmdb',?,?,?,?) ON CONFLICT(id) DO UPDATE SET archive=excluded.archive,checksum=excluded.checksum,previous=excluded.previous,previous_checksum=excluded.previous_checksum`, pack.Archive, pack.Checksum, current.Archive, current.Checksum)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO provider_runtime(id,enabled) VALUES('tmdb',?) ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled`, enabled)
	}
	if err == nil {
		err = registerProviderPackage(ctx, tx, pack, enabled, rollback)
	}
	if err == nil {
		err = (configuration.SQLStore{DB: tx}).Register(ctx, tmdbConfigurationOwner, pack.Manifest.Settings)
	}
	if err == nil {
		// The pilot format requires exactly tmdb.api.v1/get for both engines.
		// Registration preserves an earlier denial, never grants on update.
		err = (authorization.SQLStore{DB: tx}).Register(ctx, pack.Manifest.Descriptor)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if p.extension.engine != nil {
		_ = p.extension.engine.Close()
		p.extension.engine = nil
	}
	p.extension.installed = true
	p.extension.enabled = enabled
	p.setModuleAvailable(enabled)
	if enabled {
		p.extension.engine = candidate
		keep = true
	}
	return nil
}
func (p *tmdbProvider) removePackage(ctx context.Context, checksum string) error {
	// Closing the engine cancels its current work; credentials and media belong
	// to the shared host settings/catalogue and must not be deleted.
	p.extension.mu.Lock()
	defer p.extension.mu.Unlock()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM provider_packages WHERE id='tmdb' AND checksum=?`, checksum)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("Paquet absent ou modifié. Actualisez la liste.")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE provider_runtime SET enabled=0 WHERE id='tmdb'`); err != nil {
		return err
	}
	if err = removeRegisteredProvider(ctx, tx); err != nil {
		return err
	}
	// The host connection adapter still consumes these settings. Preserve consent
	// and credentials until that remaining consumer has itself been extracted.
	if err = (configuration.SQLStore{DB: tx}).Register(ctx, tmdbConfigurationOwner, bundled.Settings()); err != nil {
		return err
	}
	if err = (authorization.SQLStore{DB: tx}).Register(ctx, bundled.Descriptor()); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	p.extension.installed = false
	p.extension.enabled = false
	p.setModuleAvailable(false)
	if p.extension.engine != nil {
		_ = p.extension.engine.Close()
		p.extension.engine = nil
	}
	return nil
}
