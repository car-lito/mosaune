// Package detailsections describes text and referenced resources, never HTML or URLs.
package detailsections

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Capability = "media.detail-sections.v1"

type Person struct {
	Name string `json:"name"`
	Role string `json:"role"`
}
type Trailer struct {
	Name     string `json:"name"`
	Language string `json:"language"`
}
type Query struct {
	Kind        string    `json:"kind"`
	Overview    string    `json:"overview"`
	ReleaseDate string    `json:"releaseDate"`
	Status      string    `json:"status"`
	Countries   []string  `json:"countries"`
	Languages   []string  `json:"languages"`
	Cast        []string  `json:"cast"`
	LocalCount  int       `json:"localCount"`
	People      []Person  `json:"people"`
	Trailers    []Trailer `json:"trailers"`
}
type Field struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type Reference struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}
type Block struct {
	Key        string      `json:"key"`
	Region     string      `json:"region"`
	Kind       string      `json:"kind"`
	Title      string      `json:"title"`
	Text       string      `json:"text,omitempty"`
	Fields     []Field     `json:"fields,omitempty"`
	References []Reference `json:"references,omitempty"`
}
type Result struct {
	Blocks []Block `json:"blocks"`
}

func validText(s string, limit int) bool {
	return len(s) <= limit && utf8.ValidString(s) && strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) < 0
}
func size(v any, limit int) bool { b, e := json.Marshal(v); return e == nil && len(b) <= limit }
func (q Query) Validate() error {
	bad := errors.New("invalid section facts")
	if !size(q, 48<<10) || !validText(q.Kind, 80) || q.Kind == "" || !validText(q.Overview, 16000) || !validText(q.ReleaseDate, 100) || !validText(q.Status, 500) || q.LocalCount < 0 || q.LocalCount > 1000000 || len(q.People) > 128 || len(q.Trailers) > 32 {
		return bad
	}
	for _, list := range [][]string{q.Countries, q.Languages, q.Cast} {
		if len(list) > 128 {
			return bad
		}
		for _, s := range list {
			if !validText(s, 500) {
				return bad
			}
		}
	}
	for _, v := range q.People {
		if !validText(v.Name, 500) || !validText(v.Role, 500) {
			return bad
		}
	}
	for _, v := range q.Trailers {
		if !validText(v.Name, 500) || !validText(v.Language, 100) {
			return bad
		}
	}
	return nil
}
func (r Result) Validate(q Query) error {
	bad := errors.New("invalid section result")
	if q.Validate() != nil || r.Blocks == nil || len(r.Blocks) > 24 || !size(r, 96<<10) {
		return bad
	}
	keys := map[string]bool{}
	for _, b := range r.Blocks {
		if b.Key == "" || keys[b.Key] || !validText(b.Key, 80) || !validText(b.Title, 500) || !validText(b.Text, 16000) || (b.Region != "main" && b.Region != "aside" && b.Region != "full") {
			return bad
		}
		keys[b.Key] = true
		switch b.Kind {
		case "paragraph":
			if len(b.Fields) > 0 || len(b.References) > 0 {
				return bad
			}
		case "facts":
			if len(b.Fields) == 0 || len(b.Fields) > 32 || len(b.References) > 0 || b.Text != "" {
				return bad
			}
			for _, f := range b.Fields {
				if strings.TrimSpace(f.Label) == "" || !validText(f.Label, 200) || !validText(f.Value, 2000) {
					return bad
				}
			}
		case "people", "trailers":
			if len(b.Fields) > 0 || len(b.References) == 0 {
				return bad
			}
			limit := len(q.People)
			if b.Kind == "trailers" {
				limit = len(q.Trailers)
			}
			seen := map[int]bool{}
			for _, ref := range b.References {
				if ref.Index < 0 || ref.Index >= limit || seen[ref.Index] || !validText(ref.Label, 500) {
					return bad
				}
				seen[ref.Index] = true
			}
		default:
			return bad
		}
	}
	return nil
}
