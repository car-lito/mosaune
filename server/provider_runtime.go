package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// Trusted host adapters own startup/shutdown. The catalogue neither constructs
// a concrete provider nor reads its private runtime settings.
type metadataProviderRuntime interface {
	MetadataProvider
	StartMetadataProvider(context.Context) error
	CloseMetadataProvider() error
}

type metadataProviderFactory struct {
	InstallDefault func(context.Context, *sql.Tx) error
	ID             string
	Create         func(*catalog) (metadataProviderRuntime, error)
}

var providerRegistrationID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func nilProvider(p metadataProviderRuntime) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func (c *catalog) startMetadataProviders(ctx context.Context, factories []metadataProviderFactory) error {
	if len(c.providers) != 0 {
		return errors.New("Les fournisseurs sont déjà démarrés.")
	}
	if err := validateProviderFactories(factories); err != nil {
		return err
	}
	started := []MetadataProvider{}
	fail := func(err error) error { return errors.Join(err, closeMetadataProviders(started)) }
	for _, f := range factories {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		p, err := f.Create(c)
		if !nilProvider(p) {
			started = append(started, p)
		}
		if err != nil {
			return fail(fmt.Errorf("Initialisation du fournisseur %s : %w", f.ID, err))
		}
		if nilProvider(p) || p.Info().ID != f.ID {
			return fail(fmt.Errorf("Identité du fournisseur %s incohérente.", f.ID))
		}
		if err = p.StartMetadataProvider(ctx); err != nil {
			return fail(fmt.Errorf("Démarrage du fournisseur %s : %w", f.ID, err))
		}
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	// Publish only the complete composition; failed startup leaves no providers.
	c.providers = started
	return nil
}

func closeMetadataProviders(providers []MetadataProvider) error {
	var errs []error
	for i := len(providers) - 1; i >= 0; i-- {
		if p, ok := providers[i].(interface{ CloseMetadataProvider() error }); ok {
			errs = append(errs, p.CloseMetadataProvider())
		}
	}
	return errors.Join(errs...)
}

func validateProviderFactories(factories []metadataProviderFactory) error {
	if len(factories) > 128 {
		return errors.New("Trop de fournisseurs.")
	}
	seen := map[string]bool{}
	// Validate composition before any constructor can allocate resources.
	for _, f := range factories {
		if !providerRegistrationID.MatchString(f.ID) || f.Create == nil || seen[f.ID] {
			return errors.New("Composition des fournisseurs invalide.")
		}
		seen[f.ID] = true
	}
	return nil
}
