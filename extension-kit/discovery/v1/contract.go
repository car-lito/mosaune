// Package discovery describes a provider's public search and selection surface.
// These declarations grant no network access, credentials, or executable UI.
package discovery

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var identifier = regexp.MustCompile("^[a-z][a-z0-9_.-]{0,79}$")

func Identifier(v string) bool { return identifier.MatchString(v) }

type Search struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Similar     bool   `json:"similar"`
}
type Shelf struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Selector string `json:"selector"`
}
type Surface struct {
	Version int      `json:"version"`
	Search  []Search `json:"search"`
	Shelves []Shelf  `json:"shelves"`
}

func text(v string, max int) bool {
	return strings.TrimSpace(v) != "" && utf8.ValidString(v) && len(v) <= max && strings.IndexFunc(v, unicode.IsControl) < 0
}
func (s Surface) Validate(kinds []string) error {
	bad := errors.New("invalid discovery surface")
	if s.Version != 1 || len(s.Search) > 16 || len(s.Shelves) > 32 {
		return bad
	}
	seen := map[string]bool{}
	for _, v := range s.Search {
		if !Identifier(v.Kind) || !slices.Contains(kinds, v.Kind) || seen[v.Kind] || !text(v.Label, 80) || !text(v.Placeholder, 200) {
			return bad
		}
		seen[v.Kind] = true
	}
	seen = map[string]bool{}
	for _, v := range s.Shelves {
		key := v.Kind + ":" + v.ID
		if !Identifier(v.ID) || !slices.Contains(kinds, v.Kind) || seen[key] || !text(v.Label, 120) || (v.Selector != "kind" && v.Selector != "contributors") {
			return bad
		}
		seen[key] = true
	}
	return nil
}
func (s Surface) ForSearch(kind string) (Search, bool) {
	for _, v := range s.Search {
		if v.Kind == kind {
			return v, true
		}
	}
	return Search{}, false
}
func (s Surface) ForShelf(kind, id string) (Shelf, bool) {
	for _, v := range s.Shelves {
		if v.Kind == kind && v.ID == id {
			return v, true
		}
	}
	return Shelf{}, false
}
