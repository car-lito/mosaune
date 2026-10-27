package detailheader

import (
	"math"
	"mosaune/extensionkit/protocol"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	q := Query{Kind: "films", Title: "Film"}
	good := Result{Title: "Film", Facts: []Fact{{Key: "a", Text: "2020", Note: "Note"}}}
	if good.Validate(q) != nil {
		t.Fatal("valid result refused")
	}
	for _, r := range []Result{{}, {Title: " ", Facts: []Fact{}}, {Title: "Film"}, {Title: "Film", Facts: []Fact{{Key: "x", Text: "a"}, {Key: "x", Text: "b"}}}, {Title: "Film", Facts: []Fact{{Key: "x", Text: ""}}}, {Title: "Film", Facts: []Fact{{Key: "x", Text: "bad\x00"}}}, {Title: strings.Repeat("x", 2049), Facts: []Fact{}}, {Title: "Film", Facts: make([]Fact, 17)}} {
		if r.Validate(q) == nil {
			t.Fatalf("invalid reply accepted %+v", r)
		}
	}
	if (Result{Title: "Film", Facts: []Fact{}}).Validate(q) != nil {
		t.Fatal("minimal header refused")
	}
	for _, bad := range []Query{{Kind: ""}, {Kind: "films", Rating: math.NaN()}, {Kind: "films", Rating: 11}, {Kind: "films", Year: -1}, {Kind: "films", Title: strings.Repeat("x", 2049)}, {Kind: "films", Genres: make([]string, 33)}} {
		if bad.Validate() == nil {
			t.Fatal("invalid facts accepted", bad)
		}
	}
	var r Result
	if protocol.DecodeOne([]byte(`{"title":"Film","facts":[],"html":"<script>"}`), &r) == nil {
		t.Fatal("arbitrary UI code accepted")
	}
}
