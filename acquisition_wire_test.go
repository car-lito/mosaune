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
	"sync/atomic"
	"testing"
)

func TestAcquisitionWireReplacement(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{`"id": "id"`, `"id": "pk"`}, {`"externalId": "tmdbId"`, `"externalId": "filmId"`}, {`"monitored": "monitored"`, `"monitored": "watchFlag"`}, {"MoviesSearch", "CinemaSearch"}} {
		if len(pair[0]) != len(pair[1]) || !bytes.Contains(code, []byte(pair[0])) {
			t.Fatal("invalid fixture", pair)
		}
		code = bytes.ReplaceAll(code, []byte(pair[0]), []byte(pair[1]))
	}
	desc.ID = "org.example.acquisition.formats"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Formats personnalisés", Publisher: "Fixture", MinCore: "0.0.63", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
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
	var calls, writes atomic.Int32
	var ambiguous atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Api-Key") != "host-only" {
			t.Error("credential missing")
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v3/movie":
			if ambiguous.Load() {
				w.Write([]byte(`[{"pk":12,"id":13,"filmId":42}]`))
				return
			}
			w.Write([]byte(`[{"pk":12,"filmId":42,"path":"/keep/film","qualityProfileId":87,"foreign":9007199254740993,"watchFlag":false}]`))
		case "PUT /api/v3/movie/12":
			writes.Add(1)
			var obj arrObject
			d := json.NewDecoder(r.Body)
			d.UseNumber()
			if d.Decode(&obj) != nil || number(obj["pk"]) != 12 || number(obj["filmId"]) != 42 || !truth(obj["watchFlag"]) || number(obj["qualityProfileId"]) != 87 || word(obj["path"]) != "/keep/film" || obj["foreign"] != json.Number("9007199254740993") {
				t.Error("resource changed", obj)
			}
			w.Write([]byte(`{}`))
		case "POST /api/v3/command":
			writes.Add(1)
			var obj struct {
				Name string `json:"name"`
				IDs  []int  `json:"movieIds"`
			}
			if json.NewDecoder(r.Body).Decode(&obj) != nil || obj.Name != "CinemaSearch" || len(obj.IDs) != 1 || obj.IDs[0] != 12 {
				t.Error("wrong search", obj)
			}
			w.Write([]byte(`{}`))
		default:
			t.Error("unexpected call", r.Method, r.URL.Path)
			http.Error(w, "fixture", 400)
		}
	}))
	defer server.Close()
	client := arrClient{arrConfig{owner: c, ID: "radarr", URL: server.URL, Key: "host-only", Root: "/new", Profile: 1}}
	obj, err := client.find(ctx, 42)
	if err != nil || number(obj["id"]) != 12 || number(obj["externalId"]) != 42 {
		t.Fatal("format not applied", err, obj)
	}
	if id, err := client.submit(ctx, obj, -1); err != nil || id != 12 || writes.Load() != 2 {
		t.Fatal("submission format", id, err, writes.Load())
	}
	ambiguous.Store(true)
	if _, err = client.find(ctx, 42); err == nil {
		t.Fatal("ambiguous resource accepted")
	}
	bad := pack
	bad.Code = bytes.ReplaceAll(code, []byte("describe-wire"), []byte("disabled-wire"))
	if validateExtractionCode(ctx, bad) == nil {
		t.Fatal("missing format accepted")
	}
	bad.Manifest.MinCore = "0.0.62"
	if err = validateExtractionCode(ctx, bad); err != nil {
		t.Fatal(err)
	}
	session := &acquisitionSession{owner: c, descriptor: desc, code: bad.Code}
	count := calls.Load()
	if err = client.exchange(context.WithValue(ctx, acquisitionSessionKey{}, session), "find", 42, 0, nil, &[]arrObject{}); err == nil || calls.Load() != count {
		t.Fatal("native fallback")
	}
	s, _ := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err = c.changeRecognition(ctx, desc.ID, "disable", s.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = client.find(ctx, 42); err == nil || calls.Load() != count {
		t.Fatal("disabled codec still active")
	}
}

func TestAcquisitionWireFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.4.0"
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
			if _, e = c.db.Exec("PRAGMA user_version=42"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionWire(c.db); e != nil {
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
				want = "0.4.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
