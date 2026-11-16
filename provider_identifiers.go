package main

import (
	"context"
	"errors"
	"fmt"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"strings"
)

// Other providers implement this same contract; download connectors do not
// need knowledge of their API paths or response fields.
type MetadataIdentifierProvider interface {
	ResolveIdentifier(context.Context, string, metadata.IdentifierQuery) (string, error)
}

func (p *tmdbProvider) ResolveIdentifier(ctx context.Context, secret string, q metadata.IdentifierQuery) (string, error) {
	if q.Validate() != nil {
		return "", errors.New("Identifiant demandé invalide.")
	}
	// An already stored provider identifier is local data, available while paused.
	if q.Namespace == p.Info().ID {
		return q.ID, nil
	}
	return providerResult(ctx, p, func(ctx context.Context) (string, error) {
		if e := p.moduleReady(); e != nil {
			return "", e
		}
		// Serialize with package updates. Lifecycle cancellation handles removal.
		select {
		case p.extension.gate <- struct{}{}:
		case <-ctx.Done():
			return "", context.Cause(ctx)
		}
		defer func() { <-p.extension.gate }()
		if e := p.moduleReady(); e != nil {
			return "", e
		}
		pack, e := p.installedPackage(ctx)
		if e != nil {
			return "", e
		}
		profile, e := p.configuredProfile(ctx)
		if e != nil {
			return "", e
		}
		route, source, path, e := metadata.IdentifierRequest(pack.Manifest.Identifiers, profile, q)
		if e != nil {
			return "", errors.New("Ce fournisseur installé ne propose pas cet identifiant. Vérifiez sa mise à jour dans Extensions.")
		}
		if p.apiBase != "" && source.ID == "api" {
			source.BaseURL = p.apiBase
		} // host-only test fixture
		call, e := providerSourceCall(p.client, &p.rate, source, path, nil, secret)
		if e != nil {
			return "", e
		}
		call.DB = p.db
		call.Provider = p.Info().ID
		call.BeforeCommit = providerCommit
		var value string
		call.Validate = func(body []byte) error {
			var err error
			value, err = metadata.ExtractIdentifier(body, route.Field)
			return err
		}
		_, status, e := fetchProviderHTTP(ctx, call)
		if status != 0 && status != 200 {
			return "", fmt.Errorf("Identifiant indisponible auprès du fournisseur (HTTP %d).", status)
		}
		if e != nil {
			return "", fmt.Errorf("Le fournisseur n’a pas pu fournir un identifiant %s valide. Aucune demande envoyée.", strings.ToUpper(q.Namespace))
		}
		return value, nil
	})
}
