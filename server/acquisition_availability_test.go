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

func TestAcquisitionAvailabilityReplacement(t *testing.T) {
	c, h, token, cfg, work, f := setupAcquisition(t, "films")
	ctx := testContext(context.Background())
	if resp := confirmAcquisition(t, h, token, work, -1); resp.Code != 201 {
		t.Fatal(resp.Body.String())
	}
	lib, err := c.requestLibrary(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	putFile(t, lib.Path, "opaque.mp4", "fixture")
	if _, err = c.scan(ctx, lib, false); err != nil {
		t.Fatal(err)
	}
	w, err := c.work(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	if len(items) != 1 {
		t.Fatal(items)
	}
	if err = c.saveMetadata(ctx, items[0], w.Metadata); err != nil {
		t.Fatal(err)
	}
	f.object["hasFile"] = true
	f.object["path"] = cfg.Root
	f.object["movieFile"] = map[string]any{"relativePath": "opaque.mp4"}
	requests, err := c.requests(ctx)
	if err != nil || len(requests) != 1 {
		t.Fatal(err)
	}
	v := requests[0]
	if err = c.syncRequest(ctx, cfg, &v, allKinds); err != nil || v.State != "available" || v.Local != 1 {
		t.Fatal(v, err)
	}
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	before, after := []byte(`{"mode":"verified"}`), []byte(`{"mode":"deferred"}`)
	if !bytes.Contains(code, before) {
		t.Fatal("missing policy fixture")
	}
	code = bytes.ReplaceAll(code, before, after)
	desc.ID = "org.example.acquisition.availability"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Disponibilité personnalisée", Publisher: "Fixture", MinCore: "0.0.69", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
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
		s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = c.changeRecognition(ctx, desc.ID, op, s.Revision); e != nil {
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
	if err = c.syncRequest(ctx, cfg, &v, allKinds); err != nil || v.Local != 0 || v.State != "imported" {
		t.Fatal("replacement ignored", v, err)
	}
	state, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.changeRecognition(ctx, desc.ID, "disable", state.Revision); err != nil {
		t.Fatal(err)
	}
	f.reads = nil
	if err = c.syncRequest(ctx, cfg, &v, allKinds); err == nil || len(f.reads) != 0 {
		t.Fatal("disabled fallback", err, f.reads)
	}
	if len(f.writes) != 1 {
		t.Fatal("tracking wrote remotely", f.writes)
	}
}
func TestAcquisitionAvailabilityMissingProtocol(t *testing.T) {
	desc, code, err := modules.Lookup("acquisition")
	if err != nil {
		t.Fatal(err)
	}
	old, next := []byte("local-availability"), []byte("local-unavailable!")
	if len(old) != len(next) || !bytes.Contains(code, old) {
		t.Fatal("fixture")
	}
	code = bytes.ReplaceAll(code, old, next)
	reader, err := loadFactProjector(context.Background(), desc, code, a.Capability)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.engine.Close()
	if _, err = readAcquisitionAvailability(context.Background(), reader, a.AvailabilityQuery{Scope: a.PreviewQuery{Kind: "films", Season: -1}}); err == nil {
		t.Fatal("missing protocol fell back")
	}
	desc.ID = "org.example.acquisition.oldavailability"
	manifest := packageformat.ExecutableManifest{Schema: 2, Name: "Old", Publisher: "Fixture", MinCore: "0.0.69", Module: desc, Contributions: []packageformat.Contribution{{Role: a.Capability, Target: "video"}}}
	raw, err := packageformat.CreateExecutable(manifest, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateExtractionCode(context.Background(), pack); err == nil {
		t.Fatal("missing protocol accepted for new package")
	}
}
