package detailsections

import (
	d "mosaune/extensionkit/capabilities/detailsections/v1"
	"strings"
	"testing"
)

func TestSections(t *testing.T) {
	for _, kind := range []string{"films", "series"} {
		q := d.Query{Kind: kind, Overview: "A story", ReleaseDate: "2020-01-01", LocalCount: 2, People: []d.Person{{Name: "Person"}}, Trailers: []d.Trailer{{Name: "Trailer", Language: "fr"}, {Name: "Other", Language: "en"}}}
		r := Project(q)
		if e := r.Validate(q); e != nil {
			t.Fatal(e)
		}
		if len(r.Blocks) != 4 || r.Blocks[0].Text != "A story" || r.Blocks[1].References[0].Label != "Français" || r.Blocks[2].Fields[1].Value != "2 fichier(s) disponible(s)" {
			t.Fatal(r)
		}
	}
	q := d.Query{Kind: "films", Cast: []string{"First", "Second"}}
	r := Project(q)
	if r.Validate(q) != nil || !strings.Contains(r.Blocks[0].Text, "pas encore") || r.Blocks[2].Text != "First · Second" {
		t.Fatal(r)
	}
	if r := Project(d.Query{Kind: "music"}); r.Blocks == nil || len(r.Blocks) != 0 {
		t.Fatal(r)
	}
}
