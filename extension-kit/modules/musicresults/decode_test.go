package musicresults

import (
	"encoding/json"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"strings"
	"testing"
)

const releaseID = "11111111-1111-1111-1111-111111111111"
const artistID = "22222222-2222-2222-2222-222222222222"

func TestReleasePresentationAndTracks(t *testing.T) {
	for _, kind := range []string{"Album", "Single", "EP", "Other"} {
		raw := `{"id":"` + releaseID + `","title":"Album","release-group":{"primary-type":"` + kind + `"},"artist-credit":[{"artist":{"id":"` + artistID + `","name":"Artiste"}}],"media":[{"position":1,"tracks":[{"id":"track","position":2,"recording":{"title":"Chanson","length":123456}}]}]}`
		q := d.Query{Operation: "release", ID: releaseID, Page: 1, Body: json.RawMessage(raw)}
		r, e := Decode(q)
		if e != nil || r.Validate(q) != nil {
			t.Fatal(r, e)
		}
		m := r.Items[0]
		expected := kind
		if kind == "Other" {
			expected = "Sortie musicale"
		}
		if m.Presentation.Label != expected || m.Presentation.Preview.AspectRatio != 1 || m.Tracks[0].Title != "Chanson" || m.Tracks[0].Duration != 123 || m.Tracks[0].Artists[0].ID != artistID {
			t.Fatal(m)
		}
	}
}
func TestResponseBoundsAndIdentity(t *testing.T) {
	q := d.Query{Operation: "releases", Page: 2, Body: json.RawMessage(`{"count":10000,"releases":[{"id":"` + releaseID + `","title":"Album"},{"id":"` + releaseID + `","title":"Duplicate"},{"id":"bad","title":"Invalid"}]}`)}
	r, e := Decode(q)
	if e != nil || len(r.Items) != 1 || r.Pages != 50 {
		t.Fatal(r, e)
	}
	for _, raw := range []string{`{`, strings.Repeat(" ", d.MaxBody+1), `{"id":"` + artistID + `","title":"Wrong"}`} {
		q.Operation = "release"
		q.ID = releaseID
		q.Body = json.RawMessage(raw)
		if _, e := Decode(q); e == nil {
			t.Fatal("bad response accepted")
		}
	}
	q = d.Query{Operation: "artists", Page: 1, Body: json.RawMessage(`{"artists":[{"id":"` + artistID + `","name":"Artiste"}]}`)}
	r, e = Decode(q)
	if e != nil || len(r.Artists) != 1 {
		t.Fatal(r, e)
	}
	r.Artists[0].ID = "https://arbitrary.invalid"
	if r.Validate(q) == nil {
		t.Fatal("unsafe guest ID accepted")
	}
}
