package musicresults

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// State is opaque to the host so an implementation can change its workflow.
// Each call receives at most one bounded JSON reply, never image bytes.
type VisualQuery struct {
	Operation, Ref string
	State          json.RawMessage
	Status         int
	Body           json.RawMessage
}
type VisualFetch struct {
	Source, Path string
	Params       map[string]string
	Image        bool
}
type Photo struct{ File, URL, Source, Credit, License string }
type VisualStep struct {
	Fetch   *VisualFetch
	Photo   *Photo
	Missing bool
	State   json.RawMessage
}

var EntityID = regexp.MustCompile(`^Q[1-9][0-9]{0,18}$`)
var coverPath = regexp.MustCompile(`^/(release|release-group)/[0-9a-f-]{36}/front-(250|500|1200)$`)

func SafeCommonsFile(file string) bool {
	low := strings.ToLower(file)
	return text(file, 500) && len(file) > 0 && !strings.ContainsAny(file, "/\\\r\n") && (strings.HasSuffix(low, ".jpg") || strings.HasSuffix(low, ".jpeg") || strings.HasSuffix(low, ".png"))
}
func SafeUpload(raw string) bool {
	u, e := url.Parse(raw)
	return text(raw, 3000) && e == nil && u.Scheme == "https" && u.User == nil && (u.Host == "upload.wikimedia.org" || u.Host == "thumb.wikimedia.org") && u.Fragment == ""
}
func (q VisualQuery) Validate() error {
	bad := errors.New("invalid visual query")
	if len(q.State) > 8192 || len(q.State) > 0 && !json.Valid(q.State) || len(q.Body) > MaxBody || len(q.Body) > 0 && !json.Valid(q.Body) || (q.Status != 0 && q.Status != 200 && q.Status != 404) {
		return bad
	}
	switch q.Operation {
	case "artist":
		if !ID.MatchString(q.Ref) {
			return bad
		}
	case "artwork":
		if strings.HasPrefix(q.Ref, "commons:") {
			if !SafeCommonsFile(strings.TrimPrefix(q.Ref, "commons:")) {
				return bad
			}
		} else {
			parts := strings.Split(q.Ref, "/")
			if len(parts) > 2 || !ID.MatchString(parts[0]) || len(parts) == 2 && !ID.MatchString(parts[1]) {
				return bad
			}
		}
	default:
		return bad
	}
	return nil
}
func (p Photo) Validate() error {
	if !SafeCommonsFile(p.File) || !SafeUpload(p.URL) || p.Source != "https://commons.wikimedia.org/wiki/File:"+url.PathEscape(p.File) || !text(p.Credit, 2000) || !text(p.License, 2000) {
		return errors.New("invalid visual attribution")
	}
	return nil
}
func (f VisualFetch) Validate() error {
	bad := errors.New("invalid visual fetch")
	if len(f.Params) > 8 {
		return bad
	}
	for k, v := range f.Params {
		if !text(k, 40) || !text(v, 1000) || strings.ContainsAny(v, "\r\n") {
			return bad
		}
	}
	switch f.Source {
	case "musicbrainz":
		parts := strings.Split(f.Path, "/")
		if f.Image || len(parts) != 3 || parts[0] != "" || !ID.MatchString(parts[2]) || len(f.Params) != 1 {
			return bad
		}
		if !(parts[1] == "artist" && f.Params["inc"] == "url-rels" || parts[1] == "release" && f.Params["inc"] == "release-groups") {
			return bad
		}
	case "wikidata":
		if f.Image || f.Path != "/w/api.php" || len(f.Params) != 4 || f.Params["action"] != "wbgetclaims" || f.Params["format"] != "json" || f.Params["property"] != "P18" || !EntityID.MatchString(f.Params["entity"]) {
			return bad
		}
	case "commons":
		if f.Image || f.Path != "/w/api.php" || len(f.Params) != 8 || f.Params["action"] != "query" || f.Params["format"] != "json" || f.Params["prop"] != "imageinfo" || !strings.HasPrefix(f.Params["titles"], "File:") || !SafeCommonsFile(strings.TrimPrefix(f.Params["titles"], "File:")) || f.Params["iiprop"] != "url|extmetadata" || f.Params["iiextmetadatafilter"] != "Artist|LicenseShortName" || f.Params["iiurlwidth"] != "320" || f.Params["redirects"] != "1" {
			return bad
		}
	case "coverart":
		if !f.Image || len(f.Params) != 0 || !coverPath.MatchString(f.Path) || !ID.MatchString(strings.Split(f.Path, "/")[2]) {
			return bad
		}
	case "image":
		if !f.Image || len(f.Params) != 0 || !SafeUpload(f.Path) {
			return bad
		}
	default:
		return bad
	}
	return nil
}
func (s VisualStep) Validate(q VisualQuery) error {
	bad := errors.New("invalid visual step")
	if q.Validate() != nil || len(s.State) > 8192 || len(s.State) > 0 && !json.Valid(s.State) {
		return bad
	}
	n := 0
	if s.Missing {
		n++
	}
	if s.Photo != nil {
		n++
		if q.Operation != "artist" || s.Photo.Validate() != nil {
			return bad
		}
	}
	if s.Fetch != nil {
		n++
		if s.Fetch.Validate() != nil || q.Operation == "artist" && s.Fetch.Image {
			return bad
		}
	}
	if n != 1 {
		return bad
	}
	return nil
}
