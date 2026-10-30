package videocatalog

import (
	"strings"
	"testing"
)

func TestCatalogueContractBounds(t *testing.T) {
	q := Query{Kind: "films", Text: "Voyage", Page: 1, Language: "fr-FR"}
	r := Result{Items: []Item{{ID: "42", Kind: "films", Title: "Voyage"}}, Page: 1, Pages: 1}
	if e := r.Validate("discover", q); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*Query){func(q *Query) { q.Page = 0 }, func(q *Query) { q.Page = 51 }, func(q *Query) { q.ID = "also-seed" }, func(q *Query) { q.Shelf = "extra" }, func(q *Query) { q.Language = "" }, func(q *Query) { q.Text = strings.Repeat("x", 501) }, func(q *Query) { q.Text = "bad\x00text" }, func(q *Query) { q.Kind = "music" }} {
		bad := q
		mutate(&bad)
		if bad.Validate("discover") == nil {
			t.Fatal("query accepted", bad)
		}
	}
	for _, mutate := range []func(*Result){func(r *Result) { r.Page = 2 }, func(r *Result) { r.Pages = 51 }, func(r *Result) { r.Items = make([]Item, 21) }, func(r *Result) { r.Person = &Person{ID: "42", Name: "Extra"} }, func(r *Result) { r.Posters = []Poster{{Ref: "extra", Width: 1, Height: 1}} }, func(r *Result) { r.Items = []Item{{ID: "42", Kind: "series", Title: "Wrong kind"}} }} {
		bad := r
		mutate(&bad)
		if bad.Validate("discover", q) == nil {
			t.Fatal("result accepted", bad)
		}
	}
	if q.Validate("other") == nil {
		t.Fatal("unknown operation")
	}
	q = Query{ID: "person", Page: 3, Language: "fr"}
	r = Result{Person: &Person{ID: "person", Name: "A"}, Page: 3, Pages: 2}
	if e := r.Validate("profile", q); e != nil {
		t.Fatal("valid empty page", e)
	}
	r.Person.ID = "someone-else"
	if r.Validate("profile", q) == nil {
		t.Fatal("wrong person")
	}
}
