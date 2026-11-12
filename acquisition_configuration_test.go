package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/acquisitionconfig"
	"mosaune/extensionkit/configuration"
	"strings"
	"testing"
)

type countedConfigurationConnection struct {
	configuration.Connection
	reads int
}

func (c *countedConfigurationConnection) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	c.reads++
	return c.Connection.QueryRowContext(ctx, query, args...)
}

func TestAcquisitionConfigurationStorage(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	store := a.ConfigurationStore{DB: c.db}
	first, err := store.Load(ctx, "radarr")
	if err != nil {
		t.Fatal(err)
	}
	first.URL = "http://host/radarr"
	first.Key = "private-radarr"
	first.Enabled = true
	first.Profile = 3
	first.Root = "/films"
	first.Library = "fixture"
	if err = store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	stale := first
	stale.Key = "wrong"
	if err = store.Save(ctx, stale); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale write", err)
	}
	second, err := store.Load(ctx, "sonarr")
	if err != nil || second.Key != "" {
		t.Fatal("connector leak", err)
	}
	second.Key = "private-sonarr"
	if err = store.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	bound, _ := store.Bound("radarr")
	public, err := bound.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(public)
	if strings.Contains(string(raw), first.Key) || !public.SecretsConfigured["key"] {
		t.Fatal("secret exposed")
	}
	if _, err = store.Load(ctx, "mosaune.tmdb-search"); err == nil {
		t.Fatal("arbitrary owner allowed")
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	store = a.ConfigurationStore{DB: c.db}
	counted := &countedConfigurationConnection{Connection: c.db}
	if _, err = (a.ConfigurationStore{DB: counted}).Load(ctx, "radarr"); err != nil || counted.reads != 1 {
		t.Fatal("destination and secret must share one database snapshot", err, counted.reads)
	}
	loaded, err := store.Load(ctx, "radarr")
	if err != nil || loaded.Key != first.Key || loaded.Root != first.Root || loaded.Profile != 3 || !loaded.Enabled {
		t.Fatal(loaded, err)
	}
	if err = store.Pause(ctx, "radarr"); err != nil {
		t.Fatal(err)
	}
	paused, err := store.Load(ctx, "radarr")
	if err != nil || paused.Enabled || paused.Key != first.Key {
		t.Fatal(paused, err)
	}
	loaded.Key = "changed"
	if err = store.Save(ctx, loaded); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("pause overwritten", err)
	}
	// The public route cannot address shared connector owners by guessing their ID.
	if _, ok := c.configurationFor("mosaune.acquisition.radarr"); ok {
		t.Fatal("unregistered owner exposed")
	}
}

func TestAcquisitionConfigurationMigration(t *testing.T) {
	for _, mode := range []string{"valid", "conflict", "corrupt", "identity", "whitespace", "oversized", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			store := a.ConfigurationStore{DB: c.db}
			one := a.Configuration{ID: "radarr", URL: "http://nas/radarr", Key: "first-private", Enabled: false, Profile: 4, Root: "/films", Library: "films"}
			two := a.Configuration{ID: "sonarr", URL: "http://nas/sonarr", Key: "second-private", Enabled: true, Profile: 6, Root: "/series", Library: "series"}
			if mode == "conflict" {
				other := two
				other.Key = "other-private"
				if err := store.Save(ctx, other); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "whitespace" {
				two.Root = " /series "
			}
			if mode == "oversized" {
				two.Root = strings.Repeat("x", 4097)
			}
			raw, _ := json.Marshal(one)
			if _, err := c.db.Exec("INSERT INTO acquisition_settings(id,data) VALUES('radarr',?)", string(raw)); err != nil {
				t.Fatal(err)
			}
			raw, _ = json.Marshal(two)
			if mode == "corrupt" {
				raw = []byte("broken")
			}
			if mode == "identity" {
				two.ID = "radarr"
				raw, _ = json.Marshal(two)
			}
			id := "sonarr"
			if mode == "unknown" {
				id = "other"
				two.ID = id
				raw, _ = json.Marshal(two)
			}
			if _, err := c.db.Exec("INSERT INTO acquisition_settings(id,data) VALUES(?,?); PRAGMA user_version=45;", id, string(raw)); err != nil {
				t.Fatal(err)
			}
			err := migrateAcquisitionConfiguration(c.db)
			var count, version int
			c.db.QueryRow("SELECT COUNT(*) FROM acquisition_settings").Scan(&count)
			c.db.QueryRow("PRAGMA user_version").Scan(&version)
			if mode != "valid" && mode != "whitespace" {
				if err == nil || count != 2 || version != 45 {
					t.Fatal("failed migration changed state", err, count, version)
				}
				after, e := store.Load(ctx, "radarr")
				if e != nil || after.Key != "" {
					t.Fatal("partial migration", e)
				}
				return
			}
			if err != nil || count != 0 || version != 46 {
				t.Fatal(err, count, version)
			}
			for _, want := range []a.Configuration{one, two} {
				got, e := store.Load(ctx, want.ID)
				if e != nil || got.Key != want.Key || got.Enabled != want.Enabled || got.Root != want.Root || got.Profile != want.Profile {
					t.Fatal(got, e)
				}
			}
			if err = migrateAcquisitionConfiguration(c.db); err != nil {
				t.Fatal("not idempotent", err)
			}
		})
	}
}

func TestAcquisitionConfigurationStaleForm(t *testing.T) {
	c, h, token, cfg, _, _ := setupAcquisition(t, "films")
	input := c.acquisitionTestInput(t, cfg)
	saved := call(t, h, "PUT", "/api/v1/acquisition/settings", token, input)
	if saved.Code != 200 {
		t.Fatal(saved.Code, saved.Body.String())
	}
	stale := call(t, h, "PUT", "/api/v1/acquisition/settings", token, input)
	if stale.Code != 409 {
		t.Fatal("stale form accepted", stale.Code)
	}
	var response struct{ Config acquisitionSettingsView }
	if err := json.Unmarshal(saved.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	response.Config.Key = ""
	next := struct {
		arrConfig
		FormRevision string `json:"formRevision"`
	}{response.Config.arrConfig, response.Config.FormRevision}
	again := call(t, h, "PUT", "/api/v1/acquisition/settings", token, next)
	if again.Code != 200 {
		t.Fatal("fresh returned stamp unusable", again.Code, again.Body.String())
	}
	current, err := c.arrConfig(context.Background(), cfg.ID)
	if err != nil || current.Key != cfg.Key {
		t.Fatal("secret lost", err)
	}
	next.URL = "http://different-host"
	next.Key = ""
	denied := call(t, h, "POST", "/api/v1/acquisition/test", token, next)
	if denied.Code != 400 {
		t.Fatal("secret reused across hosts", denied.Code)
	}
}
