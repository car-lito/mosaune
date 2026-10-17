package configuration

import (
	"encoding/json"
	"strings"
	"testing"
)

func sample() Schema {
	return Schema{Version: 1, Title: "Example", Fields: []Field{{ID: "token", Label: "Key", Type: "secret", MinLength: 20, MaxLength: 100, Format: "token"}, {ID: "title", Label: "Title", Type: "text", MaxLength: 100}, {ID: "enabled", Label: "Enabled", Type: "boolean"}, {ID: "region", Label: "Region", Type: "select", Options: []Option{{Value: "be", Label: "Belgique"}, {Value: "fr", Label: "France"}}}}}
}
func TestPatchValidationAndSecretSemantics(t *testing.T) {
	schema := sample()
	for _, raw := range []string{`{"values":{}}`, `{"values":{"enabled":false,"region":"be","title":"Nom"}}`, `{"values":{"token":"  01234567890123456789  "}}`, `{"clearSecrets":["token"]}`} {
		var p Patch
		_ = json.Unmarshal([]byte(raw), &p)
		out, e := schema.NormalizePatch(p)
		if e != nil {
			t.Fatal(raw, e)
		}
		if v, ok := out.Values["token"]; ok && string(v) != `"01234567890123456789"` {
			t.Fatal("token normalization", string(v))
		}
	}
	for _, raw := range []string{`{"values":{"unknown":true}}`, `{"values":{"token":"short"}}`, `{"values":{"token":null}}`, `{"values":{"enabled":"true"}}`, `{"values":{"region":"xx"}}`, `{"values":{"title":false}}`, `{"values":{"token":"Bearer 01234567890123456789"}}`, `{"clearSecrets":["title"]}`, `{"clearSecrets":["token","token"]}`, `{"values":{"token":"01234567890123456789"},"clearSecrets":["token"]}`} {
		var p Patch
		_ = json.Unmarshal([]byte(raw), &p)
		if _, e := schema.NormalizePatch(p); e == nil {
			t.Fatal("accepted", raw)
		} else if strings.Contains(e.Error(), "012345") {
			t.Fatal("secret in error")
		}
	}
}
func TestSchemaRejectsUnsupportedDefinitions(t *testing.T) {
	for _, mutate := range []func(*Schema){func(s *Schema) { s.Version = 2 }, func(s *Schema) { s.Fields[0].Type = "javascript" }, func(s *Schema) { s.Fields[0].MaxLength = 1000000 }, func(s *Schema) { s.Fields[1].ID = s.Fields[0].ID }, func(s *Schema) { s.Links = []Link{{Label: "Bad", URL: "javascript:alert(1)"}} }, func(s *Schema) { s.Links = []Link{{Label: "Bad", URL: "https://user:secret@example.org"}} }} {
		s := sample()
		mutate(&s)
		if s.Validate() == nil {
			t.Fatal("invalid schema accepted")
		}
	}
}
