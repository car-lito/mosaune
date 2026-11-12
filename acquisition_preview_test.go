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

func TestAcquisitionPreviewReplacement(t *testing.T) {
	c, h, token, _, work, f := setupAcquisition(t, "films")
	ctx := context.Background()
	f.object = arrObject{"id": float64(12), "tmdbId": float64(42), "path": "/private/existing", "qualityProfileId": float64(87)}
	initial := call(t, h, "POST", "/api/v1/acquisition/preview", token, requestInput{WorkID: work, Season: -1})
	if initial.Code != 200 {
		t.Fatal(initial.Code, initial.Body.String())
	}
	var previous requestPreview
	json.Unmarshal(initial.Body.Bytes(), &previous)
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{`"reads":["status","profiles","roots","find"]`, `"reads":["status","find","roots","profiles"]`},
		{`"missing":"lookup"`, `"missing":"reject"`},
		{`"profilePrefix":"Profil #"`, `"profilePrefix":"Choix  #"`},
	} {
		if len(pair[0]) != len(pair[1]) || !bytes.Contains(code, []byte(pair[0])) {
			t.Fatal("missing replacement fixture")
		}
		code = bytes.ReplaceAll(code, []byte(pair[0]), []byte(pair[1]))
	}
	desc.ID = "org.example.acquisition.preview"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Préparation personnalisée", Publisher: "Fixture", MinCore: "0.0.67", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
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
	h, token = handlerFor(t, c, "video,music")
	stale := call(t, h, "POST", "/api/v1/acquisition/requests", token, requestInput{WorkID: work, Season: -1, Confirmation: previous.Confirmation})
	if stale.Code != 409 || len(f.writes) != 0 {
		t.Fatal("stale request wrote remotely", stale.Code, f.writes)
	}
	f.reads = nil
	response := call(t, h, "POST", "/api/v1/acquisition/preview", token, requestInput{WorkID: work, Season: -1})
	var next requestPreview
	json.Unmarshal(response.Body.Bytes(), &next)
	if response.Code != 200 || next.Root != "/private/existing" || next.Profile != "Choix  #87" || strings.Join(f.reads, ",") != "system/status,movie,rootfolder,qualityprofile" {
		t.Fatal(response.Code, next, f.reads)
	}
	f.object = nil
	f.reads = nil
	response = call(t, h, "POST", "/api/v1/acquisition/preview", token, requestInput{WorkID: work, Season: -1})
	if response.Code != 400 || !strings.Contains(response.Body.String(), "uniquement") || strings.Contains(strings.Join(f.reads, ","), "lookup") || len(f.writes) != 0 {
		t.Fatal("missing-resource policy ignored", response.Code, response.Body.String(), f.reads)
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(ctx, desc.ID, "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	f.reads = nil
	response = call(t, h, "POST", "/api/v1/acquisition/preview", token, requestInput{WorkID: work, Season: -1})
	if response.Code != 400 || len(f.reads) != 0 {
		t.Fatal("disabled preview used network", response.Code, f.reads)
	}
}

func TestAcquisitionPreviewNoFallback(t *testing.T) {
	ctx := context.Background()
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	old := bytes.ReplaceAll(code, []byte("preparation-plan"), []byte("preparation-none"))
	if bytes.Equal(old, code) {
		t.Fatal("fixture not changed")
	}
	desc.ID = "org.example.acquisition.oldpreview"
	pack := packageformat.Executable{Manifest: packageformat.ExecutableManifest{Schema: 2, Name: "Old", Publisher: "Fixture", MinCore: "0.0.67", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}, Code: old}
	if validateExtractionCode(ctx, pack) == nil {
		t.Fatal("missing preview accepted")
	}
	pack.Manifest.MinCore = "0.0.66"
	if err = validateExtractionCode(ctx, pack); err != nil {
		t.Fatal(err)
	}
	reader, err := loadFactProjector(ctx, desc, old, a.Capability)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.engine.Close()
	if _, err = readAcquisitionPreview(ctx, reader, a.PreviewQuery{Kind: "films", Season: -1}); err == nil {
		t.Fatal("native fallback")
	}
	client := arrClient{arrConfig{ID: "radarr", URL: "http://127.0.0.1:1"}}
	bad := a.PreviewPlan{Version: 1, Connector: "radarr", Reads: []string{"status", "create", "roots", "find"}, Missing: "lookup", MissingMessage: "No", SeasonMessage: "No", ProfilePrefix: "#"}
	if _, _, _, err = client.previewResource(ctx, a.PreviewQuery{Kind: "films", Season: -1}, bad, 42); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatal("unsafe operation was not rejected before transport", err)
	}
}
