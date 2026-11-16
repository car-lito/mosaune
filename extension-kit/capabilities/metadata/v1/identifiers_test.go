package v1

import (
	"mosaune/extensionkit/transport"
	"testing"
)

func TestIdentifierRoutesAndValues(t *testing.T) {
	p := transport.Profile{Sources: []transport.Source{{ID: "api", BaseURL: "https://example.test/api", Format: "json"}}}
	r := IdentifierRoute{Kind: "series", Namespace: "tvdb", Source: "api", Path: "/series/{id}/identifiers", Field: "external"}
	q := IdentifierQuery{Kind: "series", ID: "42", Namespace: "tvdb"}
	_, s, path, e := IdentifierRequest([]IdentifierRoute{r}, p, q)
	if e != nil || path != "/series/42/identifiers" || s.ID != "api" {
		t.Fatal(path, e)
	}
	for _, edit := range []func(*IdentifierRoute){
		func(v *IdentifierRoute) { v.Path = "//evil.test/{id}" }, func(v *IdentifierRoute) { v.Path = "/../{id}" }, func(v *IdentifierRoute) { v.Path = "/%2e/{id}" }, func(v *IdentifierRoute) { v.Path = "/x?key={id}" }, func(v *IdentifierRoute) { v.Path = "/x/{id}/{other}" }, func(v *IdentifierRoute) { v.Source = "undeclared" }, func(v *IdentifierRoute) { v.Field = "nested.field" }, func(v *IdentifierRoute) { v.Namespace = "" },
	} {
		v := r
		edit(&v)
		if ValidateIdentifierRoutes([]IdentifierRoute{v}, &p) == nil {
			t.Fatal("accepted", v)
		}
	}
	if ValidateIdentifierRoutes([]IdentifierRoute{r, r}, &p) == nil {
		t.Fatal("duplicate mapping")
	}
	for _, id := range []string{"../42", "42?x=1", "https://evil.test", "", "..", "%32"} {
		v := q
		v.ID = id
		if _, _, _, e := IdentifierRequest([]IdentifierRoute{r}, p, v); e == nil {
			t.Fatal("accepted ID", id)
		}
	}
	for _, body := range []string{`{"external":84}`, `{"external":"84"}`, `{"external":"tt12345"}`} {
		if _, e := ExtractIdentifier([]byte(body), "external"); e != nil {
			t.Fatal(body, e)
		}
	}
	for _, body := range []string{`{}`, `{"external":null}`, `{"external":0}`, `{"external":-1}`, `{"external":1.5}`, `{"external":1e3}`, `{"external":{}}`, `{"external":"../42"}`, `{"external":[]}`} {
		if v, e := ExtractIdentifier([]byte(body), "external"); e == nil {
			t.Fatal(body, v)
		}
	}
}
