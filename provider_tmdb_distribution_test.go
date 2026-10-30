package main

import (
	"context"
	"database/sql"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"testing"
)

func TestTMDBStarterIsAnOrdinaryInstalledPackage(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	expected, err := packageformat.Parse(providerCatalogueExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := p.installedPackage(ctx)
	if err != nil || pack.Checksum != expected.Checksum {
		t.Fatal("starter missing or different from catalogue", err)
	}
	entry, _, err := p.registeredPackage(ctx, pack)
	if err != nil || !entry.Enabled || p.extensionStatus()["loaded"] {
		t.Fatal("not registered lazily", err)
	}
	if err = p.setExtension(ctx, false); err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	kept, err := p.installedPackage(ctx)
	if err != nil || kept.Checksum != pack.Checksum || p.extensionStatus()["enabled"] || !p.extensionStatus()["installed"] {
		t.Fatal("starter was reprovisioned", err)
	}
}

func TestTMDBDistributionMigrationPreservesMissingPausedAndDenied(t *testing.T) {
	for _, wanted := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy-active", false: "legacy-removed-or-paused"}[wanted], func(t *testing.T) {
			c, dir := testCatalog(t)
			ctx := context.Background()
			p := c.provider("tmdb").(*tmdbProvider)
			removeTMDBPackageFixture(t, p)
			secret := "fixture-kept-key-distribution-12345"
			if err := c.setProviderCredential(ctx, "tmdb", secret); err != nil {
				t.Fatal(err)
			}
			if err := p.setProviderEnabled(ctx, false); err != nil {
				t.Fatal(err)
			}
			changeTMDBAccess(t, p, false)
			if _, err := c.db.Exec("UPDATE provider_runtime SET enabled=? WHERE id='tmdb'; PRAGMA user_version=28", wanted); err != nil {
				t.Fatal(err)
			}
			c.close()
			c, err := openCatalog(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			p = c.provider("tmdb").(*tmdbProvider)
			_, err = p.installedPackage(ctx)
			if wanted && err != nil || !wanted && err != sql.ErrNoRows {
				t.Fatal("incorrect migration choice", err)
			}
			status := p.extensionStatus()
			if status["installed"] != wanted || status["enabled"] != wanted || status["providerEnabled"] || status["effective"] || status["serviceAccess"] || status["loaded"] || c.credential(ctx, "tmdb") != secret {
				t.Fatal("migration changed consent, settings or started code", status)
			}
			if !wanted {
				if err = p.setExtension(ctx, true); err == nil {
					t.Fatal("activation without reinstall")
				}
			}
		})
	}
}

func TestTMDBDistributionMigrationIsAtomic(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	removeTMDBPackageFixture(t, p)
	if _, err := c.db.Exec("UPDATE provider_runtime SET enabled=1 WHERE id='tmdb'; PRAGMA user_version=28; CREATE TEMP TRIGGER fail_starter_registry BEFORE UPDATE ON extension_registry_state BEGIN SELECT RAISE(ABORT,'fixture registry failure'); END;"); err != nil {
		t.Fatal(err)
	}
	if err := migrateTMDBInstalledPackage(c.db); err == nil {
		t.Fatal("failed registry write accepted")
	}
	var v, n int
	if err := c.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if err := c.db.QueryRow("SELECT count(*) FROM provider_packages WHERE id='tmdb'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	snapshot, err := (registry.SQLStore{DB: c.db}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, exists := registryEntryFor(snapshot, tmdbConfigurationOwner)
	if v != 28 || n != 0 || exists {
		t.Fatal("partial starter installation", v, n, exists)
	}
	if _, err = c.db.Exec("DROP TRIGGER fail_starter_registry"); err != nil {
		t.Fatal(err)
	}
	if err = migrateTMDBInstalledPackage(c.db); err != nil {
		t.Fatal(err)
	}
}
