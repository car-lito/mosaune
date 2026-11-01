package catalogue

import (
	"errors"
	presentation "mosaune/extensionkit/presentation/v1"
	"strings"
)

type SearchEntry struct {
	MetadataPresentation bool                 `json:"metadataPresentation,omitempty"`
	Target               string               `json:"target"` // group or item; host binds to the current input item only
	Title                string               `json:"title"`
	Subtitle             string               `json:"subtitle"`
	Terms                []string             `json:"terms"`
	Priority             int                  `json:"priority"`
	Presentation         *presentation.Result `json:"presentation,omitempty"`
}
type SearchIndex struct {
	Version int           `json:"version"`
	Entries []SearchEntry `json:"entries"`
}

func (s *SearchIndex) Validate() error {
	if s == nil {
		return nil
	} // legacy projection: generic filename lookup only
	bad := errors.New("invalid catalogue search index")
	if s.Version != 1 || len(s.Entries) > 4 {
		return bad
	}
	seen := map[string]bool{}
	for _, e := range s.Entries {
		if (e.Target != "group" && e.Target != "item") || seen[e.Target] || e.Priority < 0 || e.Priority > 100 || strings.TrimSpace(e.Title) == "" || !text(e.Title) || !text(e.Subtitle) || len(e.Terms) > 8 {
			return bad
		}
		seen[e.Target] = true
		for _, v := range e.Terms {
			if !text(v) {
				return bad
			}
		}
		if e.Presentation == nil || e.Presentation.Validate() != nil {
			return bad
		}
	}
	return nil
}
