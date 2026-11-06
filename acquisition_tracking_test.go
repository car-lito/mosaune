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

func TestAcquisitionTrackingFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.1.0"
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
			if _, e = c.db.Exec("PRAGMA user_version=39"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionTracking(c.db); e != nil {
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
			current, _ := modules.Descriptor("acquisition")
			want := current.Version
			if mode == "modified" || mode == "imported" {
				want = "0.1.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}

func TestAcquisitionTrackingReplacement(t *testing.T) {
	c, h, token, cfg, work, fixture := setupAcquisition(t, "films")
	if response := confirmAcquisition(t, h, token, work, -1); response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	from, to := []byte("En attente d’une disponibilité dans le service."), []byte("Surveille  d’une disponibilité dans le service.")
	// The declaration and fallback text may share or duplicate a data segment;
	// replace every occurrence, keeping the binary lengths unchanged.
	if len(from) != len(to) || !bytes.Contains(code, from) {
		t.Fatal("invalid replacement fixture", len(from), len(to))
	}
	code = bytes.ReplaceAll(code, from, to)
	desc.ID = "org.example.acquisition.tracking"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Suivi personnalisé", Publisher: "Fixture", MinCore: "0.0.60", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
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
	directory := c.dataDir
	c.close()
	c, err = openCatalog(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	all, err := c.requests(ctx)
	if err != nil || len(all) != 1 {
		t.Fatal(all, err)
	}
	v := all[0]
	writes := len(fixture.writes)
	if err = c.syncRequest(ctx, cfg, &v, videoKind); err != nil || v.State != "waiting" || !strings.HasPrefix(v.Message, "Surveille") || len(fixture.writes) != writes {
		t.Fatal(v, err, fixture.writes)
	}
	// A v0.0.59 component remains installable for submission, but may not silently
	// use native tracking when it lacks the operation. New packages must declare it.
	broken := bytes.ReplaceAll(code, []byte("tracking-info"), []byte("tracking-none"))
	pack.Code = broken
	if err = validateExtractionCode(ctx, pack); err == nil {
		t.Fatal("missing tracking accepted for new package")
	}
	cfg.owner = c
	session := &acquisitionSession{owner: c, descriptor: desc, code: broken}
	if reader, err := (arrClient{cfg}).trackingReader(context.WithValue(ctx, acquisitionSessionKey{}, session)); err == nil {
		reader.engine.Close()
		t.Fatal("native fallback used")
	}
}

func TestAcquisitionTrackingPagination(t *testing.T) {
	pages, total := 0, 101
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if r.Method != "GET" || r.URL.Path != "/api/v3/queue" {
			t.Error("unexpected request", r.Method, r.URL.Path)
		}
		rows := []arrObject{}
		if r.URL.Query().Get("page") == "1" {
			for i := 0; i < 100; i++ {
				rows = append(rows, arrObject{"movieId": float64(99)})
			}
		} else {
			rows = append(rows, arrObject{"movieId": float64(12), "size": float64(100), "sizeleft": float64(25), "downloadId": "private-id", "title": "private-title", "path": "private-path"})
		}
		json.NewEncoder(w).Encode(map[string]any{"records": rows, "totalRecords": total})
	}))
	defer server.Close()
	client := arrClient{arrConfig{ID: "radarr", URL: server.URL, Key: "private-key"}}
	q, _, err := client.trackingObservations(context.Background(), acquisitionRequest{Season: -1}, arrObject{"id": float64(12), "monitored": true})
	if err != nil || pages != 2 || len(q.Queue) != 101 || q.Queue[100].Remaining != 25 {
		t.Fatal(q, err, pages)
	}
	raw, _ := json.Marshal(q)
	if bytes.Contains(raw, []byte("private-")) {
		t.Fatal("private fields crossed module boundary")
	}
	total = 2001
	pages = 0
	if _, _, err = client.trackingObservations(context.Background(), acquisitionRequest{Season: -1}, arrObject{"id": float64(12)}); err == nil || pages != 1 {
		t.Fatal("queue bound ignored", err, pages)
	}
}

func TestAcquisitionTrackingLargeEnvelope(t *testing.T) {
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
	q := a.Tracking{Connector: "sonarr", RemoteID: 12, Season: -1, Monitored: true}
	for i := 1; i <= 5000; i++ {
		q.Episodes = append(q.Episodes, a.Episode{ID: i, Season: 1, Number: i})
	}
	for i := 0; i < 2000; i++ {
		q.Queue = append(q.Queue, a.QueueItem{RemoteID: 12, EpisodeID: i + 1, Size: 100, Remaining: 25})
	}
	var v acquisitionRequest
	if err = applyAcquisitionTracking(ctx, reader, q, &v); err != nil || v.Total != 5000 || v.Progress != 75 || v.State != "downloading" {
		t.Fatal(v, err)
	}
	q.Queue = append(q.Queue, a.QueueItem{})
	if err = applyAcquisitionTracking(ctx, reader, q, &v); err == nil {
		t.Fatal("unbounded queue accepted")
	}
}
