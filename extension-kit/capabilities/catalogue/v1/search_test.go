package catalogue

import (
	presentation "mosaune/extensionkit/presentation/v1"
	"strings"
	"testing"
)

func TestSearchProjectionBoundary(t *testing.T) {
	valid := func() SearchIndex {
		return SearchIndex{Version: 1, Entries: []SearchEntry{{Target: "item", Title: "Title", Terms: []string{"word"}, Priority: 60, Presentation: &presentation.Result{Version: 1, Label: "Jeu"}}}}
	}
	for _, mutate := range []func(*SearchIndex){func(s *SearchIndex) { s.Version = 2 }, func(s *SearchIndex) { s.Entries[0].Target = "https://bad" }, func(s *SearchIndex) { s.Entries[0].Priority = 101 }, func(s *SearchIndex) { s.Entries[0].Terms = []string{"bad\n"} }, func(s *SearchIndex) { s.Entries[0].Title = "" }, func(s *SearchIndex) { s.Entries[0].Presentation = nil }, func(s *SearchIndex) { s.Entries[0].Presentation.Label = strings.Repeat("x", 41) }, func(s *SearchIndex) { s.Entries = append(s.Entries, s.Entries[0]) }} {
		s := valid()
		mutate(&s)
		if s.Validate() == nil {
			t.Fatal("untrusted candidate accepted", s)
		}
	}
	s := valid()
	if s.Validate() != nil {
		t.Fatal("valid candidate rejected")
	}
	empty := SearchIndex{Version: 1}
	if empty.Validate() != nil {
		t.Fatal("empty deliberate index rejected")
	}
	q := Query{Items: []Facts{{ID: "one", Name: "one"}}}
	r := Result{Items: []Item{{Title: "one", Search: &s}}}
	if r.Validate(q) != nil {
		t.Fatal("projection rejected")
	}
}
