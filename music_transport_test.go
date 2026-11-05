package main

import (
	"bytes"
	"context"
	"encoding/json"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	defaults "mosaune/extensionkit/providers/musicbrainz"
	"mosaune/extensionkit/registry"
	"net/http"
	"net/url"
	"testing"
)

func TestMusicTransportPermissions(t *testing.T) {
	for _, mode := range []string{"valid", "host", "port", "rate", "bytes", "format", "timeout", "missing", "image-path", "credential"} {
		t.Run(mode, func(t *testing.T) {
			profile := defaults.Transport()
			source := &profile.Sources[0]
			switch mode {
			case "host":
				source.BaseURL = "https://example.org/ws/2"
			case "port":
				source.BaseURL = "https://musicbrainz.org:443/ws/2"
			case "rate":
				source.IntervalMS = 0
			case "bytes":
				source.MaxBytes = 8 << 20
			case "format":
				source.Format = "image"
			case "timeout":
				source.TimeoutMS = 20000
			case "missing":
				profile.Sources = profile.Sources[:5]
			case "image-path":
				profile.Sources[4].BaseURL += "/changed"
			case "credential":
				source.Credential = "password"
			}
			if err := validateMusicTransport(profile); (err == nil) != (mode == "valid") {
				t.Fatal(mode, err)
			}
		})
	}
}

func TestMusicTransportInstalledReplacement(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	desc, code, err := modules.Lookup("music-results")
	if err != nil {
		t.Fatal(err)
	}
	// Equal-length edits preserve the Wasm data segment and change only module
	// declarations. The server must observe them through the installed module.
	for _, change := range [][2]string{{"musicbrainz.org/ws/2", "musicbrainz.org/ws/3"}, {`"maxBytes":2097152`, `"maxBytes":1048576`}} {
		if bytes.Count(code, []byte(change[0])) != 1 {
			t.Fatal("fixture declaration not unique", change[0])
		}
		code = bytes.Replace(code, []byte(change[0]), []byte(change[1]), 1)
	}
	desc.ID = "org.example.music.transport"
	manifest := packageformat.ExecutableManifest{
		Schema: 2, Name: "Sources musicales", Publisher: "Fixture", MinCore: "0.0.58", Module: desc,
		Contributions: []packageformat.Contribution{{Role: d.Capability, Target: "music"}},
	}
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
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	call, done, err := p.openMusicResults(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	reader := call.Value(musicResultsKey{}).(*musicResultsReader)
	if reader.engine.id != desc.ID {
		t.Fatal("replacement not selected")
	}
	source, err := p.musicCall(call, "musicbrainz", "/release", url.Values{"fmt": {"json"}})
	if err != nil || source.MaxBytes != 1048576 {
		t.Fatal(source.MaxBytes, err)
	}
	calls := 0
	p.client.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/ws/3/release" {
			t.Fatal("installed declaration ignored", req.URL)
		}
		return visualResponse(200, string(bytes.Repeat([]byte(" "), 1048577))), nil
	})
	var result json.RawMessage
	if err = p.get(call, "/release", nil, &result); err == nil || calls != 1 {
		t.Fatal("module response limit ignored", err, calls)
	}
	var cached int
	if err = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE provider='musicbrainz'").Scan(&cached); err != nil || cached != 0 {
		t.Fatal("oversized response cached", err, cached)
	}
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/a/ab/A%20portrait.jpg"
	imageCall, err := p.musicCall(call, "image", imageURL, nil)
	if err != nil || imageCall.URL != imageURL {
		t.Fatal("escaped image URL changed", err)
	}
	if _, err = p.musicCall(call, "image", "https://example.org/image.jpg", nil); err == nil {
		t.Fatal("undeclared image origin accepted")
	}
	// A module declaring a forbidden origin is rejected at import, before any
	// call to its requested network destination.
	badCode := bytes.Replace(code, []byte("musicbrainz.org/ws/3"), []byte("examplebad.org/ws/3"), 1)
	badPack := pack
	badPack.Code = badCode
	if err = validateExtractionCode(ctx, badPack); err == nil {
		t.Fatal("invalid transport imported")
	}
}

func TestMusicTransportFactoryUpgrade(t *testing.T) {
	for _, mode := range []string{"enabled", "disabled", "removed", "modified", "imported"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			store := registry.SQLStore{DB: c.db}
			s, e := store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			entry, _ := registryEntryFor(s, "mosaune.music.results")
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
				if e = c.changeRecognition(ctx, "music-results", op, plan.After.Revision); e != nil {
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
			if _, e = c.db.Exec("PRAGMA user_version=37"); e != nil {
				t.Fatal(e)
			}
			if e = migrateMusicTransport(c.db); e != nil {
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
			want := "0.4.0"
			if mode == "modified" || mode == "imported" {
				want = "0.3.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
