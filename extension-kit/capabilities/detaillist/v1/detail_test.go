package detaillist

import (
	"strings"
	"testing"
)

func TestValidateReferencesAndLimits(t *testing.T) {
	q := Query{Entries: []Entry{{ID: "visible", Title: "Episode", Season: 1, Episode: 2}}, Seasons: []Season{{Number: 1, Name: "Season", Episodes: 10}}}
	good := func() Result {
		return Result{Files: "Files", Empty: "Empty", Groups: []Group{}, Rows: []Row{{ID: "visible", Title: "Episode"}}}
	}
	if good().Validate(q) != nil {
		t.Fatal("valid result refused")
	}
	for _, change := range []func(*Result){func(r *Result) { r.Groups = nil }, func(r *Result) { r.Rows[0].ID = "hidden" }, func(r *Result) { r.Rows = nil }, func(r *Result) { r.Rows = append(r.Rows, r.Rows[0]) }, func(r *Result) { r.Rows[0].Group = "unknown" }, func(r *Result) { r.Rows[0].Title = "bad\ntext" }, func(r *Result) { r.Files = "" }, func(r *Result) { n := 7; r.Groups = []Group{{Key: "a", Title: "a", Artwork: &n}} }, func(r *Result) { n := 99; r.Groups = []Group{{Key: "a", Title: "a", RequestSeason: &n}} }} {
		r := good()
		change(&r)
		if r.Validate(q) == nil {
			t.Fatal("unsafe reply accepted", r)
		}
	}
	q.Entries = append(q.Entries, q.Entries[0])
	if q.Validate() == nil {
		t.Fatal("duplicate IDs accepted")
	}
	q.Entries = q.Entries[:1]
	q.Entries[0].Title = strings.Repeat("x", 1025)
	if q.Validate() == nil {
		t.Fatal("oversized text")
	}
}
