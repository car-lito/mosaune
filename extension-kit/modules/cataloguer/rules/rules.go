package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"strconv"
	"strings"
)

func first(values ...string) string {
	for _, s := range values {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return "Sans titre"
}
func key(values ...string) string {
	raw, _ := json.Marshal(values)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func number(n int) int {
	if n < 0 {
		return 0
	}
	if n > 1000000 {
		return 1000000
	}
	return n
}
func Film(f catalogue.Facts) catalogue.Item {
	subtitle := "Film"
	if f.Year > 0 {
		subtitle = strconv.Itoa(f.Year)
	}
	return catalogue.Item{Title: first(f.Title, f.Name), Subtitle: subtitle, SortText: f.Name}
}
func Series(f catalogue.Facts) catalogue.Item {
	v := catalogue.Item{Title: first(f.Title, f.Name), Order: [2]int{number(f.Season), number(f.Episode)}, SortText: f.Filename}
	if f.Series != "" {
		v.Title = f.Series
		v.Subtitle = "Série"
		if f.ProviderID != "" {
			v.Key = key(f.Provider, f.ProviderID)
		} else {
			v.Key = key(strings.ToLower(f.Series))
		}
	}
	return v
}
func Music(f catalogue.Facts) catalogue.Item {
	artist := first(f.AlbumArtist, f.Artist, "Artiste inconnu")
	v := catalogue.Item{Title: first(f.Title, f.Name), Subtitle: artist, Order: [2]int{number(f.Disc), number(f.Track)}, SortText: f.Filename}
	if f.Album != "" {
		v.Title = f.Album
		v.Key = key(strings.ToLower(artist), strings.ToLower(f.Album))
	}
	return v
}
