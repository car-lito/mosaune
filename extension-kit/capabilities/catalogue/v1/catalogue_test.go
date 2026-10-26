package catalogue

import (
	"strings"
	"testing"
)

func TestUntrustedProjection(t *testing.T) {
	q := Query{Items: []Facts{{ID: "one", Name: "Test"}}}
	r := Result{Items: []Item{{Title: "Test"}}}
	if r.Validate(q) != nil {
		t.Fatal("valid denied")
	}
	for _, bad := range []Result{{}, {Items: []Item{{Title: "Test"}, {Title: "Extra"}}}, {Items: []Item{{Title: ""}}}, {Items: []Item{{Title: "Test", Key: "bad\n"}}}, {Items: []Item{{Title: "Test", Order: [2]int{-1, 0}}}}} {
		if bad.Validate(q) == nil {
			t.Fatal("bad reply accepted", bad)
		}
	}
	q.Items[0].Title = strings.Repeat("<", 2048)
	q.Items = append(q.Items, q.Items[0], q.Items[0], q.Items[0], q.Items[0])
	if q.Validate() == nil {
		t.Fatal("oversized escaped input accepted")
	}
}
