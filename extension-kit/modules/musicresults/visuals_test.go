package musicresults

import (
	"encoding/json"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"strings"
	"testing"
)

func TestVisualCoverFallback(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	group := "22222222-2222-2222-2222-222222222222"
	for _, ref := range []string{id, id + "/" + group} {
		q := d.VisualQuery{Operation: "artwork", Ref: ref}
		first, e := Visual(q)
		if e != nil || first.Fetch == nil || first.Fetch.Path != "/release/"+id+"/front-500" {
			t.Fatal(first, e)
		}
		q.State = first.State
		q.Status = 404
		next, e := Visual(q)
		if e != nil {
			t.Fatal(e)
		}
		if ref == id {
			if next.Fetch.Source != "musicbrainz" {
				t.Fatal("group lookup absent", next)
			}
			q.State = next.State
			q.Status = 200
			q.Body = json.RawMessage(`{"release-group":{"id":"` + group + `"}}`)
			next, e = Visual(q)
		}
		if e != nil || next.Fetch == nil || next.Fetch.Path != "/release-group/"+group+"/front-500" {
			t.Fatal(next, e)
		}
		q.State = next.State
		q.Status = 404
		q.Body = nil
		last, e := Visual(q)
		if e != nil || !last.Missing {
			t.Fatal("unbounded fallback", last, e)
		}
	}
}

func TestVisualPortraitWorkflow(t *testing.T) {
	q := d.VisualQuery{Operation: "artist", Ref: "11111111-1111-1111-1111-111111111111"}
	replies := []string{
		`{"relations":[{"type":"wikidata","url":{"resource":"https://evil.test/wiki/Q1"}},{"type":"wikidata","url":{"resource":"https://www.wikidata.org/wiki/Q42"}}]}`,
		`{"claims":{"P18":[{"rank":"deprecated","mainsnak":{"datavalue":{"value":"Old.jpg"}}},{"rank":"normal","mainsnak":{"datavalue":{"value":"Photo.jpg"}}}]}}`,
		`{"query":{"pages":{"1":{"imageinfo":[{"thumburl":"https://evil.test/image.jpg"},{"thumburl":"https://upload.wikimedia.org/image.jpg","extmetadata":{"Artist":{"value":"<a>Photo &amp; Co</a>"},"LicenseShortName":{"value":"CC BY-SA"}}}]}}}}`,
	}
	for i, source := range []string{"musicbrainz", "wikidata", "commons"} {
		step, e := Visual(q)
		if e != nil || step.Fetch == nil || step.Fetch.Source != source {
			t.Fatal(step, e)
		}
		q.State = step.State
		q.Status = 200
		q.Body = json.RawMessage(replies[i])
	}
	step, e := Visual(q)
	if e != nil || step.Photo == nil || step.Photo.File != "Photo.jpg" || step.Photo.Credit != "Photo & Co" || step.Photo.License != "CC BY-SA" {
		t.Fatal(step, e)
	}
}

func TestVisualBoundaries(t *testing.T) {
	for _, q := range []d.VisualQuery{
		{Operation: "artwork", Ref: "https://localhost/x"},
		{Operation: "artwork", Ref: "commons:../private.jpg"},
		{Operation: "artist", Ref: "11111111-1111-1111-1111-111111111111", Body: json.RawMessage(strings.Repeat(" ", d.MaxBody+1))},
	} {
		if _, e := Visual(q); e == nil {
			t.Fatal("invalid query accepted")
		}
	}
	for _, f := range []d.VisualFetch{
		{Source: "image", Path: "https://upload.wikimedia.org.evil.test/x", Image: true},
		{Source: "image", Path: "http://upload.wikimedia.org/x", Image: true},
		{Source: "image", Path: "https://user@upload.wikimedia.org/x", Image: true},
		{Source: "musicbrainz", Path: "evil/artist/11111111-1111-1111-1111-111111111111", Params: map[string]string{"inc": "url-rels"}},
		{Source: "coverart", Path: "/release/11111111-1111-1111-1111-111111111111/front-500?redirect=evil", Image: true},
		{Source: "wikidata", Path: "/w/api.php", Params: map[string]string{"action": "edit", "format": "json", "entity": "Q42", "property": "P18"}},
	} {
		if f.Validate() == nil {
			t.Fatal("unsafe fetch accepted", f)
		}
	}
	q := d.VisualQuery{Operation: "artwork", Ref: "11111111-1111-1111-1111-111111111111"}
	if (d.VisualStep{Missing: true, Fetch: &d.VisualFetch{Source: "image", Path: "https://upload.wikimedia.org/x", Image: true}}).Validate(q) == nil {
		t.Fatal("ambiguous result accepted")
	}
}
