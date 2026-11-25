package main

import (
	"context"
	"database/sql"
	"errors"
	a "mosaune/extensionkit/acquisitionconfig"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"testing"
)

func retirementSeed(t *testing.T, c *catalog) a.Configuration {
	t.Helper()
	ctx := context.Background()
	s := a.ConfigurationStore{DB: c.db}
	for _, id := range []string{"radarr", "sonarr"} {
		v, err := s.Load(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		v.Key = "private-" + id
		v.URL = "http://example.test"
		v.Root = "/private"
		v.Library = "library"
		v.Enabled = true
		if err = s.Save(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	v, err := s.Load(ctx, "radarr")
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func retirementChange(t *testing.T, c *catalog, id, op string) error {
	t.Helper()
	s, err := (registry.SQLStore{DB: c.db}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return c.changeRecognition(context.Background(), id, op, s.Revision)
}
func TestAcquisitionRetirementSharingAndReinstall(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	old := retirementSeed(t, c)
	if err := bundleApply(c, bundlePreview(t, c, "films", "remove")); err != nil {
		t.Fatal(err)
	}
	shared, err := (a.ConfigurationStore{DB: c.db}).Load(ctx, "radarr")
	if err != nil || shared.Key != old.Key {
		t.Fatal("family removal erased shared connector", shared, err)
	}
	before := c.acquisitionSettings(ctx, c.configFromDocument(old)).FormRevision
	if err := retirementChange(t, c, "acquisition", "disable"); err != nil {
		t.Fatal(err)
	}
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	desc.ID = "org.example.acquisition.retained"
	raw, err := packageformat.CreateExecutable(packageformat.ExecutableManifest{Schema: 2, Name: "Shared", Publisher: "Fixture", MinCore: "0.0.69", Module: desc, Contributions: []packageformat.Contribution{{Role: "acquisition.plan.v1", Target: "video"}}}, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.importRecognition(ctx, raw, pack.Checksum, snap.Revision); err != nil {
		t.Fatal(err)
	}
	if err = retirementChange(t, c, "acquisition", "remove"); err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	stored, err := (a.ConfigurationStore{DB: c.db}).Load(ctx, "radarr")
	if err != nil || stored.Key != old.Key || stored.Revision != old.Revision {
		t.Fatal("inactive replacement lost settings", stored, err)
	}
	if err = retirementChange(t, c, desc.ID, "remove"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"radarr", "sonarr"} {
		b, _ := (a.ConfigurationStore{DB: c.db}).Bound(id)
		if _, err = b.Read(ctx); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("retired private state remains", err)
		}
	}
	empty, err := c.arrConfig(ctx, "radarr")
	if err != nil || empty.Configured || empty.Enabled || empty.URL != "" {
		t.Fatal(empty, err)
	}
	if err = retirementChange(t, c, "acquisition", "install"); err != nil {
		t.Fatal(err)
	}
	if err = retirementChange(t, c, "acquisition", "enable"); err != nil {
		t.Fatal(err)
	}
	stored, err = (a.ConfigurationStore{DB: c.db}).Load(ctx, "radarr")
	if err != nil || stored.Key != "" || stored.URL != "" || stored.Revision <= old.Revision {
		t.Fatal("reinstall reused state", stored, err)
	}
	if err = (a.ConfigurationStore{DB: c.db}).Save(ctx, old); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale settings revived", err)
	}
	after := c.acquisitionSettings(ctx, c.configFromDocument(stored)).FormRevision
	if before == "" || after == "" || before == after {
		t.Fatal("old form still valid")
	}
	if err = retirementChange(t, c, "acquisition", "remove"); err != nil {
		t.Fatal(err)
	}
	snap, err = (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.importRecognition(ctx, raw, pack.Checksum, snap.Revision); err != nil {
		t.Fatal(err)
	}
	b, _ := (a.ConfigurationStore{DB: c.db}).Bound("radarr")
	state, err := b.Read(ctx)
	if err != nil || state.SecretsConfigured["key"] || state.Revision <= stored.Revision {
		t.Fatal("external reinstall failed", state, err)
	}
}
func TestAcquisitionRetirementAtomic(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	old := retirementSeed(t, c)
	active, done, _, _, err := c.beginRecognition(ctx, "acquisition")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if _, err = c.db.Exec(`CREATE TRIGGER refuse_connector_retirement BEFORE DELETE ON extension_configuration WHEN OLD.owner='mosaune.acquisition.sonarr' BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if err = retirementChange(t, c, "acquisition", "remove"); err == nil {
		t.Fatal("partial removal committed")
	}
	if active.Err() != nil {
		t.Fatal("rollback cancelled active session")
	}
	for _, id := range []string{"radarr", "sonarr"} {
		v, e := (a.ConfigurationStore{DB: c.db}).Load(ctx, id)
		if e != nil || v.Key != "private-"+id || v.Revision != old.Revision {
			t.Fatal("partial configuration removal", v, e)
		}
	}
	var n int
	if err = c.db.QueryRow("SELECT count(*) FROM extension_configuration_epochs WHERE owner LIKE 'mosaune.acquisition.%'").Scan(&n); err != nil || n != 0 {
		t.Fatal("retirement counter escaped rollback", n, err)
	}
	if _, err = c.db.Exec("DROP TRIGGER refuse_connector_retirement"); err != nil {
		t.Fatal(err)
	}
	if err = retirementChange(t, c, "acquisition", "remove"); err != nil {
		t.Fatal(err)
	}
	if active.Err() == nil {
		t.Fatal("successful removal did not cancel")
	}
}
func TestAcquisitionRetirementMigration(t *testing.T) {
	for _, mode := range []string{"disabled", "removed"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			old := retirementSeed(t, c)
			if mode == "removed" {
				if _, err := c.db.Exec("DELETE FROM recognition_packages WHERE kind='acquisition'"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := retirementChange(t, c, "acquisition", "disable"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := c.db.Exec("PRAGMA user_version=49"); err != nil {
				t.Fatal(err)
			}
			if err := migrateAcquisitionRetirement(c.db); err != nil {
				t.Fatal(err)
			}
			v, err := (a.ConfigurationStore{DB: c.db}).Load(ctx, "radarr")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "disabled" && v.Key != old.Key || mode == "removed" && v.Key != "" {
				t.Fatal(mode, v)
			}
		})
	}
}
