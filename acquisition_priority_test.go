package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type arrivalProvider struct{ sampleProvider }

func (p *arrivalProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "tmdb", Kinds: []string{"films", "series"}}
}
func (p *arrivalProvider) Details(_ context.Context, _ string, s MetadataSelection) (Metadata, error) {
	return Metadata{Provider: "tmdb", ProviderID: s.ID, Title: "Test", Season: s.Season, Episode: s.Episode, MatchState: "matched"}, nil
}
func TestAcquisitionImportedFileBackgroundAvailability(t *testing.T) {
	for _, kind := range []string{"films", "series"} {
		t.Run(kind, func(t *testing.T) {
			c, h, token, cfg, work, f := setupAcquisition(t, kind)
			ctx := testContext(context.Background())
			season := 1
			if kind == "films" {
				season = -1
			}
			if r := confirmAcquisition(t, h, token, work, season); r.Code != 201 {
				t.Fatal(r.Body.String())
			}
			// Replace only the metadata network provider after the TVDB resolution fixture.
			p := &arrivalProvider{}
			c.providers = []MetadataProvider{p}
			l, e := c.requestLibrary(ctx, cfg)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.Mkdir(filepath.Join(l.Path, "Arrival"), 0700); e != nil {
				t.Fatal(e)
			}
			putFile(t, l.Path, "Arrival/opaque.mp4", "readable media fixture")
			f.object["path"] = "/new/Arrival"
			if kind == "films" {
				f.object["hasFile"] = true
				f.object["movieFile"] = map[string]any{"relativePath": "opaque.mp4"}
			} else {
				f.episodes = []arrObject{{"id": float64(1), "hasFile": true, "episodeFileId": float64(9), "seasonNumber": float64(1), "episodeNumber": float64(2)}}
				f.episodeFiles = []arrObject{{"id": float64(9), "relativePath": "opaque.mp4"}}
			}
			requests, e := c.requests(ctx)
			if e != nil || len(requests) != 1 {
				t.Fatal(requests, e)
			}
			v := requests[0]
			if e = c.syncRequest(ctx, cfg, &v, allKinds); e != nil || v.State != "imported" || v.Local != 0 {
				t.Fatal(v, e)
			}
			c.metadataStep(ctx, allKinds)
			if e = c.syncRequest(ctx, cfg, &v, allKinds); e != nil || v.State != "available" || v.Local != 1 {
				t.Fatal(v, e)
			}
			if p.searches != 0 {
				t.Fatal("trusted imported identity replaced by a name search")
			}
			m := getItems(t, c)[0]
			if kind == "series" && (m.Metadata.Season != 1 || m.Metadata.Episode != 2) {
				t.Fatal(m.Metadata)
			}
			if e = c.saveRequest(ctx, v); e != nil {
				t.Fatal(e)
			}
			r := call(t, h, "GET", "/api/v1/media/"+m.ID+"/stream", "", nil)
			if r.Code != 200 || r.Body.String() != "readable media fixture" {
				t.Fatal(r.Code, r.Body.String())
			}
			if len(f.writes) != 1 {
				t.Fatal("background tracking wrote to service", f.writes)
			}
		})
	}
}
