package main

import (
	"context"
	"io"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"strings"
	"testing"
)

func TestMusicResultsLifecycleAndRestart(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	calls := 0
	p.client.Transport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"count":1,"releases":[{"id":"11111111-1111-1111-1111-111111111111","title":"Album","release-group":{"primary-type":"Album"}}]}`))}, nil
	})
	got, e := p.Discover(ctx, "", DiscoveryQuery{Kind: "music", Query: "Album", Page: 1})
	if e != nil || len(got.Items) != 1 || got.Items[0].Presentation.Label != "Album" {
		t.Fatal(got, e)
	}
	change := func(op string) {
		t.Helper()
		s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = c.changeRecognition(ctx, "music-results", op, s.Revision); e != nil {
			t.Fatal(e)
		}
	}
	call, done, e := p.openMusicResults(ctx)
	if e != nil {
		t.Fatal(e)
	}
	change("disable")
	if call.Err() == nil {
		t.Fatal("active decode lease not cancelled")
	}
	done()
	before := calls
	if _, e = p.Discover(ctx, "", DiscoveryQuery{Kind: "music", Query: "Other", Page: 1}); e == nil || calls != before {
		t.Fatal("network/native fallback while disabled", e)
	}
	surface, e := providerDiscoverySurface(ctx, p)
	if e != nil || len(surface.Search) != 0 {
		t.Fatal("disabled discovery exposed", e)
	}
	change("enable")
	if _, e = p.Discover(ctx, "", DiscoveryQuery{Kind: "music", Query: "Album", Page: 1}); e != nil {
		t.Fatal(e)
	}
	change("remove")
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("musicbrainz").(*musicBrainzProvider)
	if _, e = p.Discover(ctx, "", DiscoveryQuery{Kind: "music", Query: "Album", Page: 1}); e == nil {
		t.Fatal("removed decoder resurrected")
	}
	var cached int
	if e = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE provider='musicbrainz'").Scan(&cached); e != nil || cached == 0 {
		t.Fatal("shared cache removed", e)
	}
}

func TestMusicResultsExternalReplacement(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	desc, code, e := modules.Lookup("music-results")
	if e != nil {
		t.Fatal(e)
	}
	desc.ID = "org.example.music.decoder"
	m := packageformat.ExecutableManifest{Schema: 2, Name: "Décodeur de remplacement", Publisher: "Fixture", MinCore: "0.0.53", Module: desc, Contributions: []packageformat.Contribution{{Role: d.Capability, Target: "music"}}}
	raw, e := packageformat.CreateExecutable(m, code)
	if e != nil {
		t.Fatal(e)
	}
	pack, e := packageformat.ParseExecutable(raw, version)
	if e != nil {
		t.Fatal(e)
	}
	installRecognitionExample(t, c, raw, pack)
	change := func(id, op string) {
		t.Helper()
		s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = c.changeRecognition(ctx, id, op, s.Revision); e != nil {
			t.Fatal(e)
		}
	}
	change(desc.ID, "enable")
	change(desc.ID, "select")
	change("music-results", "remove")
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	call, done, e := p.openMusicResults(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	reader := call.Value(musicResultsKey{}).(*musicResultsReader)
	if reader.engine.id != desc.ID {
		t.Fatal("replacement not used")
	}
	r, e := p.decodeMusic(call, "releases", "", 1, []byte(`{"count":1,"releases":[{"id":"11111111-1111-1111-1111-111111111111","title":"From replacement"}]}`))
	if e != nil || len(r.Items) != 1 || r.Items[0].Title != "From replacement" {
		t.Fatal(r, e)
	}
}
func TestMusicResultsMigrationPreservesMusicConsent(t *testing.T) {
	for _, operation := range []string{"disable", "remove"} {
		t.Run(operation, func(t *testing.T) {
			c, dir := testCatalog(t)
			ctx := context.Background()
			change := func(kind, op string) {
				t.Helper()
				s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
				if e != nil {
					t.Fatal(e)
				}
				if e = c.changeRecognition(ctx, kind, op, s.Revision); e != nil {
					t.Fatal(e)
				}
			}
			change("music-results", "remove")
			change("music", operation)
			if _, e := c.db.Exec("DELETE FROM recognition_selection WHERE kind='music-results'; PRAGMA user_version=33"); e != nil {
				t.Fatal(e)
			}
			c.close()
			c, e := openCatalog(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer c.close()
			s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			entry, present := registryEntryFor(s, "mosaune.music.results")
			if present != (operation == "disable") || entry.Enabled {
				t.Fatal("migration restored unwanted capability", present, entry.Enabled)
			}
		})
	}
}
