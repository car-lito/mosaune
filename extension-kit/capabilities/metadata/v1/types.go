// Package v1 defines the first metadata capabilities independently of the
// transport and runtime. Optional future capabilities do not alter this schema.
package v1

import (
	"errors"
	presentation "mosaune/extensionkit/presentation/v1"
	"strings"
)

const SearchCapability = "metadata.search.v1"
const DetailsCapability = "metadata.details.v1"

type Query struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Year     int    `json:"year,omitempty"`
	Language string `json:"language,omitempty"`
}
type Selection struct {
	ID string `json:"id"`
}
type Candidate struct {
	Presentation  *presentation.Result `json:"presentation,omitempty"`
	OriginalTitle string               `json:"originalTitle,omitempty"`
	ID            string               `json:"id"`
	Title         string               `json:"title"`
	Year          int                  `json:"year"`
	Overview      string               `json:"overview"`
}
type SearchResult struct {
	Items []Candidate `json:"items"`
}

func (q Query) Validate() error {
	if len(q.Kind) < 1 || len(q.Kind) > 64 || len(q.Title) > 250 || q.Year < 0 || q.Year > 3000 || len(q.Language) > 32 {
		return errors.New("invalid metadata query")
	}
	return nil
}
func (c Candidate) Validate() error {
	if err := c.Presentation.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.ID) == "" || len(c.ID) > 160 || strings.TrimSpace(c.Title) == "" || len(c.Title) > 500 || len(c.OriginalTitle) > 500 || c.Year < 0 || c.Year > 3000 || len(c.Overview) > 16000 {
		return errors.New("invalid metadata candidate")
	}
	return nil
}
func (r SearchResult) Validate() error {
	if r.Items == nil || len(r.Items) > 50 {
		return errors.New("invalid metadata result count")
	}
	seen := map[string]bool{}
	for _, c := range r.Items {
		if err := c.Validate(); err != nil {
			return err
		}
		if seen[c.ID] {
			return errors.New("duplicate metadata identifier")
		}
		seen[c.ID] = true
	}
	return nil
}
