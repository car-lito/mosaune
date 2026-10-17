package v1

import "testing"

func TestMetadataBoundary(t *testing.T) {
	c := Candidate{ID: "opaque:other-provider:1", Title: "An album", Year: 2025}
	if c.Validate() != nil {
		t.Fatal("opaque id refused")
	}
	if (SearchResult{Items: []Candidate{}}).Validate() != nil {
		t.Fatal("empty search refused")
	}
	for _, r := range []SearchResult{{}, {Items: []Candidate{{ID: "", Title: "x"}}}, {Items: []Candidate{c, c}}, {Items: make([]Candidate, 51)}} {
		if r.Validate() == nil {
			t.Fatal("malformed result accepted")
		}
	}
	if (Query{Kind: "future.books", Title: "Book"}).Validate() != nil {
		t.Fatal("future media kind artificially restricted")
	}
}
