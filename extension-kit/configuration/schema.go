// Package configuration describes host-rendered forms. It never grants storage,
// network or executable UI access; the host binds a schema to an authorized store.
package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}
type Field struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Type      string   `json:"type"`
	Help      string   `json:"help,omitempty"`
	MinLength int      `json:"minLength,omitempty"`
	MaxLength int      `json:"maxLength,omitempty"`
	Format    string   `json:"format,omitempty"`
	Options   []Option `json:"options,omitempty"`
}
type Schema struct {
	Version      int      `json:"version"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Instructions []string `json:"instructions,omitempty"`
	Links        []Link   `json:"links,omitempty"`
	Fields       []Field  `json:"fields"`
}
type Patch struct {
	Revision     *int64                     `json:"revision,omitempty"`
	Values       map[string]json.RawMessage `json:"values"`
	ClearSecrets []string                   `json:"clearSecrets,omitempty"`
}

// State deliberately separates ordinary values from the presence of secrets.
// No API response may contain a secret value, even for administrators.
type State struct {
	Revision          int64                      `json:"revision"`
	Schema            Schema                     `json:"schema"`
	Values            map[string]json.RawMessage `json:"values"`
	SecretsConfigured map[string]bool            `json:"secretsConfigured"`
}

var fieldID = regexp.MustCompile(`^[a-z][a-zA-Z0-9_-]{0,47}$`)

func text(s string, max int) bool {
	return utf8.ValidString(s) && len(s) <= max && !strings.ContainsAny(s, "\x00\r")
}
func (s Schema) Validate() error {
	invalid := errors.New("invalid configuration schema")
	if s.Version != 1 || s.Title == "" || !text(s.Title, 160) || !text(s.Description, 2000) || len(s.Fields) == 0 || len(s.Fields) > 32 || len(s.Instructions) > 12 || len(s.Links) > 8 {
		return invalid
	}
	for _, line := range s.Instructions {
		if !text(line, 2000) {
			return invalid
		}
	}
	for _, link := range s.Links {
		u, e := url.Parse(link.URL)
		if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(link.URL) > 2048 || link.Label == "" || !text(link.Label, 160) {
			return invalid
		}
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		if !fieldID.MatchString(f.ID) || seen[f.ID] || f.Label == "" || !text(f.Label, 160) || !text(f.Help, 2000) {
			return invalid
		}
		seen[f.ID] = true
		switch f.Type {
		case "text", "secret":
			if f.MinLength < 0 || f.MaxLength < 1 || f.MaxLength > 4096 || f.MinLength > f.MaxLength || len(f.Options) > 0 || (f.Format != "" && f.Format != "token") {
				return invalid
			}
		case "boolean", "select":
			if f.MinLength != 0 || f.MaxLength != 0 || f.Format != "" {
				return invalid
			}
			if f.Type == "boolean" && len(f.Options) != 0 {
				return invalid
			}
			if f.Type == "select" {
				if len(f.Options) == 0 || len(f.Options) > 32 {
					return invalid
				}
				options := map[string]bool{}
				for _, o := range f.Options {
					if o.Value == "" || !text(o.Value, 128) || options[o.Value] || o.Label == "" || !text(o.Label, 160) {
						return invalid
					}
					options[o.Value] = true
				}
			}
		default:
			return invalid
		}
	}
	return nil
}

// NormalizePatch checks all changes before any write. Missing fields mean keep
// the current value. Empty secret inputs are omitted by clients, never deletion.
func (s Schema) NormalizePatch(p Patch) (Patch, error) {
	invalid := errors.New("Valeurs de réglages invalides. Vérifiez les champs indiqués.")
	if s.Validate() != nil || len(p.Values) > 32 || len(p.ClearSecrets) > 32 {
		return Patch{}, invalid
	}
	fields := map[string]Field{}
	for _, f := range s.Fields {
		fields[f.ID] = f
	}
	if p.Revision != nil && *p.Revision < 0 {
		return Patch{}, invalid
	}
	out := Patch{Values: map[string]json.RawMessage{}, Revision: p.Revision}
	for key, raw := range p.Values {
		f, ok := fields[key]
		if !ok {
			return Patch{}, invalid
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return Patch{}, invalid
		}
		switch f.Type {
		case "text", "secret", "select":
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return Patch{}, invalid
			}
			if f.Format == "token" {
				value = strings.TrimSpace(value)
				if strings.ContainsAny(value, " \r\n\t") {
					return Patch{}, invalid
				}
			}
			if f.Type == "select" {
				valid := false
				for _, o := range f.Options {
					if value == o.Value {
						valid = true
					}
				}
				if !valid {
					return Patch{}, invalid
				}
			} else if !text(value, f.MaxLength) || len(value) < f.MinLength {
				return Patch{}, invalid
			}
			out.Values[key], _ = json.Marshal(value)
		case "boolean":
			var value bool
			if json.Unmarshal(raw, &value) != nil {
				return Patch{}, invalid
			}
			out.Values[key], _ = json.Marshal(value)
		}
	}
	seen := map[string]bool{}
	for _, key := range p.ClearSecrets {
		f, ok := fields[key]
		_, changed := p.Values[key]
		if !ok || f.Type != "secret" || seen[key] || changed {
			return Patch{}, invalid
		}
		seen[key] = true
		out.ClearSecrets = append(out.ClearSecrets, key)
	}
	return out, nil
}
