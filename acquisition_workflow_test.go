package main

import (
	"bytes"
	"context"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestAcquisitionWorkflowNoFallback(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	desc.ID = "org.example.acquisition.workflow"
	old := bytes.ReplaceAll(code, []byte("workflow-info"), []byte("disabled-info"))
	old = bytes.ReplaceAll(old, []byte("submission-workflow"), []byte("submission-disabled"))
	if bytes.Equal(code, old) {
		t.Fatal("missing fixture")
	}
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Workflow", Publisher: "Fixture", MinCore: "0.0.64", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	pack := packageformat.Executable{Manifest: manifest, Code: old}
	if validateExtractionCode(ctx, pack) == nil {
		t.Fatal("missing workflow info accepted")
	}
	pack.Manifest.MinCore = "0.0.63"
	if err = validateExtractionCode(ctx, pack); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(`{}`)) }))
	defer server.Close()
	client := arrClient{arrConfig{owner: c, ID: "radarr", URL: server.URL, Key: "host-only"}}
	session := &acquisitionSession{owner: c, descriptor: desc, code: old}
	_, err = client.submit(context.WithValue(ctx, acquisitionSessionKey{}, session), arrObject{"id": float64(12)}, -1)
	if err == nil || calls.Load() != 0 {
		t.Fatal("native fallback executed", err, calls.Load())
	}
	// The untrusted module contains its own validator. Altering its interned
	// binding changes both builder and validator there; the host must still reject.
	altered := bytes.ReplaceAll(code, []byte("destination.profile"), []byte("destination.hacked!"))
	if bytes.Equal(altered, code) {
		t.Fatal("missing binding fixture")
	}
	session = &acquisitionSession{owner: c, descriptor: desc, code: altered}
	_, err = client.submit(context.WithValue(ctx, acquisitionSessionKey{}, session), arrObject{}, -1)
	if err == nil || calls.Load() != 0 {
		t.Fatal("host trusted altered module binding", err, calls.Load())
	}
}

func TestAcquisitionWorkflowFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.5.0"
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
			if _, e = c.db.Exec("PRAGMA user_version=43"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionWorkflow(c.db); e != nil {
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
				want = "0.5.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
