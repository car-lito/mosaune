package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"time"
)

// A trusted adapter may depend on a selected executable slot. This binding
// follows replacements, rather than naming a factory package or its publisher.
type providerModuleBinding struct {
	owner *catalog
	slot  string
}
type moduleBackedProvider interface{ providerModule() providerModuleBinding }
type providerModuleLease struct {
	binding    providerModuleBinding
	descriptor protocol.Descriptor
	code       []byte
}
type providerModuleLeaseKey struct{}

func providerModuleAvailable(ctx context.Context, p MetadataProvider) bool {
	if p == nil {
		return false
	}
	adapter, ok := p.(moduleBackedProvider)
	if !ok {
		return true
	}
	binding := adapter.providerModule()
	if binding.owner == nil {
		return true
	}
	c := binding.owner
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	if c.extensionSafeMode {
		return false
	}
	var id string
	if err := c.db.QueryRowContext(ctx, "SELECT s.id FROM recognition_selection s JOIN recognition_packages p ON p.id=s.id AND p.kind=s.kind WHERE s.kind=?", binding.slot).Scan(&id); err != nil {
		return false
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		return false
	}
	entry, ok := registryEntryFor(state, id)
	// Integrity and executable contracts are checked when opening the operation;
	// reporting availability must not copy/hash a multi-MiB program on every poll.
	return ok && entry.Enabled
}

func beginProviderModule(ctx context.Context, p MetadataProvider) (context.Context, func(), error) {
	adapter, ok := p.(moduleBackedProvider)
	if !ok {
		return ctx, func() {}, context.Cause(ctx)
	}
	binding := adapter.providerModule()
	if binding.owner == nil {
		return ctx, func() {}, context.Cause(ctx)
	}
	if lease, ok := ctx.Value(providerModuleLeaseKey{}).(*providerModuleLease); ok {
		if lease.binding != binding {
			return ctx, func() {}, errors.New("Une opération ne peut pas changer de composant fournisseur.")
		}
		return ctx, func() {}, context.Cause(ctx)
	}
	call, done, descriptor, code, err := binding.owner.beginRecognition(ctx, binding.slot)
	if err != nil {
		return ctx, func() {}, err
	}
	return context.WithValue(call, providerModuleLeaseKey{}, &providerModuleLease{binding: binding, descriptor: descriptor, code: code}), done, nil
}

func beginProvider(ctx context.Context, p MetadataProvider) (context.Context, func(), error) {
	call, release, err := beginProviderModule(ctx, p)
	if err != nil {
		return ctx, func() {}, err
	}
	call, done, err := beginProviderState(call, p)
	if err != nil {
		release()
		return ctx, func() {}, err
	}
	return call, func() { done(); release() }, nil
}

// Same lock order as executable mutation: module lifecycle before SQL, then
// optional provider state. No result may commit after a successful retirement.
func providerCommit(ctx context.Context) (func(), error) {
	release := func() {}
	if lease, ok := ctx.Value(providerModuleLeaseKey{}).(*providerModuleLease); ok {
		var err error
		release, err = lease.binding.owner.recognitionCommit(ctx)
		if err != nil {
			return func() {}, err
		}
	}
	done, err := commitProviderState(ctx)
	if err != nil {
		release()
		return func() {}, err
	}
	return func() { done(); release() }, nil
}

func (p *musicBrainzProvider) providerModule() providerModuleBinding {
	if p.db == nil {
		return providerModuleBinding{}
	}
	return providerModuleBinding{owner: p.owner, slot: "music-results"}
}
