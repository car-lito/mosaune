package v1

import (
	"encoding/json"
	"errors"
	"mosaune/extensionkit/transport"
	"regexp"
	"strings"
)

// IdentifierQuery names an identifier system, not an acquisition application.
type IdentifierQuery struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
}

// IdentifierRoute belongs to the signed provider manifest. Network origins,
// credentials, cache and budgets remain owned by its installed transport profile.
type IdentifierRoute struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	Path      string `json:"path"`
	Field     string `json:"field"`
}

var identifierName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,47}$`)
var identifierValue = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
var identifierField = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

func (q IdentifierQuery) Validate() error {
	if !identifierName.MatchString(q.Kind) || !identifierName.MatchString(q.Namespace) || !identifierValue.MatchString(q.ID) || q.ID == "." || q.ID == ".." {
		return errors.New("invalid identifier query")
	}
	return nil
}

func ValidateIdentifierRoutes(routes []IdentifierRoute, profile *transport.Profile) error {
	bad := errors.New("invalid identifier routes")
	if len(routes) > 32 || len(routes) > 0 && profile == nil {
		return bad
	}
	seen := map[string]bool{}
	for _, r := range routes {
		key := r.Kind + ":" + r.Namespace
		if !identifierName.MatchString(r.Kind) || !identifierName.MatchString(r.Namespace) || seen[key] || strings.Count(r.Path, "{id}") != 1 || len(r.Path) > 256 || !identifierField.MatchString(r.Field) {
			return bad
		}
		seen[key] = true
		source, err := profile.Source(r.Source)
		if err != nil || source.Format != "json" {
			return bad
		}
		path := strings.ReplaceAll(r.Path, "{id}", "42")
		if strings.ContainsAny(path, "{}") {
			return bad
		}
		if _, err = source.URL(path); err != nil {
			return bad
		}
	}
	return nil
}

func IdentifierRequest(routes []IdentifierRoute, profile transport.Profile, q IdentifierQuery) (IdentifierRoute, transport.Source, string, error) {
	if q.Validate() != nil || ValidateIdentifierRoutes(routes, &profile) != nil {
		return IdentifierRoute{}, transport.Source{}, "", errors.New("invalid identifier request")
	}
	for _, r := range routes {
		if r.Kind == q.Kind && r.Namespace == q.Namespace {
			source, err := profile.Source(r.Source)
			path := strings.ReplaceAll(r.Path, "{id}", q.ID)
			if err == nil {
				_, err = source.URL(path)
			}
			return r, source, path, err
		}
	}
	return IdentifierRoute{}, transport.Source{}, "", errors.New("identifier mapping unavailable in installed provider")
}

// ExtractIdentifier accepts strings and positive decimal integers, never floats,
// objects, null, signed or rounded numbers. Callers enforce their own ID format.
func ExtractIdentifier(body []byte, field string) (string, error) {
	bad := errors.New("identifier unavailable in provider response")
	var fields map[string]json.RawMessage
	if len(body) > 2<<20 || !identifierField.MatchString(field) || json.Unmarshal(body, &fields) != nil {
		return "", bad
	}
	raw := fields[field]
	var value string
	if json.Unmarshal(raw, &value) != nil {
		value = string(raw)
		if value == "" || value[0] < '1' || value[0] > '9' {
			return "", bad
		}
		for _, c := range value {
			if c < '0' || c > '9' {
				return "", bad
			}
		}
	}
	if !identifierValue.MatchString(value) || value == "0" || value == "null" {
		return "", bad
	}
	return value, nil
}
