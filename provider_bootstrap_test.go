package main

import (
	"context"
	"database/sql"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"path/filepath"
	"testing"
)

func assertNoInitialTMDB(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		"SELECT COUNT(*) FROM provider_packages WHERE id='tmdb'",
		"SELECT COUNT(*) FROM provider_runtime WHERE id='tmdb'",
		"SELECT COUNT(*) FROM extension_configuration WHERE owner='mosaune.tmdb-search'",
		"SELECT COUNT(*) FROM extension_authorizations WHERE owner='mosaune.tmdb-search'",
	} {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil || n != 0 {
			t.Fatal(q, n, err)
		}
	}
	s, err := (registry.SQLStore{DB: db}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registryEntryFor(s, tmdbConfigurationOwner); ok {
		t.Fatal("TMDB registered without selection")
	}
}

func TestProviderBootstrapEmptySelectionPersists(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalogWithProviders(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNoInitialTMDB(t, c.db)
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	assertNoInitialTMDB(t, c.db)
	p := c.provider("tmdb").(*tmdbProvider)
	if providerAvailable(p) || p.extensionStatus()["installed"] {
		t.Fatal("later default composition restored TMDB")
	}
	pack, err := packageformat.Parse(providerIdentifiersExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(context.Background(), pack, "", false); err != nil {
		t.Fatal(err)
	}
	if err = p.setExtension(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if providerAvailable(p) {
		t.Fatal("manual install granted service access")
	}
	changeTMDBAccess(t, p, true)
	if !providerAvailable(p) {
		t.Fatal("explicit manual consent ineffective")
	}
}

func TestProviderBootstrapAtomicRetryKeepsOriginalSelection(t *testing.T) {
	dir := t.TempDir()
	events := []string{}
	failure := errors.New("fixture installation failure")
	extra := runtimeFactory("extra", &events)
	extra.InstallDefault = func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO provider_states(id,enabled) VALUES('extra',1)"); err != nil {
			return err
		}
		return failure
	}
	factories := append(suppliedMetadataProviders(), extra)
	if c, err := openCatalogWithProviders(dir, factories); !errors.Is(err, failure) {
		if c != nil {
			c.close()
		}
		t.Fatal("expected initialization failure", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "mosaune.db"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoInitialTMDB(t, db)
	var n, v int
	var done bool
	if err = db.QueryRow("SELECT COUNT(*) FROM provider_states WHERE id='extra'").Scan(&n); err != nil || n != 0 {
		t.Fatal("partial callback committed", n, err)
	}
	if err = db.QueryRow("SELECT completed FROM provider_bootstrap").Scan(&done); err != nil || done {
		t.Fatal("failure marked complete", err)
	}
	if err = db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 50 {
		t.Fatal("failed install advanced schema", v, err)
	}
	db.Close()
	if c, err := openCatalogWithProviders(dir, suppliedMetadataProviders()); err == nil {
		c.close()
		t.Fatal("lost original selection during retry")
	}
	extra.InstallDefault = func(ctx context.Context, tx *sql.Tx) error {
		_, e := tx.ExecContext(ctx, "INSERT INTO provider_states(id,enabled) VALUES('extra',1)")
		return e
	}
	late := runtimeFactory("late", &events)
	lateCalls := 0
	late.InstallDefault = func(context.Context, *sql.Tx) error { lateCalls++; return nil }
	c, err := openCatalogWithProviders(dir, append(suppliedMetadataProviders(), extra, late))
	if err != nil {
		t.Fatal(err)
	}
	if lateCalls != 0 {
		t.Fatal("new defaults added to recorded selection")
	}
	if _, err = c.provider("tmdb").(*tmdbProvider).installedPackage(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = c.db.QueryRow("SELECT completed FROM provider_bootstrap").Scan(&done); err != nil || !done {
		t.Fatal("completion missing", err)
	}
	c.close()
	// A completed bootstrap does not call installers again, even if one would fail.
	extra.InstallDefault = func(context.Context, *sql.Tx) error { return failure }
	c, err = openCatalogWithProviders(dir, append(suppliedMetadataProviders(), extra))
	if err != nil {
		t.Fatal("completed bootstrap replayed", err)
	}
	c.close()
}

func TestProviderBootstrapLegacyChoicesAreNotProvisionedAgain(t *testing.T) {
	for _, removed := range []bool{false, true} {
		t.Run(map[bool]string{false: "paused", true: "removed"}[removed], func(t *testing.T) {
			c, dir := testCatalog(t)
			p := c.provider("tmdb").(*tmdbProvider)
			ctx := context.Background()
			const secret = "fixture-bootstrap-kept-key-123456"
			if err := c.setProviderCredential(ctx, "tmdb", secret); err != nil {
				t.Fatal(err)
			}
			changeTMDBAccess(t, p, false)
			if removed {
				removeTMDBPackageFixture(t, p)
			} else if err := p.setExtension(ctx, false); err != nil {
				t.Fatal(err)
			}
			if _, err := c.db.Exec("DROP TABLE provider_bootstrap; PRAGMA user_version=50"); err != nil {
				t.Fatal(err)
			}
			c.close()
			c, err := openCatalog(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			p = c.provider("tmdb").(*tmdbProvider)
			if p.extensionStatus()["enabled"] || providerAvailable(p) || c.credential(ctx, "tmdb") != secret {
				t.Fatal("legacy choices changed")
			}
			_, err = p.installedPackage(ctx)
			if removed && err != sql.ErrNoRows || !removed && err != nil {
				t.Fatal("legacy package changed", err)
			}
		})
	}
}

func TestProviderBootstrapMalformedSelectionFailsClosed(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	for _, raw := range []string{`null`, `{}`, `["unknown"]`, `["tmdb","tmdb"]`} {
		if _, err := c.db.Exec("UPDATE provider_bootstrap SET completed=0,providers=?", raw); err != nil {
			t.Fatal(err)
		}
		if err := finishProviderBootstrap(context.Background(), c.db, suppliedMetadataProviders()); err == nil {
			t.Fatal("invalid selection accepted", raw)
		}
		var done bool
		if err := c.db.QueryRow("SELECT completed FROM provider_bootstrap").Scan(&done); err != nil || done {
			t.Fatal("invalid selection mutated", err)
		}
	}
}
