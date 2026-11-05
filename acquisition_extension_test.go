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
	"time"
)

func TestAcquisitionExtensionReplacementLifecycle(t *testing.T) {
	c, directory := testCatalog(t)
	ctx := context.Background()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Api-Key") != "fixture-only" {
			t.Error("missing host credential")
		}
		if r.Method != "PUT" || r.URL.Path != "/api/v3/movie/12" {
			t.Error("unexpected write", r.Method, r.URL.Path)
		}
		var body arrObject
		if json.NewDecoder(r.Body).Decode(&body) != nil || word(body["path"]) != "/existing/film" || number(body["qualityProfileId"]) != 87 || word(body["customField"]) != "keep" || !truth(body["monitored"]) {
			t.Error("existing remote configuration changed")
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	cfg := arrConfig{ID: "radarr", URL: server.URL, Key: "fixture-only", Profile: 2, Root: "/new", owner: c}
	client := arrClient{cfg}
	sessionContext, stop, err := client.openSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oldChecksum := sessionContext.Value(acquisitionSessionKey{}).(*acquisitionSession).checksum
	stop()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	from, to := []byte(`"searchMode":"auto"`), []byte(`"searchMode":"none"`)
	if bytes.Count(code, from) != 1 {
		t.Fatal("policy fixture not unique")
	}
	code = bytes.Replace(code, from, to, 1)
	desc.ID = "org.example.acquisition.quiet"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Surveillance seule", Publisher: "Fixture", MinCore: "0.0.59", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	raw, err := packageformat.CreateExecutable(manifest, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	installRecognitionExample(t, c, raw, pack)
	change := func(id, operation string) {
		t.Helper()
		state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = c.changeRecognition(ctx, id, operation, state.Revision); err != nil {
			t.Fatal(err)
		}
	}
	change(desc.ID, "enable")
	change(desc.ID, "select")
	object := func() arrObject {
		return arrObject{"id": float64(12), "path": "/existing/film", "qualityProfileId": float64(87), "customField": "keep"}
	}
	cfg.planChecksum = oldChecksum
	if _, err = (arrClient{cfg}).submit(ctx, object(), -1); err == nil || calls.Load() != 0 {
		t.Fatal("stale confirmation sent a write", err)
	}
	cfg.planChecksum = ""
	c.close()
	c, err = openCatalog(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	cfg.owner = c
	if id, err := (arrClient{cfg}).submit(ctx, object(), -1); err != nil || id != 12 || calls.Load() != 1 {
		t.Fatal("replacement policy not used after restart", id, err, calls.Load())
	}
	change(desc.ID, "disable")
	if _, err = (arrClient{cfg}).options(ctx); err == nil || calls.Load() != 1 {
		t.Fatal("disabled module contacted service", err)
	}
	change(desc.ID, "remove")
	if _, err = (arrClient{cfg}).submit(ctx, object(), -1); err == nil || calls.Load() != 1 {
		t.Fatal("removed module silently replaced", err)
	}
}

func TestAcquisitionExtensionDisableCancelsConnection(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	finished := make(chan error, 1)
	go func() {
		_, err := (arrClient{arrConfig{ID: "radarr", URL: server.URL, Key: "fixture", owner: c}}).options(ctx)
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("request did not start")
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(ctx, "acquisition", "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("cancelled request succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request not cancelled")
	}
}

func TestAcquisitionPlanScope(t *testing.T) {
	query := a.Query{Connector: "sonarr", RemoteID: 12, Season: 2, Seasons: []int{1, 2, 3}}
	valid := a.Plan{Write: "update", Seasons: []int{2}, MonitorEpisodes: true, Search: "season"}
	if err := valid.Validate(query); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"other-season", "all-seasons", "reset", "create", "search-all", "new-path-policy", "duplicate"} {
		plan := valid
		switch mode {
		case "other-season":
			plan.Seasons = []int{1}
		case "all-seasons":
			plan.MonitorNewItems = "all"
		case "reset":
			plan.ResetSeasons = true
		case "create":
			plan.Write = "create"
		case "search-all":
			plan.Search = "series"
		case "new-path-policy":
			plan.SeasonFolder = true
		case "duplicate":
			plan.Seasons = []int{2, 2}
		}
		if err := plan.Validate(query); err == nil {
			t.Fatal("scope escaped", mode)
		}
	}
	query.RemoteID = 0
	fresh := a.Plan{Write: "create", Seasons: []int{2}, ResetSeasons: true, MonitorNewItems: "none"}
	if err := fresh.Validate(query); err != nil {
		t.Fatal(err)
	}
	fresh.ResetSeasons = false
	if err := fresh.Validate(query); err == nil {
		t.Fatal("inherited season flags accepted")
	}
	fresh.ResetSeasons, fresh.MonitorNewItems = true, ""
	if err := fresh.Validate(query); err == nil {
		t.Fatal("inherited future monitoring accepted")
	}
}

func TestAcquisitionExtensionUpgrade(t *testing.T) {
	for _, mode := range []string{"active", "paused", "empty", "custom"} {
		t.Run(mode, func(t *testing.T) {
			c, directory := testCatalog(t)
			ctx := context.Background()
			recognitionChange(t, c, "acquisition", "remove")
			if _, err := c.db.Exec("DELETE FROM recognition_selection WHERE kind='acquisition'"); err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"films", "series"} {
				if mode == "paused" {
					recognitionChange(t, c, kind, "disable")
				}
				if mode == "empty" {
					recognitionChange(t, c, kind, "remove")
				}
				manifest, err := bundledFamily(kind)
				if err != nil {
					t.Fatal(err)
				}
				manifest.Version = "1.0.0"
				manifest.Components = manifest.Components[:len(manifest.Components)-1]
				if mode == "custom" {
					manifest.Name = "Ma famille"
				}
				raw, _ := json.Marshal(manifest)
				if _, err = c.db.Exec("UPDATE media_bundles SET manifest=? WHERE id=?", string(raw), manifest.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := c.db.Exec("PRAGMA user_version=38"); err != nil {
				t.Fatal(err)
			}
			c.close()
			c, err := openCatalog(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
			if err != nil {
				t.Fatal(err)
			}
			entry, exists := registryEntryFor(state, "mosaune.acquisition.plan")
			if exists != (mode != "empty") || entry.Enabled != (mode == "active" || mode == "custom") {
				t.Fatal(mode, exists, entry.Enabled)
			}
			var raw string
			if err = c.db.QueryRow("SELECT manifest FROM media_bundles WHERE id='org.mosaune.media.films'").Scan(&raw); err != nil {
				t.Fatal(err)
			}
			var manifest mediaBundleManifest
			if err = json.Unmarshal([]byte(raw), &manifest); err != nil {
				t.Fatal(err)
			}
			if mode == "custom" {
				if manifest.Name != "Ma famille" || manifest.Version != "1.0.0" {
					t.Fatal("custom recipe overwritten")
				}
			} else if manifest.Version != "1.1.0" {
				t.Fatal("factory recipe not upgraded")
			}
		})
	}
}

func TestRepositoryAcquisitionLifecycle(t *testing.T) {
	testRepositoryExtractionLifecycle(t, func(t *testing.T, release string) ([]byte, packageformat.Executable) {
		desc, code, err := modules.Lookup("acquisition")
		if err != nil {
			t.Fatal(err)
		}
		desc.ID, desc.Version = "org.example.acquisition.repository", release
		manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Demandes", Publisher: "Fixture", MinCore: "0.0.59", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
		raw, err := packageformat.CreateExecutable(manifest, code)
		if err != nil {
			t.Fatal(err)
		}
		pack, err := packageformat.ParseExecutable(raw, version)
		if err != nil {
			t.Fatal(err)
		}
		return raw, pack
	})
}
