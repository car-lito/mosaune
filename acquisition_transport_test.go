package main

import (
	"bytes"
	"context"
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAcquisitionTransportFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.2.0"
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
			if _, e = c.db.Exec("PRAGMA user_version=40"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionTransport(c.db); e != nil {
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
				want = "0.2.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}

func TestAcquisitionTransportReplacement(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(code, []byte("api/v3")) || !bytes.Contains(code, []byte("8388608")) {
		t.Fatal("missing transport fixture")
	}
	code = bytes.ReplaceAll(code, []byte("api/v3"), []byte("api/v4"))
	code = bytes.ReplaceAll(code, []byte("8388608"), []byte("1048576"))
	desc.ID = "org.example.acquisition.transport"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Routes personnalisées", Publisher: "Fixture", MinCore: "0.0.61", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	raw, err := packageformat.CreateExecutable(manifest, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	installRecognitionExample(t, c, raw, pack)
	for _, op := range []string{"enable", "select"} {
		state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = c.changeRecognition(ctx, desc.ID, op, state.Revision); err != nil {
			t.Fatal(err)
		}
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	calls := 0
	oversized := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/radarr/api/v4/movie" || r.URL.Query().Get("tmdbId") != "42" || r.Header.Get("X-Api-Key") != "secret-host-only" || r.Method != "GET" {
			t.Error("installed route ignored", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if oversized {
			w.Write(bytes.Repeat([]byte(" "), 1048577))
			return
		}
		w.Write([]byte("[]"))
	}))
	defer server.Close()
	client := arrClient{arrConfig{ID: "radarr", URL: server.URL + "/radarr", Key: "secret-host-only", owner: c}}
	var out []arrObject
	if err = client.exchange(ctx, "find", 42, 0, nil, &out); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	oversized = true
	if err = client.exchange(ctx, "find", 42, 0, nil, &out); err == nil || strings.Contains(err.Error(), "secret-host-only") || calls != 2 {
		t.Fatal("response limit ignored", err, calls)
	}
	if err = client.exchange(ctx, "find", 42, 0, map[string]string{"bad": "write"}, &out); err == nil || calls != 2 {
		t.Fatal("read changed to write", err, calls)
	}
	state, _ := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err = c.changeRecognition(ctx, desc.ID, "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	if err = client.exchange(ctx, "find", 42, 0, nil, &out); err == nil || calls != 2 {
		t.Fatal("disabled network request", err, calls)
	}
	// Import rejects malicious destinations before any key or HTTP call exists.
	bad := pack
	bad.Code = bytes.ReplaceAll(code, []byte("api/v4"), []byte("../bad"))
	if err = validateExtractionCode(ctx, bad); err == nil {
		t.Fatal("unsafe route accepted")
	}
	old := pack
	old.Code = bytes.ReplaceAll(code, []byte("describe-transport"), []byte("disabled-transport"))
	if err = validateExtractionCode(ctx, old); err == nil {
		t.Fatal("missing transport accepted")
	}
	// A historical module may still be installed, but cannot silently use native routes.
	old.Manifest.MinCore = "0.0.60"
	if err = validateExtractionCode(ctx, old); err != nil {
		t.Fatal(err)
	}
	session := &acquisitionSession{owner: c, descriptor: desc, code: old.Code}
	if _, err = client.transportProfile(context.WithValue(ctx, acquisitionSessionKey{}, session)); err == nil {
		t.Fatal("native fallback")
	}
}

func TestAcquisitionTransportDeclaredOperations(t *testing.T) {
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reader, err := loadFactProjector(ctx, desc, code, a.Capability)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.engine.Close()
	for _, connector := range []string{"radarr", "sonarr"} {
		p, err := readAcquisitionTransport(ctx, reader, connector)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(p)
		if bytes.Contains(raw, []byte("secret")) || p.MaxBytes != 8<<20 || p.TimeoutSeconds != 12 {
			t.Fatal(p)
		}
	}
}
