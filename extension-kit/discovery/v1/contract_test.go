package discovery

import (
	"strings"
	"testing"
)

func TestSurfaceValidation(t *testing.T) {
	valid := func() Surface {
		return Surface{Version: 1, Search: []Search{{Kind: "films", Label: "Films", Placeholder: "Search"}}, Shelves: []Shelf{{ID: "festival", Kind: "films", Label: "Festival", Selector: "kind"}}}
	}
	for _, mutate := range []func(*Surface){func(s *Surface) { s.Version = 2 }, func(s *Surface) { s.Search[0].Kind = "secret" }, func(s *Surface) { s.Search[0].Label = "bad\n" }, func(s *Surface) { s.Search[0].Placeholder = strings.Repeat("x", 201) }, func(s *Surface) { s.Search = append(s.Search, s.Search[0]) }, func(s *Surface) { s.Shelves[0].ID = "https://bad" }, func(s *Surface) { s.Shelves[0].Selector = "url" }, func(s *Surface) { s.Shelves = append(s.Shelves, s.Shelves[0]) }} {
		s := valid()
		mutate(&s)
		if s.Validate([]string{"films"}) == nil {
			t.Fatal("invalid declaration accepted", s)
		}
	}
	s := valid()
	if s.Validate([]string{"films"}) != nil {
		t.Fatal("valid declaration rejected")
	}
	if (Surface{Version: 1}).Validate(nil) != nil {
		t.Fatal("empty declaration rejected")
	}
}
