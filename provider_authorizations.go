package main

import (
	"context"
	"database/sql"
	"errors"
	"mosaune/extensionkit/authorization"
	"mosaune/extensionkit/protocol"
)

var errProviderAccessDenied = errors.New("L’accès en ligne de ce fournisseur est retiré. Un administrateur peut le rétablir dans les accès de l’extension. Les fiches enregistrées restent disponibles.")

// Only trusted host adapters may bind a provider's grants to its operation
// lifecycle. A manifest or registry row alone cannot expose a permission switch.
type MetadataAuthorizationProvider interface {
	ServiceAccess() providerServiceAccess
}

type providerServiceAccess struct {
	db        *sql.DB
	owner     string
	lifecycle *providerLifecycle
	services  []extensionServiceView
}

func (a providerServiceAccess) valid() bool {
	if a.db == nil || a.lifecycle == nil || len(a.services) == 0 {
		return false
	}
	services := make([]protocol.ServiceRequirement, 0, len(a.services))
	for _, s := range a.services {
		services = append(services, protocol.ServiceRequirement{Name: s.Name, Operations: s.Operations})
	}
	return (protocol.Descriptor{ID: a.owner, Version: "1.0.0", Capabilities: []string{"host.authorization.v1"}, Services: services}).Validate() == nil
}

func (a providerServiceAccess) read(ctx context.Context) (authorization.State, error) {
	if !a.valid() {
		return authorization.State{}, authorization.ErrInvalid
	}
	state, err := (authorization.SQLStore{DB: a.db}).Read(ctx, a.owner)
	if err != nil {
		return authorization.State{}, err
	}
	// Preserve the storage revision, but never offer a switch for an operation
	// that this host adapter cannot enforce. Apply still validates raw input.
	filter := func(services []protocol.ServiceRequirement) []protocol.ServiceRequirement {
		out := []protocol.ServiceRequirement{}
		for _, s := range services {
			ops := []string{}
			for _, known := range a.services {
				if known.Name == s.Name {
					for _, op := range s.Operations {
						for _, supported := range known.Operations {
							if op == supported {
								ops = append(ops, op)
							}
						}
					}
				}
			}
			if len(ops) > 0 {
				out = append(out, protocol.ServiceRequirement{Name: s.Name, Operations: ops})
			}
		}
		return out
	}
	state.Requested, state.Granted = filter(state.Requested), filter(state.Granted)
	return state, nil
}

func (a providerServiceAccess) allows(state authorization.State) bool {
	for _, s := range a.services {
		for _, op := range s.Operations {
			if !state.Allows(s.Name, op) {
				return false
			}
		}
	}
	return true
}

// Persist first, then cancel leases under the same lock used by providerCommit.
// Failed writes leave both current calls and effective permissions unchanged.
func (a providerServiceAccess) apply(ctx context.Context, patch authorization.Patch) error {
	if !a.valid() {
		return authorization.ErrInvalid
	}
	for _, requested := range patch.Granted {
		for _, op := range requested.Operations {
			found := false
			for _, supported := range a.services {
				if supported.Name == requested.Name {
					for _, known := range supported.Operations {
						if known == op {
							found = true
						}
					}
				}
			}
			if !found {
				return authorization.ErrInvalid
			}
		}
	}
	a.lifecycle.changes.Lock()
	defer a.lifecycle.changes.Unlock()
	a.lifecycle.mu.Lock()
	defer a.lifecycle.mu.Unlock()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := authorization.SQLStore{DB: tx}
	if err = store.Apply(ctx, a.owner, patch); err != nil {
		return err
	}
	state, err := store.Read(ctx, a.owner)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	a.lifecycle.accessDenied = !a.allows(state)
	if a.lifecycle.accessDenied {
		for _, cancel := range a.lifecycle.calls {
			cancel(errProviderAccessDenied)
		}
	}
	return nil
}

// Called during startup, before the provider is made available to requests.
func (a providerServiceAccess) load(ctx context.Context) error {
	state, err := a.read(ctx)
	if err != nil {
		return err
	}
	a.lifecycle.mu.Lock()
	defer a.lifecycle.mu.Unlock()
	a.lifecycle.accessDenied = !a.allows(state)
	return nil
}

func (c *catalog) authorizationFor(owner string) (providerServiceAccess, bool) {
	for _, provider := range c.providers {
		if managed, ok := provider.(MetadataAuthorizationProvider); ok {
			access := managed.ServiceAccess()
			if access.owner == owner {
				return access, access.valid()
			}
		}
	}
	return providerServiceAccess{}, false
}

func (c *catalog) serviceViews(d protocol.Descriptor) []extensionServiceView {
	views := []extensionServiceView{}
	access, ok := c.authorizationFor(d.ID)
	if !ok {
		return views
	}
	for _, requested := range d.Services {
		for _, known := range access.services {
			if requested.Name == known.Name {
				view := known
				view.Operations = []string{}
				for _, op := range requested.Operations {
					for _, supported := range known.Operations {
						if op == supported {
							view.Operations = append(view.Operations, op)
						}
					}
				}
				if len(view.Operations) > 0 {
					views = append(views, view)
				}
			}
		}
	}
	return views
}
