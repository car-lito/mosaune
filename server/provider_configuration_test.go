package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"strings"
	"testing"
)

// A second host adapter uses a different provider, owner and secret field.
// Neither HTTP handler is allowed to infer any of those names.
type configurationProviderFixture struct {
	MetadataProvider
	bound   configuration.Bound
	failure error
}

func (p *configurationProviderFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "fixture-provider", RequiresCredential: true}
}
func (p *configurationProviderFixture) ConfigurationOwner() string { return "fixture.settings" }
func (p *configurationProviderFixture) Configuration(ctx context.Context) (configurationBinding, error) {
	if p.failure != nil {
		return configurationBinding{}, p.failure
	}
	s, err := p.bound.Read(ctx)
	return configurationBinding{schema: s.Schema, descriptor: protocol.Descriptor{ID: p.ConfigurationOwner()}, read: p.bound.Read, write: p.bound.Apply}, err
}
func (p *configurationProviderFixture) ReadCredential(ctx context.Context) (string, error) {
	return p.bound.Secret(ctx, "token")
}
func (p *configurationProviderFixture) WriteCredential(ctx context.Context, secret string) error {
	if secret == "" {
		return p.bound.Apply(ctx, configuration.Patch{ClearSecrets: []string{"token"}})
	}
	raw, _ := json.Marshal(secret)
	return p.bound.Apply(ctx, configuration.Patch{Values: map[string]json.RawMessage{"token": raw}})
}

func TestProviderConfigurationOptionalAdapterHTTP(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := context.Background()
	store := configuration.SQLStore{DB: c.db}
	if err := store.Register(ctx, "fixture.settings", configSchema()); err != nil {
		t.Fatal(err)
	}
	bound, _ := store.For("fixture.settings")
	p := &configurationProviderFixture{bound: bound}
	c.providers = append(c.providers, p)
	const path = "/api/v1/extensions/fixture.settings/configuration"
	const secret = "fixture-alternate-secret-123456"
	if w := admin.request("PUT", "/api/v1/metadata/providers/fixture-provider", map[string]string{"secret": secret}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if c.credential(ctx, "fixture-provider") != secret {
		t.Fatal("credential adapter bypassed")
	}
	w := admin.request("GET", path, nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), secret) || !strings.Contains(w.Body.String(), `"token":true`) || !strings.Contains(w.Body.String(), `"authorizationsManaged":false`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = admin.request("PUT", path, map[string]any{"values": map[string]string{"name": "Example"}}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	state, _ := bound.Read(ctx)
	if string(state.Values["name"]) != `"Example"` {
		t.Fatal("settings not routed to adapter")
	}
	if w = admin.request("DELETE", "/api/v1/metadata/providers/fixture-provider", nil); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if c.credential(ctx, "fixture-provider") != "" {
		t.Fatal("credential not cleared")
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "PUT"} {
		if w = reader.request(method, path, map[string]any{}); w.Code != 403 {
			t.Fatal("non-admin", w.Code)
		}
	}
	p.failure = errors.New("unavailable")
	if w = admin.request("GET", path, nil); w.Code != 404 {
		t.Fatal("failed adapter fell back", w.Code)
	}
}

func TestProviderConfigurationRejectsReinstalledFormHTTP(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	pack, err := p.installedPackage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const path = "/api/v1/extensions/mosaune.tmdb-search/configuration"
	w := admin.request("GET", path, nil)
	var before configuration.State
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &before) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	stale, err := p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	removeTMDBPackageFixture(t, p)
	if w = admin.request("GET", path, nil); w.Code != 404 {
		t.Fatal("uninstalled settings exposed", w.Code)
	}
	if err = p.installPackage(ctx, pack, "", false); err != nil {
		t.Fatal(err)
	}
	patch := configPatch(`{"language":"nl-NL"}`)
	if err = stale.write(ctx, patch); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale binding accepted", err)
	}
	patch.Revision = &before.Revision
	if w = admin.request("PUT", path, patch); w.Code != 409 {
		t.Fatal("stale form accepted", w.Code, w.Body.String())
	}
	fresh, err := p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := fresh.read(ctx)
	if after.Revision <= before.Revision || after.Values["language"] != nil {
		t.Fatal("failed write changed settings")
	}
	patch.Revision = &after.Revision
	if w = admin.request("PUT", path, patch); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestProviderConfigurationRollbackHidesDormantFields(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	current, err := p.installedPackage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	old, err := packageformat.Parse(providerDiscoveryExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	removeTMDBPackageFixture(t, p)
	if err = p.installPackage(ctx, old, "", false); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, current, old.Checksum, false); err != nil {
		t.Fatal(err)
	}
	bind, err := p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = bind.write(ctx, configPatch(`{"language":"nl-NL"}`)); err != nil {
		t.Fatal(err)
	}
	stale, err := p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, old, current.Checksum, true); err != nil {
		t.Fatal(err)
	}
	if err = stale.write(ctx, configPatch(`{"language":"fr-FR"}`)); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("old archive form accepted", err)
	}
	bind, err = p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := bind.read(ctx)
	if state.Values["language"] != nil {
		t.Fatal("dormant value exposed")
	}
	// The underlying store retains language for a later update. It cannot be
	// edited via the older signed schema, even alongside a valid secret clear.
	patch := configPatch(`{"language":"fr-FR"}`)
	patch.ClearSecrets = []string{"apiToken"}
	if err = bind.write(ctx, patch); err == nil {
		t.Fatal("dormant setting accepted")
	}
	if err = p.installPackage(ctx, current, old.Checksum, false); err != nil {
		t.Fatal(err)
	}
	bind, err = p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state, _ = bind.read(ctx)
	if string(state.Values["language"]) != `"nl-NL"` {
		t.Fatal("rollback lost value")
	}
}

func TestProviderConfigurationFailureIsAtomic(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	bind, err := p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := bind.read(ctx)
	if _, err = c.db.Exec(`CREATE TRIGGER reject_provider_settings BEFORE UPDATE ON extension_configuration BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if err = bind.write(ctx, configPatch(`{"language":"nl-NL","apiToken":"fixture-secret-not-persisted-123456"}`)); err == nil {
		t.Fatal("write failure ignored")
	}
	bind, err = p.Configuration(ctx)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := bind.read(ctx)
	if before.Revision != after.Revision || after.Values["language"] != nil || after.SecretsConfigured["apiToken"] {
		t.Fatal("partial write")
	}
	if _, err = c.db.Exec(`UPDATE provider_packages SET archive=x'00' WHERE id='tmdb'`); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.configurationFor(tmdbConfigurationOwner); ok {
		t.Fatal("corrupt package exposes fallback schema")
	}
}
