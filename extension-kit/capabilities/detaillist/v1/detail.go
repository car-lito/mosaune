// Package detaillist describes a bounded, declarative grouped file list.
package detaillist

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Capability = "media.detail-list.v1"

type Entry struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
}
type Season struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Episodes int    `json:"episodes"`
}
type Query struct {
	Entries []Entry  `json:"entries"`
	Seasons []Season `json:"seasons"`
}
type Group struct {
	Key           string `json:"key"`
	Title         string `json:"title"`
	Total         int    `json:"total"`
	Artwork       *int   `json:"artwork,omitempty"`
	RequestSeason *int   `json:"requestSeason,omitempty"`
}
type Row struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Ordinal string `json:"ordinal"`
	Group   string `json:"group"`
}
type Result struct {
	Title  string  `json:"title"`
	All    string  `json:"all"`
	Files  string  `json:"files"`
	Help   string  `json:"help"`
	Empty  string  `json:"empty"`
	Groups []Group `json:"groups"`
	Rows   []Row   `json:"rows"`
}

func text(s string) bool {
	return utf8.ValidString(s) && len(s) <= 1024 && strings.IndexFunc(s, unicode.IsControl) < 0
}
func bounded(v any) bool { b, e := json.Marshal(v); return e == nil && len(b) <= 192<<10 }
func (q Query) Validate() error {
	raw, _ := json.Marshal(q)
	if len(raw) > 48<<10 {
		return errors.New("detail facts too large")
	}
	bad := errors.New("invalid detail facts")
	if len(q.Entries) > 1000 || len(q.Seasons) > 128 || !bounded(q) {
		return bad
	}
	ids := map[string]bool{}
	seasons := map[int]bool{}
	for _, e := range q.Entries {
		if e.ID == "" || ids[e.ID] || !text(e.ID) || !text(e.Title) || e.Season < 0 || e.Season > 1000000 || e.Episode < 0 || e.Episode > 1000000 {
			return bad
		}
		ids[e.ID] = true
	}
	for _, s := range q.Seasons {
		if seasons[s.Number] || s.Number < 0 || s.Number > 1000000 || s.Episodes < 0 || s.Episodes > 1000000 || !text(s.Name) {
			return bad
		}
		seasons[s.Number] = true
	}
	return nil
}
func (r Result) Validate(q Query) error {
	bad := errors.New("invalid detail result")
	if r.Groups == nil || r.Rows == nil || q.Validate() != nil || len(r.Groups) > 128 || len(r.Rows) != len(q.Entries) || !bounded(r) {
		return bad
	}
	for _, s := range []string{r.Title, r.All, r.Files, r.Help, r.Empty} {
		if !text(s) {
			return bad
		}
	}
	if strings.TrimSpace(r.Files) == "" || strings.TrimSpace(r.Empty) == "" {
		return bad
	}
	ids := map[string]bool{}
	seasons := map[int]bool{}
	for _, e := range q.Entries {
		ids[e.ID] = true
		seasons[e.Season] = true
	}
	for _, s := range q.Seasons {
		seasons[s.Number] = true
	}
	groups := map[string]bool{}
	for _, g := range r.Groups {
		if g.Key == "" || groups[g.Key] || !text(g.Key) || !text(g.Title) || strings.TrimSpace(g.Title) == "" || g.Total < 0 || g.Total > 1000000 {
			return bad
		}
		groups[g.Key] = true
		if g.Artwork != nil && (*g.Artwork < 0 || *g.Artwork >= len(q.Seasons)) {
			return bad
		}
		if g.RequestSeason != nil && !seasons[*g.RequestSeason] {
			return bad
		}
	}
	if len(r.Groups) > 0 && (strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.All) == "") {
		return bad
	}
	for _, row := range r.Rows {
		if !ids[row.ID] || !text(row.Title) || strings.TrimSpace(row.Title) == "" || !text(row.Ordinal) || !text(row.Group) || (row.Group != "" && !groups[row.Group]) {
			return bad
		}
		delete(ids, row.ID)
	}
	return nil
}
