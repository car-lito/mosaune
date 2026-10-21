// Package recognition describes names only. It never grants filesystem access.
package recognition

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const Capability = "media.recognition.v1"
const BatchSize = 32
const MaxPathBytes = 1024

type Formats struct {
	Extensions []string `json:"extensions"`
}
type Query struct {
	Paths []string `json:"paths"`
}
type Item struct {
	Accepted   bool   `json:"accepted"`
	Title      string `json:"title,omitempty"`
	Year       int    `json:"year,omitempty"`
	Series     string `json:"series,omitempty"`
	Season     int    `json:"season,omitempty"`
	Episode    int    `json:"episode,omitempty"`
	EpisodeEnd int    `json:"episodeEnd,omitempty"`
}

// Items correspond by position, never by a path chosen by the extension.
type Result struct {
	Items []Item `json:"items"`
}

var suffix = regexp.MustCompile(`^\.[a-z0-9]{1,12}$`)

func (f Formats) Validate() error {
	if len(f.Extensions) == 0 || len(f.Extensions) > 64 {
		return errors.New("invalid format list")
	}
	seen := map[string]bool{}
	for _, s := range f.Extensions {
		if !suffix.MatchString(s) || seen[s] {
			return errors.New("invalid format")
		}
		seen[s] = true
	}
	return nil
}
func (f Formats) Accepts(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	for _, s := range f.Extensions {
		if ext == s {
			return true
		}
	}
	return false
}
func (q Query) Validate() error {
	if len(q.Paths) == 0 || len(q.Paths) > BatchSize {
		return errors.New("invalid recognition batch")
	}
	for _, p := range q.Paths {
		if !utf8.ValidString(p) || len(p) > MaxPathBytes || !fs.ValidPath(p) || p == "." || strings.ContainsAny(p, "\\\x00:") {
			return errors.New("invalid relative path")
		}
	}
	return nil
}
func (r Result) Validate(q Query, f Formats) error {
	if q.Validate() != nil || f.Validate() != nil || len(r.Items) != len(q.Paths) {
		return errors.New("invalid recognition reply")
	}
	for i, v := range r.Items {
		if v.Accepted != f.Accepts(q.Paths[i]) {
			return errors.New("recognition formats disagree")
		}
		if !v.Accepted {
			if v != (Item{}) {
				return errors.New("metadata on rejected file")
			}
			continue
		}
		if !utf8.ValidString(v.Title) || !utf8.ValidString(v.Series) || strings.TrimSpace(v.Title) == "" || len(v.Title) > 4096 || len(v.Series) > 4096 || strings.ContainsAny(v.Title+v.Series, "\x00\r\n") || v.Year < 0 || v.Year > 9999 || v.Season < 0 || v.Season > 9999 || v.Episode < 0 || v.Episode > 99999 || v.EpisodeEnd < 0 || v.EpisodeEnd > 99999 {
			return errors.New("invalid recognized fields")
		}
	}
	return nil
}
