// Package transport describes host-executed requests. A profile is data, not a
// network grant: the installer must check every origin against its own consent.
package transport

import (
	"errors"
	"mosaune/extensionkit/configuration"
	"net/url"
	"regexp"
	"strings"
)

type Source struct {
	ID           string         `json:"id"`
	BaseURL      string         `json:"baseURL"`
	Format       string         `json:"format"`
	MaxBytes     int64          `json:"maxBytes"`
	TimeoutMS    int            `json:"timeoutMs"`
	IntervalMS   int            `json:"intervalMs"`
	CacheSeconds int            `json:"cacheSeconds"`
	CachePaths   map[string]int `json:"cachePaths,omitempty"`
	Credential   string         `json:"credential,omitempty"`
	Auth         string         `json:"auth,omitempty"`
	QueryKey     string         `json:"queryKey,omitempty"`
}
type Profile struct {
	Version         int      `json:"version"`
	Language        string   `json:"language"`
	Region          string   `json:"region"`
	LanguageSetting string   `json:"languageSetting,omitempty"`
	RegionSetting   string   `json:"regionSetting,omitempty"`
	Sources         []Source `json:"sources"`
}

var name = regexp.MustCompile(`^[a-z][a-zA-Z0-9_-]{0,47}$`)
var language = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)
var region = regexp.MustCompile(`^[A-Z]{2}$`)

func (p Profile) Validate(settings configuration.Schema) error {
	bad := errors.New("invalid provider transport profile")
	if p.Version != 1 || !language.MatchString(p.Language) || !region.MatchString(p.Region) || len(p.Sources) < 1 || len(p.Sources) > 12 {
		return bad
	}
	fields := map[string]configuration.Field{}
	for _, f := range settings.Fields {
		fields[f.ID] = f
	}
	for _, binding := range []struct {
		id      string
		pattern *regexp.Regexp
	}{{p.LanguageSetting, language}, {p.RegionSetting, region}} {
		if binding.id == "" {
			continue
		}
		f, ok := fields[binding.id]
		if !ok || f.Type != "select" {
			return bad
		}
		for _, o := range f.Options {
			if !binding.pattern.MatchString(o.Value) {
				return bad
			}
		}
	}
	seen := map[string]bool{}
	for _, s := range p.Sources {
		u, e := url.Parse(s.BaseURL)
		if !name.MatchString(s.ID) || seen[s.ID] || e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" || len(s.BaseURL) > 512 || strings.HasSuffix(s.BaseURL, "/") || !safePath(u.Path) || s.TimeoutMS < 100 || s.TimeoutMS > 20000 || s.IntervalMS < 0 || s.IntervalMS > 5000 || s.CacheSeconds < 0 || s.CacheSeconds > 604800 || len(s.CachePaths) > 32 {
			return bad
		}
		seen[s.ID] = true
		limit := int64(2 << 20)
		if s.Format == "image" {
			limit = 8 << 20
		} else if s.Format != "json" {
			return bad
		}
		if s.MaxBytes < 1 || s.MaxBytes > limit {
			return bad
		}
		for path, ttl := range s.CachePaths {
			if path == "" || !strings.HasPrefix(path, "/") || !safePath(path) || ttl < 0 || ttl > 604800 {
				return bad
			}
		}
		if s.Auth == "" {
			if s.Credential != "" || s.QueryKey != "" {
				return bad
			}
		} else {
			f, ok := fields[s.Credential]
			if !ok || f.Type != "secret" || s.Format != "json" {
				return bad
			}
			if s.Auth != "bearer" && s.Auth != "query" && s.Auth != "key-or-bearer" {
				return bad
			}
			if s.Auth == "bearer" {
				if s.QueryKey != "" {
					return bad
				}
			} else if !name.MatchString(s.QueryKey) {
				return bad
			}
		}
	}
	return nil
}
func safePath(path string) bool {
	if strings.ContainsAny(path, "\\%?#\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}
func (p Profile) Source(id string) (Source, error) {
	for _, s := range p.Sources {
		if s.ID == id {
			return s, nil
		}
	}
	return Source{}, errors.New("undeclared provider source")
}
func (s Source) URL(path string) (string, error) {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || !safePath(path) {
		return "", errors.New("invalid provider path")
	}
	return s.BaseURL + path, nil
}
func (s Source) TTL(path string) int {
	if v, ok := s.CachePaths[path]; ok {
		return v
	}
	return s.CacheSeconds
}
