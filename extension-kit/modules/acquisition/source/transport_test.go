package main

import (
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"strings"
	"testing"
)

func TestTransportProfiles(t *testing.T) {
	for _, connector := range []string{"radarr", "sonarr"} {
		p, err := transport(connector)
		if err != nil {
			t.Fatal(err)
		}
		for op, route := range p.Routes {
			id, page := 0, 0
			if op == "find" || op == "lookup" || op == "update" || op == "episodes" || op == "episodefiles" {
				id = 42
			}
			if op == "queue" {
				page = 2
			}
			method, path, err := p.Resolve(connector, op, id, page)
			if err != nil || method != route.Method || !strings.HasPrefix(path, "api/v3/") || strings.ContainsAny(path, "{}") {
				t.Fatal(op, path, err)
			}
		}
		_, path, err := p.Resolve(connector, "lookup", 42, 0)
		prefix := "tmdb"
		if connector == "sonarr" {
			prefix = "tvdb"
		}
		if err != nil || !strings.Contains(path, "term="+prefix+"%3A42") {
			t.Fatal(path, err)
		}
		for _, bad := range []struct {
			op       string
			id, page int
		}{{"delete", 0, 0}, {"find", 0, 0}, {"queue", 0, 21}, {"queue", 3, 1}, {"status", 1, 0}, {"create", 0, 1}} {
			if _, _, err = p.Resolve(connector, bad.op, bad.id, bad.page); err == nil {
				t.Fatal(bad)
			}
		}
	}
}
func TestTransportGuards(t *testing.T) {
	for _, name := range []string{"origin", "traversal", "encoded", "backslash", "slash", "query-path", "method", "body-scope", "placeholder", "missing-id", "repeat-id", "unknown-route", "limit", "timeout", "query-injection"} {
		t.Run(name, func(t *testing.T) {
			p, _ := transport("radarr")
			r := p.Routes["find"]
			switch name {
			case "origin":
				p.Prefix = "https://evil.example"
			case "traversal":
				p.Prefix = "api/../private"
			case "encoded":
				p.Prefix = "api/%2e%2e/private"
			case "backslash":
				p.Prefix = "api\\private"
			case "slash":
				p.Prefix = "//elsewhere"
			case "query-path":
				r.Path = "movie?apikey=x"
			case "method":
				r.Method = "DELETE"
			case "body-scope":
				r.Method = "POST"
			case "placeholder":
				r.Path = "movie/{secret}"
			case "missing-id":
				r.Query = map[string]string{}
			case "repeat-id":
				r.Path = "movie/{id}"
			case "unknown-route":
				p.Routes["erase"] = a.Route{Method: "DELETE", Path: "all"}
			case "limit":
				p.MaxBytes = 9 << 20
			case "timeout":
				p.TimeoutSeconds = 60
			case "query-injection":
				r.Query["tmdbId"] = "{id}&apikey=x"
			}
			p.Routes["find"] = r
			if p.Validate("radarr") == nil {
				t.Fatal("invalid profile accepted")
			}
		})
	}
}
