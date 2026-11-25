package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func configSchema() configuration.Schema {
	return configuration.Schema{Version: 1, Title: "Fixture", Fields: []configuration.Field{
		{ID: "token", Label: "Key", Type: "secret", MinLength: 20, MaxLength: 100, Format: "token"},
		{ID: "name", Label: "Name", Type: "text", MaxLength: 100},
		{ID: "enabled", Label: "Enabled", Type: "boolean"},
	}}
}
func configPatch(values string) configuration.Patch {
	p := configuration.Patch{}
	_ = json.Unmarshal([]byte(values), &p.Values)
	return p
}
func TestConfigurationStoreIsolationAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "config.db")
	db := registryDB(t, path)
	store := configuration.SQLStore{DB: db}
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"test.first", "test.second"} {
		if err := store.Register(ctx, id, configSchema()); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := store.For("test.first")
	b, _ := store.For("test.second")
	const secret = "fixture-only-secret-123456789"
	if err := a.Apply(ctx, configPatch(`{"token":"`+secret+`","name":"Films","enabled":true}`)); err != nil {
		t.Fatal(err)
	}
	state, err := a.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(state)
	if strings.Contains(string(raw), secret) || !state.SecretsConfigured["token"] || string(state.Values["name"]) != `"Films"` {
		t.Fatal("invalid public state", string(raw))
	}
	other, err := b.Read(ctx)
	if err != nil || len(other.Values) > 0 || other.SecretsConfigured["token"] {
		t.Fatal("settings crossed owner", err)
	}
	for _, patch := range []configuration.Patch{configPatch(`{"owner":"test.second","token":"replacement-secret-123456"}`), {Values: map[string]json.RawMessage{"token": json.RawMessage(`"short"`)}, ClearSecrets: []string{"token"}}} {
		if a.Apply(ctx, patch) == nil {
			t.Fatal("invalid patch accepted")
		}
	}
	got, err := a.Secret(ctx, "token")
	if err != nil || got != secret {
		t.Fatal("invalid patch partly applied", err)
	}
	if _, err = a.Secret(ctx, "name"); err == nil {
		t.Fatal("ordinary field returned as credential")
	}
	db.Close()
	db = registryDB(t, path)
	store = configuration.SQLStore{DB: db}
	a, _ = store.For("test.first")
	b, _ = store.For("test.second")
	got, err = a.Secret(ctx, "token")
	if err != nil || got != secret {
		t.Fatal("restart lost credential", err)
	}
	if err = a.Remove(ctx, state.Revision-1); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale removal", err)
	}
	if err = a.Remove(ctx, state.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Read(ctx); err != sql.ErrNoRows {
		t.Fatal("removed owner still exists", err)
	}
	if _, err = b.Read(ctx); err != nil {
		t.Fatal("removal affected another owner", err)
	}
}
func TestConfigurationSchemaChangesAndCorruption(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "config.db"))
	store := configuration.SQLStore{DB: db}
	_ = store.Initialize(ctx)
	if err := store.Register(ctx, "test.first", configSchema()); err != nil {
		t.Fatal(err)
	}
	a, _ := store.For("test.first")
	if err := a.Apply(ctx, configPatch(`{"token":"fixture-secret-preserved-1234","name":"Films"}`)); err != nil {
		t.Fatal(err)
	}
	before, _ := a.Read(ctx)
	for _, mutate := range []func(*configuration.Schema){func(s *configuration.Schema) { s.Fields = s.Fields[1:] }, func(s *configuration.Schema) { s.Fields[0].Type = "text" }, func(s *configuration.Schema) { s.Fields[1].MaxLength = 2 }} {
		s := configSchema()
		mutate(&s)
		if err := store.Register(ctx, "test.first", s); !errors.Is(err, configuration.ErrIncompatible) {
			t.Fatal("unsafe schema change", err)
		}
		state, _ := a.Read(ctx)
		if state.Revision != before.Revision {
			t.Fatal("schema failure mutated revision")
		}
	}
	next := configSchema()
	next.Fields = append(next.Fields, configuration.Field{ID: "extra", Label: "Extra", Type: "text", MaxLength: 20})
	if err := store.Register(ctx, "test.first", next); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(ctx, configPatch(`{"extra":"kept"}`)); err != nil {
		t.Fatal(err)
	}
	if err := store.Register(ctx, "test.first", configSchema()); !errors.Is(err, configuration.ErrIncompatible) {
		t.Fatal("rollback discarded populated field", err)
	}
	if _, err := db.Exec(`UPDATE extension_configuration SET secrets_json='{"name":"leak"}' WHERE owner='test.first'`); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Read(ctx); !errors.Is(err, configuration.ErrStored) {
		t.Fatal("corruption accepted", err)
	}
	if err := store.Register(ctx, "test.first", configSchema()); !errors.Is(err, configuration.ErrStored) {
		t.Fatal("corruption silently reset", err)
	}
}
func TestConfigurationConcurrentEditsAndWriteFailure(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "config.db"))
	store := configuration.SQLStore{DB: db}
	_ = store.Initialize(ctx)
	if err := store.Register(ctx, "test.first", configSchema()); err != nil {
		t.Fatal(err)
	}
	b, _ := store.For("test.first")
	var revision int64
	start := make(chan struct{})
	out := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"One", "Two"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			<-start
			p := configPatch(`{"name":"` + name + `"}`)
			p.Revision = &revision
			out <- b.Apply(ctx, p)
		}(name)
	}
	close(start)
	wg.Wait()
	close(out)
	ok, conflict := 0, 0
	for err := range out {
		if err == nil {
			ok++
		} else if errors.Is(err, configuration.ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal(ok, conflict)
	}
	state, _ := b.Read(ctx)
	if _, err := db.Exec(`CREATE TRIGGER fail_settings BEFORE UPDATE ON extension_configuration BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := b.Apply(ctx, configPatch(`{"token":"fixture-new-secret-123456","name":"changed"}`)); err == nil {
		t.Fatal("write failure ignored")
	}
	after, _ := b.Read(ctx)
	if after.Revision != state.Revision || after.SecretsConfigured["token"] || string(after.Values["name"]) != string(state.Values["name"]) {
		t.Fatal("partial write")
	}
}
func TestConfigurationMigrationAndSharedOwner(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	const key = "fixture-migrated-token-12345678"
	removeRecognitionForLegacyFixture(t, c)
	if _, err = c.db.Exec(`DROP TABLE extension_configuration;INSERT INTO provider_settings VALUES('tmdb',?);PRAGMA user_version=15;`, key); err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	ctx := context.Background()
	if c.credential(ctx, "tmdb") != key {
		t.Fatal("migration lost credential")
	}
	var n int
	if err = c.db.QueryRow(`SELECT COUNT(*) FROM provider_settings WHERE id='tmdb'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("legacy copy retained", err)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("missing backup")
	}
	pack, err := packageformat.Parse(providerExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	p := c.provider("tmdb").(*tmdbProvider)
	removeTMDBPackageFixture(t, p)
	if err = p.installPackage(ctx, pack, "", false); err != nil {
		t.Fatal(err)
	}
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	if err = p.setExtension(ctx, false); err != nil {
		t.Fatal(err)
	}
	if err = p.removePackage(ctx, pack.Checksum); err != nil {
		t.Fatal(err)
	}
	if c.credential(ctx, "tmdb") != key {
		t.Fatal("removing implementation erased shared credential")
	}
}
func TestConfigurationHTTPConflictAndNoSecretExposure(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	path := "/api/v1/extensions/bundled/mosaune.tmdb-search/configuration"
	first := admin.request("GET", path, nil)
	var state configuration.State
	if err := json.Unmarshal(first.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	key := "fixture-newer-credential-12345678"
	if err := c.setProviderCredential(context.Background(), "tmdb", key); err != nil {
		t.Fatal(err)
	}
	stale := admin.request("PUT", path, map[string]any{"revision": state.Revision, "clearSecrets": []string{"apiToken"}})
	if stale.Code != 409 || strings.Contains(stale.Body.String(), key) || c.credential(context.Background(), "tmdb") != key {
		t.Fatal("stale edit lost secret", stale.Code)
	}
	fresh := admin.request("GET", path, nil)
	if fresh.Code != 200 || strings.Contains(fresh.Body.String(), key) {
		t.Fatal("credential exposed")
	}
}

func TestConfigurationFailureRollsBackPackageAndRegistry(t *testing.T) {
	c, err := openCatalog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	removeTMDBPackageFixture(t, p)
	p.extension.key = pub
	signed := func(example []byte, title string) packageformat.Package {
		t.Helper()
		pack, e := packageformat.Parse(example, trustedProviderKey(), version)
		if e != nil {
			t.Fatal(e)
		}
		pack.Manifest.Settings.Title = title
		raw, e := packageformat.Create(pack.Manifest, pack.Code, key)
		if e != nil {
			t.Fatal(e)
		}
		pack, e = packageformat.Parse(raw, pub, version)
		if e != nil {
			t.Fatal(e)
		}
		return pack
	}
	first := signed(providerExample, "Before")
	next := signed(providerUpdateExample, "After")
	removeTMDBPackageFixture(t, p)
	if err = p.installPackage(ctx, first, "", false); err != nil {
		t.Fatal(err)
	}
	if err = c.setProviderCredential(ctx, "tmdb", "fixture-kept-secret-123456789"); err != nil {
		t.Fatal(err)
	}
	old, _ := p.packageState(ctx)
	before, _ := tmdbConfiguration(c.db).Read(ctx)
	if _, err = c.db.Exec(`CREATE TEMP TRIGGER fail_configuration BEFORE UPDATE ON extension_configuration BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, next, first.Checksum, false); err == nil {
		t.Fatal("failed configuration did not stop update")
	}
	current, e := p.packageState(ctx)
	if e != nil || current.Package.Checksum != first.Checksum || current.Registration.Revision != old.Registration.Revision {
		t.Fatal("partial package/registry update", e)
	}
	after, e := tmdbConfiguration(c.db).Read(ctx)
	if e != nil || after.Revision != before.Revision || after.Schema.Title != "Before" {
		t.Fatal("partial schema update", e)
	}
	if err = p.removePackage(ctx, first.Checksum); err == nil {
		t.Fatal("failed schema restore did not stop removal")
	}
	if _, err = c.db.Exec(`DROP TRIGGER fail_configuration`); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, next, first.Checksum, false); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, first, next.Checksum, true); err != nil {
		t.Fatal(err)
	}
	if err = p.removePackage(ctx, first.Checksum); err != nil {
		t.Fatal(err)
	}
	if c.credential(ctx, "tmdb") != "fixture-kept-secret-123456789" {
		t.Fatal("shared secret lost")
	}
}

func TestConfigurationMigrationFailurePreservesLegacyData(t *testing.T) {
	c, err := openCatalog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	removeRecognitionForLegacyFixture(t, c)
	if _, err = c.db.Exec(`DROP TABLE extension_configuration; INSERT INTO provider_settings VALUES('tmdb','invalid-old-key');PRAGMA user_version=15;`); err != nil {
		t.Fatal(err)
	}
	if err = migrateExtensionConfiguration(c.db); err == nil {
		t.Fatal("invalid legacy value silently discarded")
	}
	var version int
	var secret string
	if err = c.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 15 {
		t.Fatal("failed migration advanced schema", err)
	}
	if err = c.db.QueryRow(`SELECT secret FROM provider_settings WHERE id='tmdb'`).Scan(&secret); err != nil || secret != "invalid-old-key" {
		t.Fatal("failed migration changed credential", err)
	}
	var tables int
	if err = c.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='extension_configuration'`).Scan(&tables); err != nil || tables != 0 {
		t.Fatal("partial schema created", err)
	}
}

func TestConfigurationLimitsRejectBeforeWriting(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "config.db"))
	store := configuration.SQLStore{DB: db}
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	huge := configuration.Schema{Version: 1, Title: "Large"}
	for i := 0; i < 32; i++ {
		f := configuration.Field{ID: fmt.Sprintf("field%d", i), Label: "Choice", Type: "select"}
		for j := 0; j < 32; j++ {
			f.Options = append(f.Options, configuration.Option{Value: fmt.Sprintf("%03d", j) + strings.Repeat("x", 125), Label: strings.Repeat("L", 160)})
		}
		huge.Fields = append(huge.Fields, f)
	}
	if err := huge.Validate(); err != nil {
		t.Fatal("fixture invalid", err)
	}
	if err := store.Register(ctx, "test.huge", huge); err == nil {
		t.Fatal("oversized schema accepted")
	}
	hugeBound, _ := store.For("test.huge")
	if _, err := hugeBound.Read(ctx); err != sql.ErrNoRows {
		t.Fatal("oversized schema partly stored", err)
	}
	for i := 0; i < 128; i++ {
		if err := store.Register(ctx, fmt.Sprintf("test.owner%d", i), configSchema()); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Register(ctx, "test.overflow", configSchema()); err == nil {
		t.Fatal("owner capacity exceeded")
	}
	if _, err := store.For("../test.owner0"); err == nil {
		t.Fatal("invalid owner accepted")
	}
}

func TestConfigurationGenericRoutesRequireRegisteredOwnerAndAdmin(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	ctx := context.Background()
	store := configuration.SQLStore{DB: c.db}
	if err := store.Register(ctx, "test.sample", configSchema()); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/extensions/test.sample/configuration"
	if w := admin.request("GET", path, nil); w.Code != 404 {
		t.Fatal("orphan storage exposed", w.Code)
	}
	entry := registryEntry("test.sample", "fixture.sample.v1")
	registryCommit(t, registry.SQLStore{DB: c.db}, registry.Change{Operation: "install", ID: entry.Descriptor.ID, Entry: &entry})
	read := admin.request("GET", path, nil)
	if read.Code != 200 {
		t.Fatal(read.Code, read.Body.String())
	}
	if w := admin.request("PUT", path, map[string]any{"values": map[string]any{"name": "Generic configuration", "token": "fixture-sample-token-123456789"}}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	read = admin.request("GET", path, nil)
	if !strings.Contains(read.Body.String(), "Generic configuration") || strings.Contains(read.Body.String(), "fixture-sample-token") {
		t.Fatal("wrong public settings")
	}
	if c.credential(ctx, "tmdb") != "" {
		t.Fatal("sample wrote another owner's key")
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	viewer := &accountClient{t: t, h: h}
	viewer.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "PUT"} {
		if w := viewer.request(method, path, map[string]any{}); w.Code != 403 {
			t.Fatal("unauthorized generic settings access", method, w.Code)
		}
	}
	if w := admin.request("PUT", path, map[string]any{"owner": "mosaune.tmdb-search", "values": map[string]any{"name": "evil"}}); w.Code != 400 {
		t.Fatal("owner override accepted", w.Code)
	}
}
