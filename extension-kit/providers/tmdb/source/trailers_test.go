package main

import "testing"

func TestTrailerSafetyAndFrenchDubPreference(t *testing.T) {
	unsafe := selectTrailers([]tmdbVideo{{Site: "YouTube", Type: "Trailer", Key: "../bad"}}, "fr")
	if len(unsafe) != 0 {
		t.Fatal(unsafe)
	}
	dubbed := selectTrailers([]tmdbVideo{{Site: "YouTube", Type: "Trailer", Key: "abcdefghijk", Name: "Bande annonce VOSTFR", Language: "fr", Official: true}, {Site: "YouTube", Type: "Trailer", Key: "lmnopqrstuv", Name: "Bande annonce VF", Language: "fr"}}, "fr")
	if dubbed[0].Name != "Bande annonce VF" {
		t.Fatal("VF should precede subtitled trailers", dubbed)
	}
}
