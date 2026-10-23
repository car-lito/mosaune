package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/examples/demo"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	ext "mosaune/extensionkit/runtime"
	"path/filepath"
	"sync"
	"testing"
)

func registryDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err = (registry.SQLStore{DB: db}).Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}
func registryEntry(id, cap string, deps ...string) registry.Entry {
	hash := sha256.Sum256(demo.Code)
	return registry.Entry{Descriptor: protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{cap}, Dependencies: deps}, CodeSHA256: hex.EncodeToString(hash[:])}
}
func registryCommit(t *testing.T, store registry.SQLStore, c registry.Change) registry.Snapshot {
	t.Helper()
	s, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Commit(context.Background(), s.Revision, c)
	if err != nil {
		t.Fatal(err)
	}
	return p.After
}
func seedRegistry(t *testing.T, store registry.SQLStore) registry.Snapshot {
	for _, entry := range []registry.Entry{registryEntry("test.player", "playback.video.v1"), registryEntry("test.alternative", "playback.video.v1"), registryEntry("test.films", "metadata.search.v1", "playback.video.v1"), registryEntry("test.series", "media.series.v1", "playback.video.v1")} {
		registryCommit(t, store, registry.Change{Operation: "install", ID: entry.Descriptor.ID, Entry: &entry})
	}
	for _, id := range []string{"test.player", "test.films", "test.series", "test.alternative"} {
		registryCommit(t, store, registry.Change{Operation: "enable", ID: id})
	}
	s, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func moduleStatus(t *testing.T, r registry.Report, id string) registry.Status {
	t.Helper()
	for _, m := range r.Modules {
		if m.ID == id {
			return m
		}
	}
	t.Fatal("status missing", id)
	return registry.Status{}
}

func TestRegistrySQLiteRestartAndRuntimeRestore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "registry.db")
	db := registryDB(t, path)
	store := registry.SQLStore{DB: db}
	snapshot := seedRegistry(t, store)
	verified := func(_ context.Context, entry registry.Entry) (registry.Artifact, error) {
		return registry.Artifact{Descriptor: entry.Descriptor, Code: demo.Code}, nil
	}
	engine, report, err := registry.Restore(ctx, snapshot, ext.DefaultLimits(), verified, false)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if moduleStatus(t, report, "test.films").State != "active" {
		t.Fatal(report)
	}
	bindings, err := engine.Bindings("test.films")
	if err != nil || bindings["playback.video.v1"] != "test.player" {
		t.Fatal(bindings, err)
	}
	if err = engine.Close(); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	// Reopen the database and create a new runtime: no in-memory choices survive.
	db = registryDB(t, path)
	store = registry.SQLStore{DB: db}
	restored, err := store.Load(ctx)
	if err != nil || restored.Revision != snapshot.Revision {
		t.Fatal(restored, err)
	}
	engine2, report, err := registry.Restore(ctx, restored, ext.DefaultLimits(), verified, false)
	if err != nil {
		t.Fatal(err)
	}
	defer engine2.Close()
	bindings, err = engine2.Bindings("test.films")
	if err != nil || bindings["playback.video.v1"] != "test.player" {
		t.Fatal("preference changed on restart", bindings, err)
	}
	out, err := engine2.Call(ctx, "test.films", protocol.Request{Protocol: 1, ID: "restart", Capability: "metadata.search.v1", Operation: "search", Payload: json.RawMessage(`{"kind":"films","title":"Voyage"}`)})
	if err != nil || out.Response.Error != nil {
		t.Fatal(out, err)
	}
}

func TestRegistryRestoreFailuresPreserveIntentAndIndependentModules(t *testing.T) {
	ctx := context.Background()
	store := registry.SQLStore{DB: registryDB(t, filepath.Join(t.TempDir(), "registry.db"))}
	snapshot := seedRegistry(t, store)
	before, _ := json.Marshal(snapshot)
	called := map[string]int{}
	verified := func(_ context.Context, entry registry.Entry) (registry.Artifact, error) {
		called[entry.Descriptor.ID]++
		if entry.Descriptor.ID == "test.player" {
			return registry.Artifact{}, errors.New("fixture missing")
		}
		return registry.Artifact{Descriptor: entry.Descriptor, Code: demo.Code}, nil
	}
	engine, report, err := registry.Restore(ctx, snapshot, ext.DefaultLimits(), verified, false)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if moduleStatus(t, report, "test.player").Code != "artifact_unavailable" || moduleStatus(t, report, "test.films").State != "blocked" || moduleStatus(t, report, "test.alternative").State != "active" {
		t.Fatal(report)
	}
	if called["test.films"] != 0 || called["test.series"] != 0 {
		t.Fatal("blocked consumers compiled")
	}
	after, err := store.Load(ctx)
	raw, _ := json.Marshal(after)
	if err != nil || string(raw) != string(before) {
		t.Fatal("failed restart rewrote intent", err)
	}
	engine.Close()
	called = map[string]int{}
	safe, report, err := registry.Restore(ctx, snapshot, ext.DefaultLimits(), verified, true)
	if err != nil || safe != nil || len(called) != 0 || moduleStatus(t, report, "test.films").State != "suspended" {
		t.Fatal("safe mode loaded code", report, err)
	}
	// Both code and descriptor must match the independently verified artifact.
	for _, mode := range []string{"code", "descriptor"} {
		t.Run(mode, func(t *testing.T) {
			source := func(_ context.Context, entry registry.Entry) (registry.Artifact, error) {
				a := registry.Artifact{Descriptor: entry.Descriptor, Code: demo.Code}
				if entry.Descriptor.ID == "test.player" {
					if mode == "code" {
						a.Code = []byte("altered")
					} else {
						a.Descriptor.Version = "9.0.0"
					}
				}
				return a, nil
			}
			e, r, err := registry.Restore(ctx, snapshot, ext.DefaultLimits(), source, false)
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			if moduleStatus(t, r, "test.player").Code != "artifact_mismatch" || moduleStatus(t, r, "test.films").State != "blocked" {
				t.Fatal(r)
			}
		})
	}
}

func TestRegistrySQLiteConflictAndFailedWrite(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "registry.db"))
	store := registry.SQLStore{DB: db}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, id := range []string{"test.one", "test.two"} {
		wait.Add(1)
		go func(id string) {
			defer wait.Done()
			<-start
			entry := registryEntry(id, "test.demo.v1")
			_, err := store.Commit(ctx, 0, registry.Change{Operation: "install", ID: id, Entry: &entry})
			results <- err
		}(id)
	}
	close(start)
	wait.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, registry.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal(success, conflicts)
	}
	s, err := store.Load(ctx)
	if err != nil || s.Revision != 1 || len(s.Entries) != 1 {
		t.Fatal(s, err)
	}
	if _, err = db.Exec(`CREATE TRIGGER reject_registry_write BEFORE UPDATE ON extension_registry_state BEGIN SELECT RAISE(ABORT,'fixture write error'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Commit(ctx, s.Revision, registry.Change{Operation: "enable", ID: s.Entries[0].Descriptor.ID}); err == nil {
		t.Fatal("failed write accepted")
	}
	after, err := store.Load(ctx)
	if err != nil || after.Revision != s.Revision || after.Entries[0].Enabled {
		t.Fatal("partial commit", after, err)
	}
}

func TestRegistryCorruptionDoesNotResetOrExecute(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "registry.db"))
	store := registry.SQLStore{DB: db}
	seedRegistry(t, store)
	if _, err := db.Exec(`UPDATE extension_registry_state SET data='{"schema":99,"revision":8,"entries":[]}'`); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	} // idempotent, never resets an existing registry
	if _, err := store.Load(ctx); err == nil {
		t.Fatal("corrupted registry accepted")
	}
	var raw string
	if err := db.QueryRow(`SELECT data FROM extension_registry_state`).Scan(&raw); err != nil || raw != `{"schema":99,"revision":8,"entries":[]}` {
		t.Fatal("corruption silently erased", raw, err)
	}
}

func TestProviderRegistryAtomicInstallAndRollback(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	first, err := packageformat.Parse(providerExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	next, err := packageformat.Parse(providerUpdateExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, first, "", false); err != nil {
		t.Fatal(err)
	}
	if err = p.setExtension(ctx, true); err != nil {
		t.Fatal(err)
	}
	entry, revision, err := p.registeredPackage(ctx, first)
	if err != nil || !entry.Enabled {
		t.Fatal(entry, err)
	}
	if _, err = c.db.Exec(`CREATE TEMP TRIGGER fail_registry BEFORE UPDATE ON extension_registry_state BEGIN SELECT RAISE(ABORT,'fixture registry failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, next, first.Checksum, false); err == nil {
		t.Fatal("registry failure did not abort archive replacement")
	}
	state, err := p.packageState(ctx)
	if err != nil || state.Package.Checksum != first.Checksum || state.PreviousChecksum != "" || state.Registration == nil || state.Registration.Revision != revision || !state.Runtime["enabled"] {
		t.Fatal(state, err)
	}
	if err = p.setExtension(ctx, false); err == nil {
		t.Fatal("configuration commit ignored registry failure")
	}
	var desired bool
	if err = c.db.QueryRow(`SELECT enabled FROM provider_runtime WHERE id='tmdb'`).Scan(&desired); err != nil || !desired {
		t.Fatal("partial activation write", err)
	}
	if err = p.removePackage(ctx, first.Checksum); err == nil {
		t.Fatal("removal ignored registry failure")
	}
	if _, err = c.db.Exec(`DROP TRIGGER fail_registry`); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, next, first.Checksum, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err = p.registeredPackage(ctx, next); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, first, next.Checksum, true); err != nil {
		t.Fatal(err)
	}
	entry, _, err = p.registeredPackage(ctx, first)
	if err != nil || !entry.Enabled {
		t.Fatal(entry, err)
	}
	if err = p.removePackage(ctx, first.Checksum); err != nil {
		t.Fatal(err)
	}
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if _, exists := registryEntryFor(s, "mosaune.tmdb-search"); err != nil || exists || len(s.Entries) != 3 {
		t.Fatal("provider removal affected recognition entries", s, err)
	}
}

func TestProviderRegistryMigrationPreservesLegacyChoice(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.Parse(providerUpdateExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	// Recreate a version-14 installation, without running a package during migration.
	removeRecognitionForLegacyFixture(t, c)
	if _, err = c.db.Exec(`INSERT INTO provider_packages(id,archive,checksum) VALUES('tmdb',?,?); INSERT INTO provider_runtime(id,enabled) VALUES('tmdb',1) ON CONFLICT(id) DO UPDATE SET enabled=1; DROP TABLE extension_registry_state;PRAGMA user_version=14;`, pack.Archive, pack.Checksum); err != nil {
		t.Fatal(err)
	}
	c.close()
	reopened, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	p := reopened.provider("tmdb").(*tmdbProvider)
	entry, _, err := p.registeredPackage(context.Background(), pack)
	if err != nil || !entry.Enabled || p.extensionStatus()["loaded"] {
		t.Fatal(entry, err)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("migration backup missing", backups)
	}
	engine, release, err := p.extensionEngine(context.Background())
	if err != nil || engine == nil {
		t.Fatal("migrated package cannot start", err)
	}
	release()
}
