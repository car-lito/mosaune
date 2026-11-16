package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
)

// Optional host-adapter contracts. The generic settings and metadata routes
// depend on these capabilities, not on provider names or credential field names.
type MetadataConfigurationProvider interface {
	ConfigurationOwner() string
	Configuration(context.Context) (configurationBinding, error)
}

type MetadataCredentialProvider interface {
	ReadCredential(context.Context) (string, error)
	WriteCredential(context.Context, string) error
}

func installedProviderPackage(ctx context.Context, db configuration.Connection, id string, key ed25519.PublicKey) (packageformat.Package, error) {
	var raw []byte
	if err := db.QueryRowContext(ctx, "SELECT archive FROM provider_packages WHERE id=?", id).Scan(&raw); err != nil {
		return packageformat.Package{}, err
	}
	return packageformat.Parse(raw, key, version)
}

func (p *tmdbProvider) ConfigurationOwner() string { return tmdbConfigurationOwner }

func (p *tmdbProvider) Configuration(ctx context.Context) (configurationBinding, error) {
	return providerPackageConfiguration(ctx, p.db, p.Info().ID, p.ConfigurationOwner(), p.packageKey())
}

func providerPackageConfiguration(ctx context.Context, db *sql.DB, providerID, owner string, key ed25519.PublicKey) (configurationBinding, error) {
	// The signed schema and public settings must come from one database snapshot.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return configurationBinding{}, err
	}
	defer tx.Rollback()
	pack, err := installedProviderPackage(ctx, tx, providerID, key)
	if err != nil {
		return configurationBinding{}, err
	}
	if pack.Manifest.Descriptor.ID != owner {
		return configurationBinding{}, configuration.ErrStored
	}
	bound, err := (configuration.SQLStore{DB: tx}).For(owner)
	if err != nil {
		return configurationBinding{}, err
	}
	state, err := bound.Read(ctx)
	if err != nil {
		return configurationBinding{}, err
	}
	state = projectProviderSettings(state, pack.Manifest.Settings)
	if err = tx.Commit(); err != nil {
		return configurationBinding{}, err
	}
	return configurationBinding{
		schema: pack.Manifest.Settings, descriptor: pack.Manifest.Descriptor,
		authorizationsManaged: true,
		read:                  func(context.Context) (configuration.State, error) { return state, nil },
		write: func(ctx context.Context, patch configuration.Patch) error {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			defer tx.Rollback()
			var checksum string
			if err = tx.QueryRowContext(ctx, "SELECT checksum FROM provider_packages WHERE id=?", providerID).Scan(&checksum); err == sql.ErrNoRows {
				return configuration.ErrConflict
			} else if err != nil {
				return err
			}
			if checksum != pack.Checksum {
				return configuration.ErrConflict
			}
			// Recheck the installed schema here too: dormant rollback fields are
			// retained in storage, but never writable through an older form.
			patch, err = pack.Manifest.Settings.NormalizePatch(patch)
			if err != nil {
				return err
			}
			if patch.Revision == nil {
				revision := state.Revision
				patch.Revision = &revision
			}
			bound, err := (configuration.SQLStore{DB: tx}).For(pack.Manifest.Descriptor.ID)
			if err != nil {
				return err
			}
			if err = bound.Apply(ctx, patch); err != nil {
				return err
			}
			return tx.Commit()
		},
	}, nil
}

// Credentials remain shared with this adapter's legacy metadata endpoint until
// its last consumer is retired. Pausing or removing the executable does not erase them.
func (p *tmdbProvider) ReadCredential(ctx context.Context) (string, error) {
	return tmdbConfiguration(p.db).Secret(ctx, "apiToken")
}
func (p *tmdbProvider) WriteCredential(ctx context.Context, secret string) error {
	patch := configuration.Patch{}
	if secret == "" {
		patch.ClearSecrets = []string{"apiToken"}
	} else {
		raw, _ := json.Marshal(secret)
		patch.Values = map[string]json.RawMessage{"apiToken": raw}
	}
	return tmdbConfiguration(p.db).Apply(ctx, patch)
}
