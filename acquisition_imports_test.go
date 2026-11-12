package main

import (
	"bytes"
	"context"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"testing"
)

func TestAcquisitionImportReplacement(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "opaque.mp4", "fixture")
	lib, err := c.scan(ctx, library{Path: dir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	work, err := c.rememberWork(ctx, "films", Metadata{Provider: "sample", ProviderID: "42", Title: "Fixture"}, true)
	if err != nil {
		t.Fatal(err)
	}
	cfg := arrConfig{ID: "radarr", Library: lib.ID, Root: "/movies"}
	v := acquisitionRequest{Kind: "films", Season: -1, Library: lib.ID, Root: "/movies", WorkID: work}
	obj := arrObject{"path": "/movies", "movieFile": map[string]any{"relativePath": "opaque.mp4"}}
	count := func() int {
		var n int
		if e := c.db.QueryRow("SELECT count(*) FROM metadata_priorities").Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, nil); err != nil || count() != 1 {
		t.Fatal("factory association", err, count())
	}
	if _, err = c.db.Exec("DELETE FROM metadata_priorities"); err != nil {
		t.Fatal(err)
	}
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	before, after := []byte(`"mode":"verified"`), []byte(`"mode":"deferred"`)
	if !bytes.Contains(code, before) {
		t.Fatal("missing fixture")
	}
	code = bytes.ReplaceAll(code, before, after)
	desc.ID = "org.example.acquisition.imports"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Imports conservateurs", Publisher: "Fixture", MinCore: "0.0.68", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
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
		state, e := (registry.SQLStore{DB: c.db}).Load(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = c.changeRecognition(ctx, desc.ID, op, state.Revision); e != nil {
			t.Fatal(e)
		}
	}
	data := c.dataDir
	c.close()
	c, err = openCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, nil); err != nil || count() != 0 {
		t.Fatal("replacement ignored after restart", err, count())
	}
	// A deferred series plan must not attempt episode-file HTTP requests.
	v.Kind = "series"
	v.Season = 1
	cfg.ID = "sonarr"
	cfg.URL = "http://127.0.0.1:1"
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, nil); err != nil {
		t.Fatal("deferred requested network", err)
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(ctx, desc.ID, "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, nil); err == nil {
		t.Fatal("disabled component fell back")
	}
}
func TestAcquisitionImportMissingProtocol(t *testing.T) {
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	code = bytes.ReplaceAll(code, []byte("match-imports"), []byte("match-missing"))
	reader, err := loadFactProjector(ctx, desc, code, a.Capability)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.engine.Close()
	var out a.ImportResult
	if err = reader.call(ctx, "match-imports", a.ImportQuery{Scope: a.PreviewQuery{Kind: "films", Season: -1}, Candidates: []a.ImportCandidate{}}, &out); err == nil {
		t.Fatal("missing matching fell back")
	}
}

func TestAcquisitionImportAmbiguousPaths(t *testing.T) {
	c, _, _, cfg, work, remote := setupAcquisition(t, "series")
	ctx := testContext(context.Background())
	lib, err := c.requestLibrary(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	putFile(t, lib.Path, "opaque.mp4", "fixture")
	if _, err = c.scan(ctx, lib, false); err != nil {
		t.Fatal(err)
	}
	v := acquisitionRequest{Kind: "series", Season: 1, Library: lib.ID, Root: cfg.Root, WorkID: work, RemoteID: 12}
	obj := arrObject{"path": cfg.Root}
	episodes := []arrObject{
		{"hasFile": true, "episodeFileId": float64(1), "seasonNumber": float64(1), "episodeNumber": float64(1)},
		{"hasFile": true, "episodeFileId": float64(2), "seasonNumber": float64(1), "episodeNumber": float64(2)},
	}
	remote.episodeFiles = []arrObject{{"id": float64(1), "relativePath": "opaque.mp4"}, {"id": float64(2), "relativePath": "./opaque.mp4"}}
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, episodes); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = c.db.QueryRow("SELECT count(*) FROM metadata_priorities").Scan(&n); err != nil || n != 0 {
		t.Fatal("ambiguous path associated", n, err)
	}
	remote.episodeFiles = remote.episodeFiles[:1]
	if err = c.prioritizeImport(ctx, arrClient{cfg}, v, obj, episodes); err != nil {
		t.Fatal(err)
	}
	if err = c.db.QueryRow("SELECT count(*) FROM metadata_priorities").Scan(&n); err != nil || n != 1 {
		t.Fatal("unique path not associated", n, err)
	}
}
