package detailexperience

import (
	"fmt"
	d "mosaune/extensionkit/capabilities/detailexperience/v1"
	"testing"
)

func checked(t *testing.T, q d.Query) d.Result {
	t.Helper()
	r := Project(q)
	if e := r.Validate(q); e != nil {
		t.Fatal(e, r)
	}
	return r
}
func TestAlbumMatchingMissingAmbiguityAndFocus(t *testing.T) {
	q := d.Query{Kind: "music", Work: true, People: []string{"Album artist", "Guest"}, Contributors: []int{0}, Files: []d.File{{ID: "one", Title: "L’été", Name: "one", Disc: 1, Number: 1, Artist: "Guest", Album: "Album"}, {ID: "dup1", Title: "Same", Name: "dup1", Disc: 1, Number: 4}, {ID: "dup2", Title: "Same", Name: "dup2", Disc: 1, Number: 4}, {ID: "bonus", Title: "Bonus", Name: "bonus"}}, Tracks: []d.Track{{Title: "L’ete", Disc: 1, Number: 1, People: []int{1}}, {Title: "Absent", Number: 2}, {Title: "Same", Disc: 1, Number: 4}}}
	r := checked(t, q)
	if r.Cover != "square" || len(r.Sections) != 2 || r.Sections[0].Rows[0].FileID != "one" || r.Sections[0].Rows[1].FileID != "" || r.Sections[0].Rows[2].FileID != "" || len(r.Sections[1].Rows) != 3 || r.People[0] != 0 {
		t.Fatal(r)
	}
	q.Focus = "one"
	r = checked(t, q)
	if !r.AlbumLink || len(r.Sections) != 1 || len(r.Sections[0].Rows) != 1 || r.Sections[0].Rows[0].FileID != "one" || len(r.People) != 1 || r.People[0] != 1 {
		t.Fatal(r)
	}
	for _, a := range r.Actions {
		if a.Operation == "enqueue" && a.FileID != "one" {
			t.Fatal("focus enqueued album")
		}
	}
}
func TestMoviesSeriesAndUnknownMedia(t *testing.T) {
	q := d.Query{Kind: "films", Work: true, Files: []d.File{{ID: "done", Name: "A", Completed: true}, {ID: "resume", Name: "B", Position: 67}}}
	r := checked(t, q)
	if r.Actions[0].FileID != "resume" || r.Actions[0].Label != "Reprendre" || r.Sections[0].Rows[1].Subtitle != "Reprendre à 1:07" {
		t.Fatal(r)
	}
	q.Kind = "series"
	r = checked(t, q)
	if !r.EpisodeList || len(r.Sections) != 0 {
		t.Fatal(r)
	}
	q.Kind = "games"
	r = checked(t, q)
	if r.EpisodeList || len(r.Sections) != 1 {
		t.Fatal(r)
	}
	q.Kind = "music"
	q.Files = nil
	r = checked(t, q)
	if r.Actions[2].Operation != "unavailable" {
		t.Fatal(r)
	}
}
func TestLargeAlbum(t *testing.T) {
	q := d.Query{Kind: "music", Files: []d.File{}, Tracks: []d.Track{}}
	for i := 0; i < 1000; i++ {
		name := fmt.Sprintf("Track %d", i)
		q.Files = append(q.Files, d.File{ID: fmt.Sprint(i), Title: name, Name: name, Number: i + 1})
		q.Tracks = append(q.Tracks, d.Track{Title: name, Number: i + 1})
	}
	r := checked(t, q)
	if len(r.Sections[0].Rows) != 1000 || len(r.Sections) != 1 {
		t.Fatal("missing files")
	}
}
