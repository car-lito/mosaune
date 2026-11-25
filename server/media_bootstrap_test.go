package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestMediaBootstrapOptionValidation(t *testing.T) {
	for _, value := range []string{"films", "series,music", " films , music ", "none"} {
		if _, err := parseInitialMedia(value, true); err != nil {
			t.Fatal(value, err)
		}
	}
	for _, value := range []string{"", "film", "films,", "films,films", "none,music"} {
		if _, err := parseInitialMedia(value, true); err == nil {
			t.Fatal("invalid option accepted", value)
		}
	}
}

func assertMediaComposition(t *testing.T, c *catalog, selection mediaBootstrapSelection) {
	t.Helper()
	expected := map[string]bool{}
	screens := map[string]bool{}
	for _, kind := range selection.Families {
		m, err := bundledFamily(kind)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range m.Components {
			expected[v.Slot] = true
		}
		screens[m.Screen] = true
		if p := bundlePreview(t, c, kind, "prepare"); p.State != "ready" || len(p.Steps) != 0 {
			t.Fatal("incomplete family", kind, p)
		}
		var chosen string
		if err = c.db.QueryRow("SELECT id FROM media_bundle_screens WHERE kind=?", kind).Scan(&chosen); err != nil || chosen != m.Screen {
			t.Fatal("screen choice missing", kind, chosen, err)
		}
	}
	if selection.Home {
		screens["org.mosaune.screens.home"] = true
	}
	rows, err := c.db.Query("SELECT kind FROM recognition_packages")
	if err != nil {
		t.Fatal(err)
	}
	actual := map[string]bool{}
	for rows.Next() {
		var slot string
		if err = rows.Scan(&slot); err != nil {
			t.Fatal(err)
		}
		if actual[slot] || !expected[slot] {
			t.Fatal("unexpected/duplicate component", slot)
		}
		actual[slot] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(expected) {
		t.Fatal("missing components", actual, expected)
	}
	var count int
	if err = c.db.QueryRow("SELECT count(*) FROM recognition_selection").Scan(&count); err != nil || count != len(expected) {
		t.Fatal("unexpected selection", count, err)
	}
	if err = c.db.QueryRow("SELECT count(*) FROM media_bundles").Scan(&count); err != nil || count != len(selection.Families) {
		t.Fatal("unexpected families", count, err)
	}
	rows, err = c.db.Query("SELECT id,enabled FROM extensions")
	if err != nil {
		t.Fatal(err)
	}
	count = 0
	for rows.Next() {
		var id string
		var enabled bool
		if err = rows.Scan(&id, &enabled); err != nil {
			t.Fatal(err)
		}
		if !screens[id] || !enabled {
			t.Fatal("unexpected screen", id, enabled)
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil || count != len(screens) {
		t.Fatal("missing screens", count, err)
	}
	for slot := range expected {
		_, done, _, _, err := c.beginRecognition(context.Background(), slot)
		if err != nil {
			t.Fatal("component not usable", slot, err)
		}
		done()
	}
	if !expected["acquisition"] {
		if err = c.db.QueryRow("SELECT count(*) FROM extension_configuration WHERE owner LIKE 'mosaune.acquisition.%'").Scan(&count); err != nil || count != 0 {
			t.Fatal("unselected connector configured", count, err)
		}
	}
}

func TestMediaBootstrapIndependentCompositions(t *testing.T) {
	for _, selection := range []mediaBootstrapSelection{{}, {Families: []string{"films"}}, {Families: []string{"series"}}, {Families: []string{"music"}}, suppliedMediaSelection()} {
		raw, _ := json.Marshal(selection)
		t.Run(string(raw), func(t *testing.T) {
			dir := t.TempDir()
			c, err := openCatalogWithComposition(dir, nil, selection)
			if err != nil {
				t.Fatal(err)
			}
			assertMediaComposition(t, c, selection)
			assertNoInitialTMDB(t, c.db)
			c.close()
			// Later default launch does not fill in deliberately absent families.
			c, err = openCatalog(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			assertMediaComposition(t, c, selection)
			assertNoInitialTMDB(t, c.db)
		})
	}
}

func TestMediaBootstrapInstallRemoveSharedAndReopen(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalogWithComposition(dir, nil, mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"films", "series", "music"} {
		if err = bundleApply(c, bundlePreview(t, c, kind, "prepare")); err != nil {
			t.Fatal(err)
		}
	}
	if err = bundleApply(c, bundlePreview(t, c, "films", "remove")); err != nil {
		t.Fatal(err)
	}
	if err = bundleApply(c, bundlePreview(t, c, "series", "remove")); err != nil {
		t.Fatal(err)
	}
	if bundlePreview(t, c, "music", "prepare").State != "ready" {
		t.Fatal("shared components removed")
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	if bundlePreview(t, c, "films", "prepare").State != "absent" || bundlePreview(t, c, "series", "prepare").State != "absent" || bundlePreview(t, c, "music", "prepare").State != "ready" {
		t.Fatal("removed families restored")
	}
}

func TestMediaBootstrapAtomicRetryKeepsSelection(t *testing.T) {
	dir := t.TempDir()
	events := []string{}
	f := runtimeFactory("fixture", &events)
	f.InstallDefault = func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `CREATE TRIGGER fail_second_family BEFORE INSERT ON extensions WHEN NEW.id='org.mosaune.screens.series' BEGIN SELECT RAISE(ABORT,'fixture failure'); END`)
		return err
	}
	selection := mediaBootstrapSelection{Families: []string{"films", "series"}, Home: true}
	if c, err := openCatalogWithComposition(dir, []metadataProviderFactory{f}, selection); err == nil {
		c.close()
		t.Fatal("failure accepted")
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "mosaune.db"))
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"recognition_packages", "recognition_selection", "media_bundles", "extensions"} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial family install", table, n, err)
		}
	}
	var done bool
	if err = db.QueryRow("SELECT completed FROM media_bootstrap").Scan(&done); err != nil || done {
		t.Fatal("failure completed", err)
	}
	if _, err = db.Exec("DROP TRIGGER fail_second_family"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	c, err := openCatalogWithComposition(dir, nil, mediaBootstrapSelection{Families: []string{"music"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	assertMediaComposition(t, c, selection)
}

func TestMediaBootstrapLegacyChoicesPreserved(t *testing.T) {
	c, dir := testCatalog(t)
	if err := bundleApply(c, bundlePreview(t, c, "films", "remove")); err != nil {
		t.Fatal(err)
	}
	if err := bundleApply(c, bundlePreview(t, c, "music", "disable")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("DROP TABLE media_bootstrap; PRAGMA user_version=51"); err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	if bundlePreview(t, c, "films", "prepare").State != "absent" || bundlePreview(t, c, "music", "prepare").State != "disabled" || bundlePreview(t, c, "series", "prepare").State != "ready" {
		t.Fatal("legacy choices changed")
	}
}

func TestMediaBootstrapMalformedSelectionFailsClosed(t *testing.T) {
	c, err := openCatalogWithComposition(t.TempDir(), nil, mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	for _, raw := range []string{`null`, `{}`, `{"families":["unknown"]}`, `{"families":["films","films"]}`} {
		if _, err = c.db.Exec("UPDATE media_bootstrap SET completed=0,selection=?", raw); err != nil {
			t.Fatal(err)
		}
		if finishMediaBootstrap(context.Background(), c.db) == nil {
			t.Fatal("invalid selection accepted", raw)
		}
		var done bool
		if err = c.db.QueryRow("SELECT completed FROM media_bootstrap").Scan(&done); err != nil || done {
			t.Fatal("invalid completion", err)
		}
	}
}
