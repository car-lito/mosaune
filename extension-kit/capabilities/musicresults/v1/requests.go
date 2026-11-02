package musicresults

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Request contains user intent, never a MusicBrainz query or URL.
type Request struct {
	Operation               string
	Title, Artist, ID, Date string
	Page                    int
}

func (q Request) Validate() error {
	bad := errors.New("invalid music request")
	if q.Page < 1 || q.Page > 50 || !text(q.Title, 1200) || !text(q.Artist, 1200) {
		return bad
	}
	switch q.Operation {
	case "discover", "match", "artists":
		if strings.TrimSpace(q.Title) == "" {
			return bad
		}
	case "release", "seed", "artist", "artist-releases", "upcoming":
		if !ID.MatchString(q.ID) {
			return bad
		}
	default:
		return bad
	}
	if q.Operation == "upcoming" {
		if _, err := time.Parse("2006-01-02", q.Date); err != nil {
			return bad
		}
	}
	return nil
}

// RequestPlan is executed against the host-owned MusicBrainz origin only.
// No headers, URL, credentials, HTTP method or arbitrary response decoder.
type RequestPlan struct {
	Path   string
	Params map[string]string
	Decode string
	ID     string
}

func (p RequestPlan) Validate(q Request) error {
	bad := errors.New("invalid music request plan")
	if q.Validate() != nil || len(p.Params) > 4 {
		return bad
	}
	expected := "releases"
	switch q.Operation {
	case "release", "seed", "artist", "artists":
		expected = q.Operation
	}
	if p.Decode != expected {
		return bad
	}
	if expected == "release" || expected == "seed" || expected == "artist" {
		resource := "release"
		if expected == "artist" {
			resource = "artist"
		}
		if p.ID != q.ID || p.Path != "/"+resource+"/"+q.ID {
			return bad
		}
	} else {
		resource := "release"
		if expected == "artists" {
			resource = "artist"
		}
		if p.ID != "" || p.Path != "/"+resource+"/" {
			return bad
		}
	}
	for k, v := range p.Params {
		if !text(v, 4096) || strings.ContainsAny(v, "\r\n") {
			return bad
		}
		switch k {
		case "query", "inc":
		case "limit":
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 20 {
				return bad
			}
		case "offset":
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 || n > 980 {
				return bad
			}
		default:
			return bad
		}
	}
	if expected == "releases" || expected == "artists" {
		if p.Params["query"] == "" || p.Params["limit"] == "" {
			return bad
		}
	}
	return nil
}
