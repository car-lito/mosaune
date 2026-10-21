package recognition

import (
	"strings"
	"testing"
)

func TestContractRejectsUntrustedNamesAndReplies(t *testing.T) {
	f := Formats{Extensions: []string{".mkv"}}
	for _, p := range []string{"../film.mkv", "/film.mkv", "C:/film.mkv", "folder\\film.mkv", "a/../film.mkv", ".", "a\x00.mkv", strings.Repeat("a", MaxPathBytes+1), string([]byte{0xff})} {
		if (Query{Paths: []string{p}}).Validate() == nil {
			t.Errorf("accepted unsafe name %q", p)
		}
	}
	if (Query{Paths: make([]string, BatchSize+1)}).Validate() == nil {
		t.Fatal("batch limit")
	}
	q := Query{Paths: []string{"Film.MKV", "note.txt"}}
	good := Result{Items: []Item{{Accepted: true, Title: "Film"}, {}}}
	if good.Validate(q, f) != nil {
		t.Fatal("valid reply rejected")
	}
	for _, bad := range []Result{
		{Items: good.Items[:1]},
		{Items: []Item{{Accepted: true, Title: "Film"}, {Accepted: true, Title: "Note"}}},
		{Items: []Item{{Accepted: true}, {}}},
		{Items: []Item{{Accepted: true, Title: "Film", Season: -1}, {}}},
		{Items: []Item{{Accepted: true, Title: "Film"}, {Year: 2000}}},
	} {
		if bad.Validate(q, f) == nil {
			t.Fatalf("accepted invalid result %+v", bad)
		}
	}
	for _, formats := range []Formats{{}, {Extensions: []string{".mkv", ".mkv"}}, {Extensions: []string{"../mkv"}}, {Extensions: []string{".MKV"}}} {
		if formats.Validate() == nil {
			t.Fatal("invalid formats accepted")
		}
	}
}
