package rules

import (
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"testing"
)

func TestGroupingIdentityAndOrder(t *testing.T) {
	a := Music(catalogue.Facts{Name: "one", AlbumArtist: "Artist", Artist: "Guest", Album: "Album", Disc: 2, Track: 10})
	b := Music(catalogue.Facts{Name: "two", Artist: "ARTIST", Album: "ALBUM", Disc: 1, Track: 2})
	if a.Key == "" || a.Key != b.Key || a.Title != "Album" || a.Order != [2]int{2, 10} {
		t.Fatal(a, b)
	}
	if a.Key == Music(catalogue.Facts{Artist: "Other", Album: "Album"}).Key {
		t.Fatal("artists merged")
	}
	if Music(catalogue.Facts{Title: "single"}).Key != "" {
		t.Fatal("unidentified single grouped")
	}
	x := Series(catalogue.Facts{Series: "Show", Provider: "tmdb", ProviderID: "1", Season: 1, Episode: 10})
	y := Series(catalogue.Facts{Series: "Localized", Provider: "tmdb", ProviderID: "1", Season: 1, Episode: 2})
	if x.Key != y.Key || x.Order[1] != 10 {
		t.Fatal(x, y)
	}
	if x.Key == Series(catalogue.Facts{Series: "Show", Provider: "other", ProviderID: "1"}).Key {
		t.Fatal("provider collision")
	}
	if f := Film(catalogue.Facts{Title: "Film", Year: 2008}); f.Key != "" || f.Subtitle != "2008" {
		t.Fatal(f)
	}
}
