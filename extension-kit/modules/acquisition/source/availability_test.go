package main

import (
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"testing"
)

func TestLocalAvailabilityPolicy(t *testing.T) {
	q := a.AvailabilityQuery{Scope: a.PreviewQuery{Kind: "films", Season: -1}, Film: true}
	r, err := localAvailability(q)
	if err != nil || r.Count() != 1 {
		t.Fatal(r, err)
	}
	old := availabilityJSON
	defer func() { availabilityJSON = old }()
	availabilityJSON = []byte(`{"mode":"deferred"}`)
	r, err = localAvailability(q)
	if err != nil || r.Count() != 0 {
		t.Fatal(r, err)
	}
	availabilityJSON = []byte(`{"mode":"invented"}`)
	if _, err = localAvailability(q); err == nil {
		t.Fatal("invalid policy")
	}
}
