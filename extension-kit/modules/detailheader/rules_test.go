package detailheader

import (
	d "mosaune/extensionkit/capabilities/detailheader/v1"
	"strings"
	"testing"
)

func TestHeaders(t *testing.T) {
	cases := []struct {
		q            d.Query
		title, label string
	}{
		{d.Query{Kind: "films", Title: "Film", Year: 2020, Runtime: 123, Genres: []string{"Action"}, Rating: 8, VoteCount: 2, Provider: "fixture"}, "Film", "FILM"},
		{d.Query{Kind: "series", Title: "Episode", Series: "Show"}, "Show", "SÉRIE"},
		{d.Query{Kind: "series", StoredWork: true, Title: "Show", Series: "Other"}, "Show", "SÉRIE"},
		{d.Query{Kind: "music", Title: "Song", Album: "Album"}, "Album", "ALBUM"},
		{d.Query{Kind: "music", Track: true, Title: "Song", Album: "Album", GroupTitle: "Album"}, "Song", "MORCEAU"},
		{d.Query{Kind: "music", Title: "Song", GroupTitle: "Custom album", Album: "Album"}, "Custom album", "ALBUM"},
		{d.Query{Kind: "games", Title: "Game"}, "Game", "games"},
		{d.Query{Kind: "films", Name: "Name"}, "Name", "FILM"},
	}
	for _, c := range cases {
		r := Project(c.q)
		if r.Validate(c.q) != nil || r.Title != c.title || !strings.HasPrefix(r.Eyebrow, c.label) {
			t.Fatal(r, c)
		}
	}
	q := d.Query{Kind: "music", Title: "Album", LocalCount: 2, Edition: "Deluxe"}
	r := Project(q)
	if r.Footnote != "Édition : Deluxe" || !strings.Contains(r.Eyebrow, "COLLECTION") {
		t.Fatal(r)
	}
	r = Project(cases[0].q)
	if len(r.Facts) != 4 || r.Facts[3].Note != "fixture · 2 votes" {
		t.Fatal("wrong provider attribution", r)
	}
}
