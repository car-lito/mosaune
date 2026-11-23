package main

import (
	"errors"
	"fmt"
	"strings"
)

// Runtime composition is explicit. Selecting none neither constructs adapters
// nor removes existing packages, credentials or metadata from the catalogue.
func selectMetadataProviders(value string, available []metadataProviderFactory) ([]metadataProviderFactory, error) {
	if err := validateProviderFactories(available); err != nil {
		return nil, err
	}
	if len(value) > 4096 {
		return nil, errors.New("Liste de fournisseurs trop longue.")
	}
	selected := []metadataProviderFactory{}
	if strings.TrimSpace(value) == "none" {
		return selected, nil
	}
	byID := map[string]metadataProviderFactory{}
	for _, factory := range available {
		byID[factory.ID] = factory
	}
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		id := strings.TrimSpace(part)
		factory, ok := byID[id]
		if !ok || seen[id] {
			return nil, fmt.Errorf("Fournisseur inconnu ou répété : %q. Utilisez none pour aucun fournisseur.", id)
		}
		seen[id] = true
		selected = append(selected, factory)
	}
	return selected, nil
}

// Official composition is separate from catalogue startup. This is an explicit
// list of trusted host adapters, not permission to load arbitrary Go code.
func suppliedMetadataProviders() []metadataProviderFactory {
	return []metadataProviderFactory{
		{ID: "tmdb", InstallDefault: installInitialTMDB, Create: func(c *catalog) (metadataProviderRuntime, error) {
			p := newTMDBProvider(c.db)
			p.extension.blocked = func() bool { return c.extensionSafeMode }
			return p, nil
		}},
		{ID: "musicbrainz", Create: func(c *catalog) (metadataProviderRuntime, error) {
			p := newMusicBrainzProvider(c.db)
			p.owner = c
			return p, nil
		}},
	}
}
