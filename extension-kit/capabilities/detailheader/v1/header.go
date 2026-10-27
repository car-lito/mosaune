// Package detailheader describes the text slots of a media header, never HTML.
package detailheader

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Capability = "media.detail-header.v1"

type Query struct {
	Kind       string   `json:"kind"`
	Track      bool     `json:"track"`
	StoredWork bool     `json:"storedWork"`
	Title      string   `json:"title"`
	Name       string   `json:"name"`
	GroupTitle string   `json:"groupTitle"`
	Album      string   `json:"album"`
	Series     string   `json:"series"`
	Year       int      `json:"year"`
	Runtime    int      `json:"runtime"`
	Genres     []string `json:"genres"`
	Rating     float64  `json:"rating"`
	VoteCount  int      `json:"voteCount"`
	Provider   string   `json:"provider"`
	Tagline    string   `json:"tagline"`
	Edition    string   `json:"edition"`
	LocalCount int      `json:"localCount"`
}
type Fact struct {
	Key  string `json:"key"`
	Text string `json:"text"`
	Note string `json:"note,omitempty"`
}
type Result struct {
	Title    string `json:"title"`
	Eyebrow  string `json:"eyebrow"`
	Facts    []Fact `json:"facts"`
	Tagline  string `json:"tagline"`
	Footnote string `json:"footnote"`
}

func validText(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func bounded(v any, n int) bool { b, e := json.Marshal(v); return e == nil && len(b) <= n }
func (q Query) Validate() error {
	bad := errors.New("invalid header facts")
	if !bounded(q, 16<<10) || strings.TrimSpace(q.Kind) == "" || !validText(q.Kind, 80) || q.Year < 0 || q.Year > 9999 || q.Runtime < 0 || q.Runtime > 1000000 || q.VoteCount < 0 || q.LocalCount < 0 || q.LocalCount > 1000000 || math.IsNaN(q.Rating) || math.IsInf(q.Rating, 0) || q.Rating < 0 || q.Rating > 10 || len(q.Genres) > 32 {
		return bad
	}
	for _, s := range []string{q.Title, q.Name, q.GroupTitle, q.Album, q.Series, q.Tagline, q.Edition} {
		if !validText(s, 2048) {
			return bad
		}
	}
	if !validText(q.Provider, 100) {
		return bad
	}
	for _, s := range q.Genres {
		if !validText(s, 200) {
			return bad
		}
	}
	return nil
}
func (r Result) Validate(q Query) error {
	bad := errors.New("invalid header result")
	if q.Validate() != nil || !bounded(r, 24<<10) || strings.TrimSpace(r.Title) == "" || !validText(r.Title, 2048) || !validText(r.Eyebrow, 500) || !validText(r.Tagline, 2048) || !validText(r.Footnote, 2048) || r.Facts == nil || len(r.Facts) > 16 {
		return bad
	}
	keys := map[string]bool{}
	for _, f := range r.Facts {
		if strings.TrimSpace(f.Key) == "" || keys[f.Key] || !validText(f.Key, 80) || strings.TrimSpace(f.Text) == "" || !validText(f.Text, 8192) || !validText(f.Note, 500) {
			return bad
		}
		keys[f.Key] = true
	}
	return nil
}
