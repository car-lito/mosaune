package musicresults

import (
	"encoding/json"
	"errors"
	presentation "mosaune/extensionkit/presentation/v1"
	"regexp"
	"strings"
	"unicode/utf8"
)

const Capability = "metadata.music-results.v1"
const MaxBody = 512 << 10

var ID = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

type Contributor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}
type CatalogTrack struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Disc     int           `json:"disc"`
	Number   int           `json:"number"`
	Duration int           `json:"duration"`
	Artists  []Contributor `json:"artists,omitempty"`
}
type Item struct {
	Presentation  *presentation.Result `json:"presentation,omitempty"`
	DetailVersion int                  `json:"detailVersion,omitempty"`
	ReleaseDate   string               `json:"releaseDate,omitempty"`
	Contributors  []Contributor        `json:"contributors,omitempty"`
	Tracks        []CatalogTrack       `json:"tracks,omitempty"`
	Edition       string               `json:"edition,omitempty"`
	Title         string               `json:"title"`
	Artist        string               `json:"artist,omitempty"`
	Album         string               `json:"album,omitempty"`
	AlbumArtist   string               `json:"albumArtist,omitempty"`
	Year          int                  `json:"year,omitempty"`
	ArtworkRef    string               `json:"artworkRef"`
	ProviderID    string               `json:"providerId,omitempty"`
}

type Query struct {
	Operation string
	ID        string
	Page      int
	Body      json.RawMessage
}
type Result struct {
	Items                  []Item
	Artists                []Contributor
	Name, Detail, ArtistID string
	Pages                  int
}

func (q Query) Validate() error {
	if len(q.Body) == 0 || len(q.Body) > MaxBody || !json.Valid(q.Body) || q.Page < 1 || q.Page > 50 {
		return errors.New("invalid or oversized music response input")
	}
	switch q.Operation {
	case "releases", "artists":
	case "release", "artist", "seed":
		if !ID.MatchString(q.ID) {
			return errors.New("invalid music identity")
		}
	default:
		return errors.New("invalid music operation")
	}
	return nil
}
func text(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && !strings.ContainsAny(s, "\x00")
}
func people(ps []Contributor, max int) bool {
	if len(ps) > max {
		return false
	}
	for _, p := range ps {
		if !ID.MatchString(p.ID) || !text(p.Name, 800) || !text(p.Role, 1200) {
			return false
		}
	}
	return true
}
func (r Result) Validate(q Query) error {
	bad := errors.New("invalid music results")
	if q.Validate() != nil || len(r.Items) > 20 || !people(r.Artists, 30) || !text(r.Name, 1200) || !text(r.Detail, 2000) || r.Pages < 1 || r.Pages > 50 || r.ArtistID != "" && !ID.MatchString(r.ArtistID) {
		return bad
	}
	switch q.Operation {
	case "release":
		if len(r.Items) != 1 || r.Items[0].ProviderID != q.ID {
			return bad
		}
	case "releases":
	case "artists", "artist", "seed":
		if len(r.Items) != 0 {
			return bad
		}
	}
	seen := map[string]bool{}
	for _, m := range r.Items {
		if !ID.MatchString(m.ProviderID) || seen[m.ProviderID] || strings.TrimSpace(m.Title) == "" || !text(m.Title, 2000) || !text(m.Album, 2000) || !text(m.Artist, 2000) || !text(m.AlbumArtist, 2000) || !text(m.Edition, 2000) || !text(m.ReleaseDate, 40) || m.Year < 0 || m.Year > 9999 || m.DetailVersion < 0 || m.DetailVersion > 2 || m.Presentation.Validate() != nil || !people(m.Contributors, 30) || len(m.Tracks) > 200 {
			return bad
		}
		seen[m.ProviderID] = true
		refs := strings.Split(m.ArtworkRef, "/")
		if len(refs) > 2 || refs[0] != m.ProviderID {
			return bad
		}
		for _, id := range refs {
			if !ID.MatchString(id) {
				return bad
			}
		}
		for _, t := range m.Tracks {
			if !text(t.ID, 160) || !text(t.Title, 2000) || t.Disc < 0 || t.Disc > 999 || t.Number < 0 || t.Number > 9999 || t.Duration < 0 || t.Duration > 604800 || !people(t.Artists, 30) {
				return bad
			}
		}
	}
	return nil
}
