package acquisition

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type TransportQuery struct {
	Connector string `json:"connector"`
}
type Route struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
}
type Transport struct {
	Version        int              `json:"version"`
	Prefix         string           `json:"prefix"`
	TimeoutSeconds int              `json:"timeoutSeconds"`
	MaxBytes       int64            `json:"maxBytes"`
	Routes         map[string]Route `json:"routes"`
}

func operations(connector string) map[string]string {
	r := map[string]string{"status": "GET", "profiles": "GET", "roots": "GET", "find": "GET", "lookup": "GET", "queue": "GET", "create": "POST", "update": "PUT", "search": "POST"}
	if connector == "sonarr" {
		r["episodes"], r["episodefiles"], r["monitor"] = "GET", "GET", "PUT"
	}
	return r
}
func needsID(op string) bool {
	return op == "find" || op == "lookup" || op == "update" || op == "episodes" || op == "episodefiles"
}
func safeSegment(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func safePath(s string) bool {
	if len(s) > 160 {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if !safeSegment(p) {
			return false
		}
	}
	return true
}
func (p Transport) Validate(connector string) error {
	bad := errors.New("invalid acquisition transport")
	if connector != "radarr" && connector != "sonarr" || p.Version != 1 || !safePath(p.Prefix) || p.TimeoutSeconds < 1 || p.TimeoutSeconds > 12 || p.MaxBytes < 1 || p.MaxBytes > 8<<20 {
		return bad
	}
	ops := operations(connector)
	if len(p.Routes) != len(ops) {
		return bad
	}
	for op, method := range ops {
		r, ok := p.Routes[op]
		if !ok || r.Method != method || len(r.Query) > 8 {
			return bad
		}
		// Only positive numeric placeholders can enter the path. Query values
		// are separately escaped by URL.Values; no arbitrary URL can be returned.
		if !safePath(strings.ReplaceAll(strings.ReplaceAll(r.Path, "{id}", "1"), "{page}", "1")) {
			return bad
		}
		idCount, pageCount := strings.Count(r.Path, "{id}"), strings.Count(r.Path, "{page}")
		for key, value := range r.Query {
			if !safeSegment(key) || len(key) > 64 || len(value) > 128 {
				return bad
			}
			idCount += strings.Count(value, "{id}")
			pageCount += strings.Count(value, "{page}")
			test := strings.ReplaceAll(strings.ReplaceAll(value, "{id}", "1"), "{page}", "1")
			for _, c := range test {
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == ':' || c == '_' || c == '-') {
					return bad
				}
			}
		}
		if needsID(op) {
			if idCount != 1 {
				return bad
			}
		} else if idCount != 0 {
			return bad
		}
		if op == "queue" {
			if pageCount != 1 {
				return bad
			}
		} else if pageCount != 0 {
			return bad
		}
	}
	return nil
}

// Resolve accepts semantic operation names and numeric identifiers only.
func (p Transport) Resolve(connector, operation string, id, page int) (method, relative string, err error) {
	if err = p.Validate(connector); err != nil {
		return
	}
	r, ok := p.Routes[operation]
	if !ok || needsID(operation) && id <= 0 || !needsID(operation) && id != 0 || operation == "queue" && (page < 1 || page > 20) || operation != "queue" && page != 0 {
		err = errors.New("invalid acquisition operation")
		return
	}
	render := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "{id}", strconv.Itoa(id)), "{page}", strconv.Itoa(page))
	}
	values := url.Values{}
	for k, v := range r.Query {
		values.Set(k, render(v))
	}
	relative = p.Prefix + "/" + render(r.Path)
	if len(values) > 0 {
		relative += "?" + values.Encode()
	}
	return r.Method, relative, nil
}
