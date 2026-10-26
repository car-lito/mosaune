// Package catalogue defines a bounded, read-only projection of catalogue facts.
package catalogue

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Capability = "media.catalogue.v1"
const BatchSize = 128

type Facts struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Title       string `json:"title"`
	Provider    string `json:"provider"`
	ProviderID  string `json:"providerId"`
	Series      string `json:"series"`
	Artist      string `json:"artist"`
	AlbumArtist string `json:"albumArtist"`
	Album       string `json:"album"`
	Year        int    `json:"year"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	Disc        int    `json:"disc"`
	Track       int    `json:"track"`
}
type Query struct {
	Items []Facts `json:"items"`
}

// Empty Key means an individual item. Keys are scoped by host to kind and owner.
// Replies correspond by position and cannot reference hidden catalogue entries.
type Item struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Order    [2]int `json:"order"`
	SortText string `json:"sortText"`
}
type Result struct {
	Items []Item `json:"items"`
}

func text(s string) bool {
	return utf8.ValidString(s) && len(s) <= 2048 && strings.IndexFunc(s, unicode.IsControl) < 0
}
func (q Query) Validate() error {
	raw, _ := json.Marshal(q)
	if len(raw) > 48<<10 {
		return errors.New("catalogue batch too large")
	}
	if len(q.Items) == 0 || len(q.Items) > BatchSize {
		return errors.New("invalid catalogue batch")
	}
	for _, v := range q.Items {
		for _, s := range []string{v.ID, v.Name, v.Filename, v.Title, v.Provider, v.ProviderID, v.Series, v.Artist, v.AlbumArtist, v.Album} {
			if !text(s) {
				return errors.New("invalid catalogue facts")
			}
		}
	}
	return nil
}
func (r Result) Validate(q Query) error {
	if q.Validate() != nil || len(r.Items) != len(q.Items) {
		return errors.New("invalid catalogue reply")
	}
	for _, v := range r.Items {
		if strings.TrimSpace(v.Title) == "" {
			return errors.New("empty catalogue title")
		}
		for _, s := range []string{v.Key, v.Title, v.Subtitle, v.SortText} {
			if !text(s) {
				return errors.New("invalid catalogue label")
			}
		}
		for _, n := range v.Order {
			if n < 0 || n > 1000000 {
				return errors.New("invalid catalogue order")
			}
		}
	}
	return nil
}
