package main

import (
	"context"
	"encoding/json"
	video "mosaune/extensionkit/capabilities/video/v1"
	presentation "mosaune/extensionkit/presentation/v1"
	"testing"
)

func TestMusicReleasePresentation(t *testing.T) {
	for _, tc := range []struct{ kind, label string }{{"Album", "Album"}, {"Single", "Single"}, {"EP", "EP"}, {"", "Sortie musicale"}, {"Other", "Sortie musicale"}} {
		var release mbRelease
		raw := `{"id":"test","title":"Record","release-group":{"primary-type":"` + tc.kind + `"}}`
		if e := json.Unmarshal([]byte(raw), &release); e != nil {
			t.Fatal(e)
		}
		m := mbMetadata(release)
		if m.Presentation.Label != tc.label || m.Presentation.Preview.AspectRatio != 1 {
			t.Fatal(m.Presentation)
		}
	}
}
func TestSearchPresentationPersistsAndValidatesAtGuestBoundary(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	p := &presentation.Result{Version: 1, Label: "Single", Keywords: []string{"Acoustique"}, Preview: &presentation.Preview{AspectRatio: 1, Shape: "rectangle", Fit: "cover"}}
	id, e := c.rememberWork(ctx, "music", Metadata{Title: "Record", Provider: "sample", ProviderID: "42", Presentation: p}, true)
	if e != nil {
		t.Fatal(e)
	}
	w, e := c.work(ctx, id)
	if e != nil || w.Metadata.Presentation.Label != "Single" {
		t.Fatal(w, e)
	}
	// A light search can add the new display fields to an older detailed record
	// without replacing its title or synopsis.
	if _, e = c.db.Exec(`UPDATE works SET data=json_remove(json_set(data,'$.overview','Keep me'),'$.presentation') WHERE id=?`, id); e != nil {
		t.Fatal(e)
	}
	if _, e = c.rememberWork(ctx, "music", Metadata{Title: "Light title", Provider: "sample", ProviderID: "42", Presentation: p}, false); e != nil {
		t.Fatal(e)
	}
	w, e = c.work(ctx, id)
	if e != nil || w.Metadata.Title != "Record" || w.Metadata.Overview != "Keep me" || w.Metadata.Presentation.Label != "Single" {
		t.Fatal("existing record lost", w, e)
	}
	p.Version = 999
	if _, e = c.rememberWork(ctx, "music", Metadata{Title: "Bad", Provider: "sample", ProviderID: "43", Presentation: p}, true); e == nil {
		t.Fatal("unvalidated presentation persisted")
	}
	raw := json.RawMessage(`{"title":"Film","presentation":{"version":1,"label":"Film","keywords":["Animation"],"preview":{"aspectRatio":0.6666667,"shape":"rectangle","fit":"cover"}}}`)
	md, e := tmdbExtensionMetadata(raw, video.Selection{Kind: "films", ID: "42"})
	if e != nil || md.Presentation.Label != "Film" {
		t.Fatal(md, e)
	}
	bad := json.RawMessage(`{"title":"Film","presentation":{"version":1,"label":"Film","preview":{"aspectRatio":99,"shape":"rectangle","fit":"cover"}}}`)
	if _, e = tmdbExtensionMetadata(bad, video.Selection{Kind: "films", ID: "42"}); e == nil {
		t.Fatal("guest geometry bypass")
	}
}
