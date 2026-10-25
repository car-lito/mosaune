package audiotags

import (
	"strings"
	"testing"
)

func TestResultValidation(t *testing.T) {
	for _, r := range []Result{{Title: "hidden"}, {Matched: true, Title: strings.Repeat("x", 501)}, {Matched: true, Title: string([]byte{0xff})}, {Matched: true, Year: 10000}, {Matched: true, Track: -1}, {Matched: true, Picture: make([]byte, MaxPicture+1)}} {
		if r.Validate() == nil {
			t.Fatal("invalid response accepted")
		}
	}
	if (Result{Matched: true, Title: "Été", Year: 2026, Track: 1}).Validate() != nil {
		t.Fatal("valid response rejected")
	}
}
