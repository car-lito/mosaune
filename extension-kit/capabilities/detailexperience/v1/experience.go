// Package detailexperience defines declarative commands and lists, never executable UI.
package detailexperience

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Capability = "media.detail-experience.v1"

type File struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Name        string  `json:"name"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	AlbumArtist string  `json:"albumArtist"`
	Disc        int     `json:"disc"`
	Number      int     `json:"number"`
	Season      int     `json:"season"`
	Episode     int     `json:"episode"`
	Position    float64 `json:"position"`
	Completed   bool    `json:"completed"`
}
type Track struct {
	Title    string `json:"title"`
	Disc     int    `json:"disc"`
	Number   int    `json:"number"`
	Duration int    `json:"duration"`
	People   []int  `json:"people"`
}
type Query struct {
	Kind         string   `json:"kind"`
	Focus        string   `json:"focus"`
	Work         bool     `json:"work"`
	Provider     string   `json:"provider"`
	Files        []File   `json:"files"`
	Tracks       []Track  `json:"tracks"`
	People       []string `json:"people"`
	Contributors []int    `json:"contributors"`
}
type Action struct {
	Operation string `json:"operation"`
	Label     string `json:"label"`
	FileID    string `json:"fileId,omitempty"`
	Primary   bool   `json:"primary,omitempty"`
}
type Row struct {
	Key      string   `json:"key"`
	FileID   string   `json:"fileId,omitempty"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Ordinal  string   `json:"ordinal"`
	Duration int      `json:"duration"`
	Actions  []Action `json:"actions"`
}
type Section struct {
	Key     string   `json:"key"`
	Title   string   `json:"title"`
	Help    string   `json:"help"`
	Rows    []Row    `json:"rows"`
	Actions []Action `json:"actions"`
}
type Suggestions struct {
	Enabled bool   `json:"enabled"`
	Title   string `json:"title"`
	Help    string `json:"help"`
}
type Result struct {
	Cover       string      `json:"cover"`
	Actions     []Action    `json:"actions"`
	People      []int       `json:"people"`
	Sections    []Section   `json:"sections"`
	EpisodeList bool        `json:"episodeList"`
	AlbumLink   bool        `json:"albumLink"`
	Suggestions Suggestions `json:"suggestions"`
}

func text(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func bounded(v any, n int) bool { b, e := json.Marshal(v); return e == nil && len(b) <= n }
func (q Query) Validate() error {
	bad := errors.New("invalid detail experience facts")
	if !bounded(q, 512<<10) || !text(q.Kind, 80) || q.Kind == "" || !text(q.Provider, 100) || len(q.Files) > 1000 || len(q.Tracks) > 1000 || len(q.People) > 1000 {
		return bad
	}
	seen := map[string]bool{}
	for _, f := range q.Files {
		if f.ID == "" || seen[f.ID] || !text(f.ID, 100) || !text(f.Title, 2048) || !text(f.Name, 2048) || !text(f.Artist, 2048) || !text(f.AlbumArtist, 2048) || !text(f.Album, 2048) || f.Number < 0 || f.Number > 1000000 || f.Disc < 0 || f.Disc > 1000000 || f.Season < 0 || f.Season > 1000000 || f.Episode < 0 || f.Episode > 1000000 || math.IsNaN(f.Position) || math.IsInf(f.Position, 0) || f.Position < 0 || f.Position > 1e9 {
			return bad
		}
		seen[f.ID] = true
	}
	if q.Focus != "" && !seen[q.Focus] {
		return bad
	}
	for _, s := range q.People {
		if s == "" || !text(s, 2048) {
			return bad
		}
	}
	refs := func(a []int) bool {
		if len(a) > 1000 {
			return false
		}
		for _, i := range a {
			if i < 0 || i >= len(q.People) {
				return false
			}
		}
		return true
	}
	if !refs(q.Contributors) {
		return bad
	}
	for _, t := range q.Tracks {
		if !text(t.Title, 2048) || t.Disc < 0 || t.Disc > 1000000 || t.Number < 0 || t.Number > 1000000 || t.Duration < 0 || t.Duration > 10000000 || !refs(t.People) {
			return bad
		}
	}
	return nil
}
func (r Result) Validate(q Query) error {
	bad := errors.New("invalid detail experience result")
	if q.Validate() != nil || !bounded(r, 768<<10) || r.Cover != "square" && r.Cover != "portrait" || r.Actions == nil || r.Sections == nil || r.People == nil || len(r.Sections) > 8 || len(r.People) > 1000 || r.EpisodeList && (len(r.Sections) > 0 || q.Kind != "series") || r.Suggestions.Enabled && !q.Work || !text(r.Suggestions.Title, 2048) || !text(r.Suggestions.Help, 4096) {
		return bad
	}
	files := map[string]bool{}
	for _, f := range q.Files {
		files[f.ID] = true
	}
	actions := func(aa []Action) bool {
		if aa == nil || len(aa) > 16 {
			return false
		}
		for _, a := range aa {
			if a.Label == "" || !text(a.Label, 500) || a.FileID != "" && !files[a.FileID] {
				return false
			}
			switch a.Operation {
			case "play", "open", "inspect", "mark", "poster", "match":
				if a.FileID == "" && (a.Operation != "poster" || !q.Work) {
					return false
				}
			case "enqueue", "playlist":
				if len(files) == 0 {
					return false
				}
			case "watchlist", "favorite", "request":
				if !q.Work || a.FileID != "" {
					return false
				}
			case "unavailable":
				if a.FileID != "" {
					return false
				}
			default:
				return false
			}
		}
		return true
	}
	if !actions(r.Actions) {
		return bad
	}
	seenPeople := map[int]bool{}
	for _, i := range r.People {
		if i < 0 || i >= len(q.People) || seenPeople[i] {
			return bad
		}
		seenPeople[i] = true
	}
	keys := map[string]bool{}
	ids := map[string]bool{}
	rows := 0
	for _, s := range r.Sections {
		if s.Key == "" || keys[s.Key] || !text(s.Key, 100) || !text(s.Title, 2048) || !text(s.Help, 4096) || s.Rows == nil || !actions(s.Actions) {
			return bad
		}
		keys[s.Key] = true
		rowkeys := map[string]bool{}
		for _, row := range s.Rows {
			rows++
			if rows > 2000 || row.Key == "" || rowkeys[row.Key] || !text(row.Key, 100) || !text(row.Title, 2048) || !text(row.Subtitle, 8192) || !text(row.Ordinal, 100) || row.Duration < 0 || row.Duration > 10000000 || !actions(row.Actions) {
				return bad
			}
			rowkeys[row.Key] = true
			if row.FileID != "" {
				if !files[row.FileID] || ids[row.FileID] {
					return bad
				}
				ids[row.FileID] = true
			}
			for _, a := range row.Actions {
				if a.FileID != "" && a.FileID != row.FileID {
					return bad
				}
			}
		}
	}
	// An extension may organize files but cannot silently lose an accessible file.
	if !r.EpisodeList {
		for id := range files {
			if q.Focus != "" && id != q.Focus {
				continue
			}
			if !ids[id] {
				return bad
			}
		}
	}
	return nil
}
