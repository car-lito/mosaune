package main

import (
	"context"
	"encoding/json"
	"mosaune/extensionkit/registry"
	"testing"
)

func TestBundleRollbackRoundtrip(t *testing.T) {
	for _, kind := range recognitionKinds {
		t.Run(kind, func(t *testing.T) {
			c, dir := testCatalog(t)
			first := testBundleArchive(t, kind, "1.0.0")
			next := testBundleArchive(t, kind, "1.1.0")
			importTestBundle(t, c, first)
			if e := bundleApply(c, bundlePreview(t, c, kind, "prepare")); e != nil {
				t.Fatal(e)
			}
			importTestBundle(t, c, next)
			c.close()
			var e error
			c, e = openCatalog(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer c.close()
			p := bundlePreview(t, c, kind, "restore")
			if p.PreviousVersion != "1.0.0" || len(p.Blockers) > 0 || len(p.Steps) < 2 {
				t.Fatal(p)
			}
			active, done, _, _, e := c.beginRecognition(context.Background(), kind)
			if e != nil {
				t.Fatal(e)
			}
			defer done()
			if e = bundleApply(c, p); e != nil {
				t.Fatal(e)
			}
			if active.Err() == nil {
				t.Fatal("previous running program not cancelled")
			}
			current := bundlePreview(t, c, kind, "prepare")
			if current.Archive.Manifest.Version != "1.0.0" || current.State != "ready" || current.PreviousVersion != "" {
				t.Fatal(current)
			}
			for _, s := range p.Steps {
				if s.Slot == "screen" {
					var raw string
					if e = c.db.QueryRow("SELECT manifest FROM extensions WHERE id=?", s.ID).Scan(&raw); e != nil {
						t.Fatal(e)
					}
					var m extensionManifest
					json.Unmarshal([]byte(raw), &m)
					if m.Version != "1.0.0" {
						t.Fatal(m.Version)
					}
				} else {
					snap, e := (registry.SQLStore{DB: c.db}).Load(context.Background())
					if e != nil {
						t.Fatal(e)
					}
					entry, ok := registryEntryFor(snap, s.ID)
					if !ok || entry.Descriptor.Version != "1.0.0" || !entry.Enabled {
						t.Fatal(entry)
					}
				}
			}
			if bundleApply(c, p) == nil {
				t.Fatal("replayed restore accepted")
			}
			importTestBundle(t, c, next)
			if e = bundleApply(c, bundlePreview(t, c, kind, "disable")); e != nil {
				t.Fatal(e)
			}
			if e = bundleApply(c, bundlePreview(t, c, kind, "restore")); e != nil {
				t.Fatal(e)
			}
			if bundlePreview(t, c, kind, "prepare").State != "disabled" {
				t.Fatal("restore enabled paused family")
			}
		})
	}
}
func TestBundleRollbackAtomicConflictAndPermissions(t *testing.T) {
	c, _, a := accountFixture(t)
	ctx := context.Background()
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	first := testBundleArchive(t, "films", "1.0.0")
	next := testBundleArchive(t, "films", "1.1.0")
	importTestBundle(t, c, first)
	importTestBundle(t, c, next)
	p := bundlePreview(t, c, "films", "restore")
	// Permissions do not advance the program registry; the preview must still expire.
	if _, e := c.db.Exec("INSERT INTO extension_grants(user_id,extension_id) SELECT id,? FROM accounts", first.Screen.Manifest.ID); e != nil {
		t.Fatal(e)
	}
	if bundleApply(c, p) == nil {
		t.Fatal("changed screen grants accepted")
	}
	p = bundlePreview(t, c, "films", "restore")
	if _, e := c.db.Exec("CREATE TEMP TRIGGER fail_family_restore BEFORE UPDATE ON extensions BEGIN SELECT RAISE(ABORT,'fixture failure'); END"); e != nil {
		t.Fatal(e)
	}
	if bundleApply(c, p) == nil {
		t.Fatal("injected failure ignored")
	}
	if _, e := c.db.Exec("DROP TRIGGER fail_family_restore"); e != nil {
		t.Fatal(e)
	}
	after := bundlePreview(t, c, "films", "restore")
	if p.Checksum != after.Checksum {
		t.Fatal("failed restore leaked writes")
	}
	if e := bundleApply(c, after); e != nil {
		t.Fatal(e)
	}
	var count int
	c.db.QueryRowContext(ctx, "SELECT count(*) FROM extension_grants WHERE extension_id=?", first.Screen.Manifest.ID).Scan(&count)
	if count != 1 {
		t.Fatal("grants lost", count)
	}
}
func TestBundleRollbackReplacementSharedAndIndependentUpdate(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	first := testBundleArchive(t, "films", "1.0.0")
	next := testBundleArchive(t, "films", "1.1.0")
	importTestBundle(t, c, first)
	importTestBundle(t, c, next)
	// Individual restore makes the family snapshot stale instead of replacing it.
	p := bundlePreview(t, c, "films", "restore")
	id := first.Modules["films"].Manifest.Module.ID
	if e := c.restoreRecognition(ctx, id, p.Revision); e != nil {
		t.Fatal(e)
	}
	blocked := bundlePreview(t, c, "films", "restore")
	if len(blocked.Blockers) == 0 || bundleApply(c, blocked) == nil {
		t.Fatal("independent update overwritten")
	}
	// An explicit alternative remains selected, even if absent. The original
	// program is now outside this family action and its separate change is kept.
	if _, e := c.db.Exec("UPDATE recognition_selection SET id='org.example.missing' WHERE kind='films'"); e != nil {
		t.Fatal(e)
	}
	if _, e := c.db.Exec("UPDATE media_bundle_screens SET id='org.example.screen' WHERE kind='films'"); e != nil {
		t.Fatal(e)
	}
	p = bundlePreview(t, c, "films", "restore")
	if len(p.Blockers) > 0 || len(p.Kept) < 2 {
		t.Fatal(p)
	}
	before, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if e := bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	after, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range before.Entries {
		if entry.Descriptor.ID == id || entry.Descriptor.ID == "mosaune.media.information" || entry.Descriptor.ID == "mosaune.media.header" {
			kept, ok := registryEntryFor(after, entry.Descriptor.ID)
			a, _ := json.Marshal(entry)
			b, _ := json.Marshal(kept)
			if !ok || string(a) != string(b) {
				t.Fatal("kept program modified", entry.Descriptor.ID)
			}
		}
	}
	var selected string
	c.db.QueryRow("SELECT id FROM recognition_selection WHERE kind='films'").Scan(&selected)
	if selected != "org.example.missing" {
		t.Fatal(selected)
	}
	if e := bundleApply(c, bundlePreview(t, c, "series", "prepare")); e != nil {
		t.Fatal("shared components damaged", e)
	}
}
func TestBundleRollbackHistoryMigrationRemovalAndRepair(t *testing.T) {
	c, dir := testCatalog(t)
	if _, e := c.db.Exec("DROP TABLE media_bundle_undo; PRAGMA user_version=31"); e != nil {
		t.Fatal(e)
	}
	c.close()
	var e error
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	if len(bundlePreview(t, c, "music", "restore").Blockers) == 0 {
		t.Fatal("invented history")
	}
	first := testBundleArchive(t, "music", "1.0.0")
	next := testBundleArchive(t, "music", "1.1.0")
	importTestBundle(t, c, first)
	importTestBundle(t, c, next)
	p := bundlePreview(t, c, "music", "restore")
	importTestBundle(t, c, next) // same-version repair must not destroy undo.
	if bundlePreview(t, c, "music", "restore").PreviousVersion != p.PreviousVersion {
		t.Fatal("repair replaced history")
	}
	c.extensionSafeMode = true
	blocked := bundlePreview(t, c, "music", "restore")
	if len(blocked.Blockers) == 0 || bundleApply(c, blocked) == nil {
		t.Fatal("safe mode bypass")
	}
	c.extensionSafeMode = false
	if e = bundleApply(c, bundlePreview(t, c, "music", "remove")); e != nil {
		t.Fatal(e)
	}
	var n int
	c.db.QueryRow("SELECT count(*) FROM media_bundle_undo").Scan(&n)
	if n != 0 {
		t.Fatal("removed family history retained")
	}
}
func TestBundleRollbackHTTPPermissions(t *testing.T) {
	c, h, a := accountFixture(t)
	a.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	importTestBundle(t, c, testBundleArchive(t, "films", "1.0.0"))
	importTestBundle(t, c, testBundleArchive(t, "films", "1.1.0"))
	p := bundlePreview(t, c, "films", "restore")
	in := map[string]any{"operation": "restore", "revision": p.Revision, "checksum": p.Checksum}
	a.create("Reader", accountTestPassword, nil)
	g := &accountClient{t: t, h: h}
	g.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	for _, action := range []string{"preview", "apply"} {
		if r := g.request("POST", "/api/v1/media-bundles/films/"+action, in); r.Code != 403 {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	p = bundlePreview(t, c, "films", "restore")
	in["checksum"] = p.Checksum
	if r := a.request("POST", "/api/v1/media-bundles/films/apply", in); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
}

func TestBundleRollbackMissingPreviousProgram(t *testing.T) {
	c, _ := testCatalog(t)
	first := testBundleArchive(t, "films", "1.0.0")
	next := testBundleArchive(t, "films", "1.1.0")
	importTestBundle(t, c, first)
	p := bundlePreview(t, c, "films", "prepare")
	id := first.Modules["films"].Manifest.Module.ID
	if e := c.changeRecognition(context.Background(), id, "remove", p.Revision); e != nil {
		t.Fatal(e)
	}
	importTestBundle(t, c, next)
	blocked := bundlePreview(t, c, "films", "restore")
	if len(blocked.Blockers) == 0 || bundleApply(c, blocked) == nil {
		t.Fatal("absent prior program fabricated")
	}
	if bundlePreview(t, c, "films", "prepare").Archive.Manifest.Version != "1.1.0" {
		t.Fatal("blocked restore changed family")
	}
}

func TestBundleRollbackKeepsUpdatedSharedProgram(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	first := testBundleArchive(t, "films", "1.0.0")
	next := testBundleArchive(t, "films", "1.1.0")
	importTestBundle(t, c, first)
	shared := first.Modules["detail-header"]
	_, rev, e := c.recognitionPackages(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.importRecognition(ctx, first.Raw["detail-header"], shared.Checksum, rev); e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"enable", "select"} {
		_, rev, e = c.recognitionPackages(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = c.changeRecognition(ctx, shared.Manifest.Module.ID, op, rev); e != nil {
			t.Fatal(e)
		}
	}
	importTestBundle(t, c, next)
	p := bundlePreview(t, c, "films", "restore")
	found := false
	for _, s := range p.Kept {
		if s.ID == shared.Manifest.Module.ID && s.Shared {
			found = true
		}
	}
	if !found {
		t.Fatal("shared exception absent from preview", p)
	}
	if e = bundleApply(c, p); e != nil {
		t.Fatal(e)
	}
	snap, e := (registry.SQLStore{DB: c.db}).Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	entry, ok := registryEntryFor(snap, shared.Manifest.Module.ID)
	if !ok || !entry.Enabled || entry.Descriptor.Version != "1.1.0" {
		t.Fatal("shared program downgraded", entry)
	}
	if e = bundleApply(c, bundlePreview(t, c, "series", "prepare")); e != nil {
		t.Fatal("other family damaged", e)
	}
}
