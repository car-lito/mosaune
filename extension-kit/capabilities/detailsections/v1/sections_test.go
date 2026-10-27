package detailsections

import (
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	q := Query{Kind: "films", People: []Person{{Name: "Someone"}}, Trailers: []Trailer{{Name: "Trailer"}}}
	good := Result{Blocks: []Block{{Key: "people", Region: "full", Kind: "people", References: []Reference{{Index: 0}}}}}
	if good.Validate(q) != nil {
		t.Fatal("valid references rejected")
	}
	for _, r := range []Result{{}, {Blocks: []Block{{Key: "x", Region: "full", Kind: "script"}}}, {Blocks: []Block{{Key: "x", Region: "main", Kind: "people", References: []Reference{{Index: 1}}}}}, {Blocks: []Block{{Key: "x", Region: "aside", Kind: "trailers", References: []Reference{{Index: -1}}}}}, {Blocks: []Block{{Key: "x", Region: "main", Kind: "people", References: []Reference{{Index: 0}, {Index: 0}}}}}, {Blocks: []Block{{Key: "x", Region: "unknown", Kind: "paragraph"}}}, {Blocks: []Block{{Key: "x", Region: "full", Kind: "paragraph"}, {Key: "x", Region: "full", Kind: "paragraph"}}}, {Blocks: []Block{{Key: "x", Region: "full", Kind: "facts", Fields: []Field{{Label: "", Value: "v"}}}}}, {Blocks: []Block{{Key: "x", Region: "full", Kind: "paragraph", Text: strings.Repeat("x", 16001)}}}} {
		if r.Validate(q) == nil {
			t.Fatalf("invalid result accepted: %+v", r)
		}
	}
	if (Result{Blocks: []Block{}}).Validate(q) != nil {
		t.Fatal("intentional empty presentation rejected")
	}
	q.Overview = strings.Repeat("x", 16001)
	if q.Validate() == nil {
		t.Fatal("unbounded input")
	}
	q.Overview = "one\ntwo"
	if q.Validate() != nil {
		t.Fatal("multiline summary rejected")
	}
	q.Overview = "bad\x00"
	if q.Validate() == nil {
		t.Fatal("control accepted")
	}
}
