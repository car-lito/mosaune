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

func TestAcquisitionObservationsNoFallback(t *testing.T) {
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	old := bytes.ReplaceAll(code, []byte("observation-plan"), []byte("observation-none"))
	if bytes.Equal(old, code) {
		t.Fatal("missing fixture")
	}
	desc.ID = "org.example.acquisition.observations"
	pack := packageformat.Executable{Manifest: packageformat.ExecutableManifest{Schema: 2, Name: "Observations", Publisher: "Fixture", MinCore: "0.0.65", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}, Code: old}
	if validateExtractionCode(ctx, pack) == nil {
		t.Fatal("missing plan imported")
	}
	pack.Manifest.MinCore = "0.0.64"
	if err = validateExtractionCode(ctx, pack); err != nil {
		t.Fatal(err)
	}
	client := arrClient{arrConfig{ID: "radarr"}}
	session := &acquisitionSession{descriptor: desc, code: old}
	if reader, err := client.trackingReader(context.WithValue(ctx, acquisitionSessionKey{}, session)); err == nil {
		reader.engine.Close()
		t.Fatal("old module used native fallback")
	}
}

func TestAcquisitionObservationsInstalledLimit(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	from, to := []byte(`"limit":20`), []byte(`"limit": 1`)
	if !bytes.Contains(code, from) {
		t.Fatal("missing policy fixture")
	}
	code = bytes.ReplaceAll(code, from, to)
	desc.ID = "org.example.acquisition.observations"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Suivi borné", Publisher: "Fixture", MinCore: "0.0.65", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	raw, err := packageformat.CreateExecutable(manifest, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateExtractionCode(ctx, pack); err != nil {
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
	directory := c.dataDir
	c.close()
	c, err = openCatalog(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v3/queue" {
			t.Error(r.Method, r.URL.Path)
		}
		rows := make([]arrObject, 100)
		for i := range rows {
			rows[i] = arrObject{"movieId": float64(12)}
		}
		json.NewEncoder(w).Encode(map[string]any{"records": rows, "totalRecords": 101})
	}))
	defer server.Close()
	client := arrClient{arrConfig{owner: c, ID: "radarr", URL: server.URL, Key: "private-key"}}
	session, done, err := client.openSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := client.trackingReader(session)
	if err != nil {
		done()
		t.Fatal(err)
	}
	_, _, err = client.trackingObservations(session, reader, acquisitionRequest{Season: -1}, arrObject{"id": float64(12)})
	reader.engine.Close()
	done()
	if err == nil || !strings.Contains(err.Error(), "limite") || calls != 1 {
		t.Fatal("installed limit ignored", err, calls)
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(ctx, desc.ID, "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	if _, done, err = client.openSession(ctx); err == nil {
		done()
		t.Fatal("disabled module still available")
	}
	if calls != 1 {
		t.Fatal("network used after disable")
	}
}
func TestAcquisitionObservationsFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.6.0"
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
			if _, e = c.db.Exec("PRAGMA user_version=44"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionObservations(c.db); e != nil {
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
				want = "0.6.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
