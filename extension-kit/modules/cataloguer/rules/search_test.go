package rules

import (
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"testing"
)

func TestSearchCandidatesAcrossFamilies(t *testing.T) {
	f := catalogue.Facts{ID: "one", Name: "File", Title: "Pilot", Series: "Show", Season: 1, Episode: 2, Artist: "Guest", AlbumArtist: "Artist", Album: "Record"}
	for _, project := range []func(catalogue.Facts) catalogue.Item{FilmSearch, SeriesSearch, MusicSearch} {
		item := project(f)
		if item.Search == nil || item.Search.Validate() != nil {
			t.Fatal(item)
		}
	}
	music := MusicSearch(f).Search.Entries
	if len(music) != 2 || music[0].Target != "group" || !music[0].MetadataPresentation || music[1].Presentation.Label != "Morceau" || music[1].Subtitle != "Guest" {
		t.Fatal(music)
	}
	series := SeriesSearch(f).Search.Entries
	if len(series) != 2 || series[1].Presentation.Preview.AspectRatio != 16.0/9 {
		t.Fatal(series)
	}
	f.Album = ""
	if len(MusicSearch(f).Search.Entries) != 1 {
		t.Fatal("invented album")
	}
	f.Series = ""
	if len(SeriesSearch(f).Search.Entries) != 1 {
		t.Fatal("invented series")
	}
}
