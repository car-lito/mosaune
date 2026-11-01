package rules

import (
	"fmt"
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	presentation "mosaune/extensionkit/presentation/v1"
	"unicode/utf8"
)

func searchEntry(target, title, subtitle, label string, ratio float64, priority int, terms ...string) catalogue.SearchEntry {
	limit := func(s string) string {
		if len(s) <= 2048 {
			return s
		}
		s = s[:2048]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		return s
	}
	title = limit(title)
	subtitle = limit(subtitle)
	return catalogue.SearchEntry{Target: target, Title: title, Subtitle: subtitle, Priority: priority, Terms: terms, Presentation: &presentation.Result{Version: 1, Label: label, Preview: &presentation.Preview{AspectRatio: ratio, Shape: "rectangle", Fit: "cover"}}}
}
func withSearch(v catalogue.Item, entries ...catalogue.SearchEntry) catalogue.Item {
	v.Search = &catalogue.SearchIndex{Version: 1, Entries: entries}
	return v
}
func FilmSearch(f catalogue.Facts) catalogue.Item {
	v := Film(f)
	return withSearch(v, searchEntry("group", v.Title, v.Subtitle, "Film", 2.0/3, 50, f.Name, f.Title, v.Subtitle))
}
func SeriesSearch(f catalogue.Facts) catalogue.Item {
	v := Series(f)
	episode := searchEntry("item", first(f.Title, f.Name), fmt.Sprintf("%s · S%d · E%d", f.Series, f.Season, f.Episode), "Épisode", 16.0/9, 40, f.Name, f.Title, f.Series)
	if v.Key == "" {
		return withSearch(v, episode)
	}
	return withSearch(v, searchEntry("group", v.Title, v.Subtitle, "Série", 2.0/3, 60, f.Series, f.Name, f.Title), episode)
}
func MusicSearch(f catalogue.Facts) catalogue.Item {
	v := Music(f)
	track := searchEntry("item", first(f.Title, f.Name), first(f.Artist, f.AlbumArtist, "Artiste inconnu"), "Morceau", 1, 40, f.Title, f.Name, f.Artist, f.AlbumArtist, f.Album)
	if v.Key == "" {
		return withSearch(v, track)
	}
	album := searchEntry("group", v.Title, v.Subtitle, "Album", 1, 60, f.Album, f.Artist, f.AlbumArtist, f.Title)
	album.MetadataPresentation = true
	return withSearch(v, album, track)
}
