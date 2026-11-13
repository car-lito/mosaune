package acquisition

import "testing"

func TestAvailabilityEvidence(t *testing.T) {
	q := AvailabilityQuery{Scope: PreviewQuery{Kind: "series", Season: 1}, Imported: []EpisodeKey{{1, 1}, {1, 2}, {1, 3}, {2, 1}}, Files: []LocalRange{{1, 1, 2}, {1, 2, 2}, {2, 1, 1}}}
	r, err := q.Eligible()
	if err != nil || r.Count() != 2 || r.Validate(q) != nil {
		t.Fatal(r, err)
	}
	for _, bad := range []AvailabilityResult{
		{Version: 1, Film: true}, {Version: 1, Episodes: []EpisodeKey{{1, 3}}}, {Version: 1, Episodes: []EpisodeKey{{2, 1}}}, {Version: 1, Episodes: []EpisodeKey{{1, 1}, {1, 1}}}, {Version: 0},
	} {
		if bad.Validate(q) == nil {
			t.Fatal("unsupported claim", bad)
		}
	}
	q.Scope.Season = -1
	r, err = q.Eligible()
	if err != nil || r.Count() != 3 {
		t.Fatal(r, err)
	}
	q.Files = []LocalRange{{1, 0, 100000}, {1, 0, 100000}}
	r, err = q.Eligible()
	if err != nil || r.Count() != 3 {
		t.Fatal("range expanded beyond imports", r, err)
	}
	q.Files[0].Last = 100001
	if q.Validate() == nil {
		t.Fatal("unbounded range")
	}
	q = AvailabilityQuery{Scope: PreviewQuery{Kind: "films", Season: -1}, Film: true}
	r, err = q.Eligible()
	if err != nil || r.Count() != 1 {
		t.Fatal(r, err)
	}
	q.Film = false
	if r.Validate(q) == nil {
		t.Fatal("unreadable film accepted")
	}
	q.Files = make([]LocalRange, 5001)
	if q.Validate() == nil {
		t.Fatal("unbounded files")
	}
}
