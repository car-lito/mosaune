package main

import (
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"testing"
)

func TestImportPolicy(t *testing.T) {
	q := a.ImportQuery{Scope: a.PreviewQuery{Kind: "series", Season: 1}, Candidates: []a.ImportCandidate{{Slot: 0, Season: 2, Episode: 1}, {Slot: 1, Season: 1, Episode: 3}}}
	r, err := matchImports(q)
	if err != nil || len(r.Slots) != 1 || r.Slots[0] != 1 {
		t.Fatal(r, err)
	}
	old := importsJSON
	defer func() { importsJSON = old }()
	importsJSON = []byte(`{"version":1,"mode":"deferred","limit":5000}`)
	r, err = matchImports(q)
	if err != nil || len(r.Slots) != 0 {
		t.Fatal(r, err)
	}
	p, err := importPlan(q.Scope)
	if err != nil || len(p.Reads) != 0 {
		t.Fatal(p, err)
	}
	importsJSON = []byte(`{"version":1,"mode":"verified","limit":1}`)
	q.Scope.Season = -1
	r, err = matchImports(q)
	if err != nil || len(r.Slots) != 1 {
		t.Fatal(r, err)
	}
}
