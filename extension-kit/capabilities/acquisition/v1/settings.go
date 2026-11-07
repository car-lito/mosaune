package acquisition

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Settings describes a form, never credentials or an executable UI. Bindings
// identify host-owned controls and option sources, not database keys or URLs.
type Settings struct {
	Version      int             `json:"version"`
	Title        string          `json:"title"`
	Instructions []string        `json:"instructions"`
	Fields       []SettingsField `json:"fields"`
}
type SettingsField struct {
	Binding     string `json:"binding"`
	Label       string `json:"label"`
	Help        string `json:"help,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`
}

func (s Settings) Validate() error {
	text := func(v string, max int) bool {
		return utf8.ValidString(v) && utf8.RuneCountInString(v) <= max && !strings.ContainsAny(v, "\x00\r")
	}
	bad := errors.New("invalid acquisition settings form")
	if s.Version != 1 || strings.TrimSpace(s.Title) == "" || !text(s.Title, 160) || len(s.Instructions) > 12 || len(s.Fields) != 5 {
		return bad
	}
	for _, line := range s.Instructions {
		if !text(line, 2000) {
			return bad
		}
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		if seen[f.Binding] || strings.TrimSpace(f.Label) == "" || !text(f.Label, 160) || !text(f.Help, 2000) || !text(f.Placeholder, 256) {
			return bad
		}
		seen[f.Binding] = true
		switch f.Binding {
		case "url":
			if f.MaxLength < 1 || f.MaxLength > 2048 {
				return bad
			}
		case "key":
			if f.MaxLength < 1 || f.MaxLength > 200 {
				return bad
			}
		case "profile", "root", "library":
			if f.MaxLength != 0 {
				return bad
			}
		default:
			return bad
		}
	}
	return nil
}

// ValidateConnection runs in the host. Secrets must never be sent to the module.
func (s Settings) ValidateConnection(address, key string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	for _, f := range s.Fields {
		value := ""
		if f.Binding == "url" {
			value = address
		} else if f.Binding == "key" {
			value = key
		} else {
			continue
		}
		if !utf8.ValidString(value) || utf8.RuneCountInString(value) > f.MaxLength {
			return errors.New("Le champ « " + f.Label + " » dépasse la taille autorisée par le connecteur.")
		}
	}
	return nil
}
