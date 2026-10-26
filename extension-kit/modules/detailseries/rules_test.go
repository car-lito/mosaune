package detailseries

import (
	d "mosaune/extensionkit/capabilities/detaillist/v1"
	"testing"
)

func TestSparseSeasonsSpecialsAndStableOrder(t *testing.T) {
	q := d.Query{Entries: []d.Entry{{ID: "ten", Title: "Ten", Season: 1, Episode: 10}, {ID: "special", Title: "Special", Season: 0}, {ID: "two", Title: "Two", Season: 1, Episode: 2}, {ID: "unlisted", Title: "New", Season: 3, Episode: 1}}, Seasons: []d.Season{{Number: 2, Name: "Future", Episodes: 8}, {Number: 1, Name: "Première saison", Episodes: 12}, {Number: 9, Name: "Empty", Episodes: 0}}}
	r := Project(q)
	if e := r.Validate(q); e != nil {
		t.Fatal(e)
	}
	if len(r.Groups) != 4 || r.Groups[0].Title != "Épisodes spéciaux" || r.Groups[1].Title != "Première saison" || *r.Groups[1].Artwork != 1 || r.Groups[2].Title != "Future" || r.Groups[3].Title != "Saison 3" {
		t.Fatal(r.Groups)
	}
	for i, id := range []string{"special", "two", "ten", "unlisted"} {
		if r.Rows[i].ID != id {
			t.Fatal(r.Rows)
		}
	}
}
