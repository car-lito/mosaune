// Package videocatalog describes bounded video discovery, people and posters.
package videocatalog

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const Capability = "metadata.video-catalog.v1"

type Query struct {
	Kind     string `json:"kind,omitempty"`
	Text     string `json:"text,omitempty"`
	ID       string `json:"id,omitempty"`
	Shelf    string `json:"shelf,omitempty"`
	Page     int    `json:"page"`
	Language string `json:"language"`
	Region   string `json:"region,omitempty"`
}

func text(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func (q Query) Validate(op string) error {
	bad := errors.New("invalid video catalogue query")
	if !text(q.Text, 500) || !text(q.ID, 200) || !text(q.Language, 35) || q.Language == "" || !text(q.Region, 8) || q.Page < 1 || q.Page > 50 {
		return bad
	}
	switch op {
	case "discover":
		if (q.Kind != "films" && q.Kind != "series") || q.Shelf != "" || q.Region != "" || (q.Text != "" && q.ID != "") {
			return bad
		}
	case "shelf":
		if (q.Kind != "films" && q.Kind != "series") || q.Shelf == "" || !text(q.Shelf, 80) || q.Text != "" || q.ID != "" || q.Page != 1 {
			return bad
		}
	case "profile":
		if q.ID == "" || q.Kind != "" || q.Text != "" || q.Shelf != "" || q.Region != "" {
			return bad
		}
	case "posters":
		if (q.Kind != "films" && q.Kind != "series") || q.ID == "" || q.Text != "" || q.Shelf != "" || q.Region != "" || q.Page != 1 {
			return bad
		}
	default:
		return bad
	}
	return nil
}

type Item struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Overview    string `json:"overview"`
	ArtworkRef  string `json:"artworkRef,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
}
type Person struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Biography  string `json:"biography"`
	Detail     string `json:"detail"`
	ArtworkRef string `json:"artworkRef,omitempty"`
}
type Poster struct {
	Ref      string `json:"ref"`
	Language string `json:"language"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}
type Result struct {
	Items   []Item   `json:"items"`
	Person  *Person  `json:"person,omitempty"`
	Posters []Poster `json:"posters,omitempty"`
	Page    int      `json:"page"`
	Pages   int      `json:"pages"`
}

func (r Result) Validate(op string, q Query) error {
	bad := errors.New("invalid video catalogue result")
	if q.Validate(op) != nil || r.Page != q.Page || r.Pages < 1 || r.Pages > 50 || len(r.Items) > 20 || len(r.Posters) > 100 {
		return bad
	}
	if op != "profile" && r.Person != nil || op == "profile" && (r.Person == nil || r.Person.ID != q.ID) || op != "posters" && len(r.Posters) > 0 || op == "posters" && len(r.Items) > 0 {
		return bad
	}
	if (op == "posters" || op == "shelf") && r.Pages != 1 {
		return bad
	}
	seen := map[string]bool{}
	for _, i := range r.Items {
		key := i.Kind + ":" + i.ID
		if i.ID == "" || !text(i.ID, 200) || strings.TrimSpace(i.Title) == "" || !text(i.Title, 500) || !text(i.Overview, 8000) || !text(i.ArtworkRef, 500) || !text(i.ReleaseDate, 40) || i.Year < 0 || i.Year > 3000 || seen[key] || (i.Kind != "films" && i.Kind != "series") || op != "profile" && i.Kind != q.Kind {
			return bad
		}
		seen[key] = true
	}
	if p := r.Person; p != nil {
		if strings.TrimSpace(p.Name) == "" || !text(p.Name, 300) || !text(p.Biography, 12000) || !text(p.Detail, 600) || !text(p.ArtworkRef, 500) {
			return bad
		}
	}
	refs := map[string]bool{}
	for _, p := range r.Posters {
		if p.Ref == "" || !text(p.Ref, 500) || refs[p.Ref] || !text(p.Language, 35) || p.Width < 1 || p.Height < 1 || p.Width > 100000 || p.Height > 100000 {
			return bad
		}
		refs[p.Ref] = true
	}
	return nil
}
