// Package v1 defines optional rich video metadata independently of the provider,
// catalog storage and user permissions. Image fields are opaque provider refs.
package v1

import (
	"errors"
	"math"
	presentation "mosaune/extensionkit/presentation/v1"
	"strings"
)

const DetailsCapability = "metadata.video-details.v1"

type Selection struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Season   int    `json:"season,omitempty"`
	Episode  int    `json:"episode,omitempty"`
	Language string `json:"language"`
}
type Contributor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
	Image string `json:"image,omitempty"`
}
type SeasonInfo struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Episodes int    `json:"episodes"`
	Image    string `json:"image,omitempty"`
}
type Trailer struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Language  string `json:"language,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
}
type Details struct {
	Presentation *presentation.Result `json:"presentation,omitempty"`
	Title        string               `json:"title"`
	Series       string               `json:"series,omitempty"`
	Season       int                  `json:"season,omitempty"`
	Episode      int                  `json:"episode,omitempty"`
	Year         int                  `json:"year,omitempty"`
	Overview     string               `json:"overview,omitempty"`
	ArtworkRef   string               `json:"artworkRef,omitempty"`
	Cast         []string             `json:"cast,omitempty"`
	Genres       []string             `json:"genres,omitempty"`
	Runtime      int                  `json:"runtime,omitempty"`
	Tagline      string               `json:"tagline,omitempty"`
	Rating       float64              `json:"rating,omitempty"`
	VoteCount    int                  `json:"voteCount,omitempty"`
	Status       string               `json:"status,omitempty"`
	ReleaseDate  string               `json:"releaseDate,omitempty"`
	Countries    []string             `json:"countries,omitempty"`
	Languages    []string             `json:"languages,omitempty"`
	Contributors []Contributor        `json:"contributors,omitempty"`
	Seasons      []SeasonInfo         `json:"seasons,omitempty"`
	Trailers     []Trailer            `json:"trailers,omitempty"`
}

func (s Selection) Validate() error {
	if strings.TrimSpace(s.ID) == "" || len(s.ID) > 160 || (s.Kind != "films" && s.Kind != "series") || s.Season < 0 || s.Season > 10000 || s.Episode < 0 || s.Episode > 100000 || len(s.Language) > 32 || s.Language == "" || (s.Kind == "films" && (s.Season != 0 || s.Episode != 0)) {
		return errors.New("invalid video selection")
	}
	return nil
}
func (d Details) Validate() error {
	if err := d.Presentation.Validate(); err != nil {
		return err
	}
	bad := errors.New("invalid video details")
	if strings.TrimSpace(d.Title) == "" || len(d.Title) > 500 || len(d.Series) > 500 || len(d.Overview) > 8000 || len(d.ArtworkRef) > 512 || d.Year < 0 || d.Year > 3000 || d.Runtime < 0 || d.Runtime > 100000 || d.Season < 0 || d.Season > 10000 || d.Episode < 0 || d.Episode > 100000 || len(d.Tagline) > 500 || len(d.Status) > 200 || len(d.ReleaseDate) > 32 || math.IsNaN(d.Rating) || math.IsInf(d.Rating, 0) || d.Rating < 0 || d.Rating > 10 || d.VoteCount < 0 || d.VoteCount > 2147483647 {
		return bad
	}
	for _, list := range [][]string{d.Cast, d.Genres, d.Countries, d.Languages} {
		if len(list) > 12 {
			return bad
		}
		for _, v := range list {
			if len(v) > 200 {
				return bad
			}
		}
	}
	if len(d.Contributors) > 10 || len(d.Seasons) > 100 || len(d.Trailers) > 4 {
		return bad
	}
	for _, c := range d.Contributors {
		if c.ID == "" || len(c.ID) > 160 || len(c.Name) > 200 || len(c.Role) > 200 || len(c.Image) > 512 {
			return bad
		}
	}
	for _, s := range d.Seasons {
		if s.Number < 0 || s.Number > 10000 || s.Episodes < 0 || s.Episodes > 100000 || len(s.Name) > 200 || len(s.Image) > 512 {
			return bad
		}
	}
	for _, v := range d.Trailers {
		if len(v.Name) > 200 || len(v.URL) > 2048 || len(v.Language) > 32 || len(v.Thumbnail) > 512 {
			return bad
		}
	}
	return nil
}
