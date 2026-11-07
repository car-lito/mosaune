package main

import (
	"bytes"
	"context"
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"strings"
	"testing"
)

func (c *catalog) acquisitionTestInput(t *testing.T, cfg arrConfig) any {
	t.Helper()
	cfg.owner = c
	view := c.acquisitionSettings(context.Background(), cfg)
	if view.Form == nil {
		t.Fatal(view.FormError)
	}
	return struct {
		arrConfig
		FormRevision string `json:"formRevision"`
	}{cfg, view.FormRevision}
}

func TestAcquisitionSettingsReplacement(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	desc.ID = "org.example.acquisition.settings"
	changed := bytes.ReplaceAll(code, []byte("Profil qualité"), []byte("Profil maison!!"))
	if bytes.Equal(changed, code) {
		t.Fatal("missing fixture")
	}
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Formulaires", Publisher: "Fixture", MinCore: "0.0.62", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	raw, err := packageformat.CreateExecutable(manifest, changed)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	installRecognitionExample(t, c, raw, pack)
	cfg := arrConfig{ID: "radarr", Key: "never-in-form", owner: c}
	before := c.acquisitionSettings(ctx, cfg)
	for _, op := range []string{"enable", "select"} {
		s, _ := (registry.SQLStore{DB: c.db}).Load(ctx)
		if err = c.changeRecognition(ctx, desc.ID, op, s.Revision); err != nil {
			t.Fatal(err)
		}
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	cfg.owner = c
	after := c.acquisitionSettings(ctx, cfg)
	encoded, _ := json.Marshal(after)
	if after.Form == nil || after.FormRevision == before.FormRevision || !bytes.Contains(encoded, []byte("Profil maison!!")) || bytes.Contains(encoded, []byte(cfg.Key)) {
		t.Fatal("form replacement not applied or secret exposed", after.FormError)
	}
	bad := pack
	bad.Code = bytes.ReplaceAll(changed, []byte("describe-settings"), []byte("disabled-settings"))
	if validateExtractionCode(ctx, bad) == nil {
		t.Fatal("missing form accepted")
	}
	bad = pack
	bad.Code = bytes.ReplaceAll(changed, []byte(`"binding": "key"`), []byte(`"binding": "bad"`))
	if validateExtractionCode(ctx, bad) == nil {
		t.Fatal("unknown binding accepted")
	}
	s, _ := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err = c.changeRecognition(ctx, desc.ID, "disable", s.Revision); err != nil {
		t.Fatal(err)
	}
	disabled := c.acquisitionSettings(ctx, cfg)
	if disabled.Form != nil || disabled.FormError == "" {
		t.Fatal("native form fallback")
	}
}

func TestAcquisitionSettingsAPI(t *testing.T) {
	c, h, token, cfg, _, remote := setupAcquisition(t, "films")
	result := call(t, h, "GET", "/api/v1/acquisition/settings", token, nil)
	if result.Code != 200 || strings.Contains(result.Body.String(), cfg.Key) {
		t.Fatal("settings leaked secret", result.Code)
	}
	var views []acquisitionSettingsView
	if err := json.Unmarshal(result.Body.Bytes(), &views); err != nil {
		t.Fatal(err)
	}
	var view acquisitionSettingsView
	for _, v := range views {
		if v.ID == cfg.ID {
			view = v
		}
	}
	if view.Form == nil || view.FormRevision == "" {
		t.Fatal("form missing")
	}
	for _, revision := range []string{"", "stale"} {
		input := struct {
			arrConfig
			FormRevision string `json:"formRevision"`
		}{cfg, revision}
		r := call(t, h, "PUT", "/api/v1/acquisition/settings", token, input)
		if r.Code != 409 {
			t.Fatal("stale form accepted", r.Code, r.Body.String())
		}
	}
	input := struct {
		arrConfig
		FormRevision string `json:"formRevision"`
	}{cfg, view.FormRevision}
	input.Key = "" // Preserve host secret when the address is unchanged.
	r := call(t, h, "PUT", "/api/v1/acquisition/settings", token, input)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	saved, err := c.arrConfig(context.Background(), cfg.ID)
	if err != nil || saved.Key != cfg.Key {
		t.Fatal("secret lost", err)
	}
	var stored string
	if err = c.db.QueryRow("SELECT data FROM acquisition_settings WHERE id=?", cfg.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "formRevision") || strings.Contains(stored, "instructions") {
		t.Fatal("transient form persisted")
	}
	s, _ := (registry.SQLStore{DB: c.db}).Load(context.Background())
	if err = c.changeRecognition(context.Background(), "acquisition", "disable", s.Revision); err != nil {
		t.Fatal(err)
	}
	r = call(t, h, "POST", "/api/v1/acquisition/test", token, input)
	if r.Code != 400 {
		t.Fatal("disabled form accepted", r.Code)
	}
	r = call(t, h, "POST", "/api/v1/acquisition/radarr/pause", token, map[string]any{})
	if r.Code != 204 || len(remote.writes) != 0 {
		t.Fatal("pause unavailable or remote write", r.Code)
	}
}

func TestAcquisitionSettingsFactoryUpgrade(t *testing.T) {
	for _, mode := range []string{"enabled", "disabled", "removed", "modified", "imported"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			store := registry.SQLStore{DB: c.db}
			s, e := store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			entry, _ := registryEntryFor(s, "mosaune.acquisition.plan")
			entry.Descriptor.Version = "0.3.0"
			entry.Enabled = false
			plan, e := store.Commit(ctx, s.Revision, registry.Change{Operation: "restore", ID: entry.Descriptor.ID, Entry: &entry})
			if e != nil {
				t.Fatal(e)
			}
			if mode == "disabled" || mode == "removed" {
				op := "disable"
				if mode == "removed" {
					op = "remove"
				}
				if e = c.changeRecognition(ctx, "acquisition", op, plan.After.Revision); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "modified" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET code=? WHERE id=?", []byte("altered"), entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "imported" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET manifest='{}' WHERE id=?", entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = c.db.Exec("PRAGMA user_version=41"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionSettings(c.db); e != nil {
				t.Fatal(e)
			}
			s, e = store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			next, present := registryEntryFor(s, entry.Descriptor.ID)
			if mode == "removed" {
				if present {
					t.Fatal("removed package restored")
				}
				return
			}
			current, _, _ := modules.Lookup("acquisition")
			want := current.Version
			if mode == "modified" || mode == "imported" {
				want = "0.3.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
