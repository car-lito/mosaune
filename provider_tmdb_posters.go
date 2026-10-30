package main

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func (p *tmdbProvider) postersNative(ctx context.Context, secret string, s MetadataSelection, language string) ([]PosterOption, error) {
	id, e := strconv.Atoi(s.ID)
	if e != nil || id <= 0 || !videoKind(s.Kind) {
		return nil, errors.New("Œuvre invalide.")
	}
	resource := "movie"
	if s.Kind == "series" {
		resource = "tv"
	}
	var response struct {
		Posters []struct {
			Path     string `json:"file_path"`
			Language string `json:"iso_639_1"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Votes    int    `json:"vote_count"`
		}
	}
	// Explicit empty language disables the default filter: keep alternatives in all languages.
	if e = p.get(ctx, secret, "/"+resource+"/"+strconv.Itoa(id)+"/images", url.Values{"language": {""}}, &response); e != nil {
		return nil, e
	}
	preferred := strings.Split(language, "-")[0]
	rank := func(s string) int {
		if s == preferred {
			return 0
		}
		if s == "" {
			return 1
		}
		return 2
	}
	sort.SliceStable(response.Posters, func(i, j int) bool {
		a, b := response.Posters[i], response.Posters[j]
		if rank(a.Language) != rank(b.Language) {
			return rank(a.Language) < rank(b.Language)
		}
		return a.Votes > b.Votes
	})
	out := []PosterOption{}
	seen := map[string]bool{}
	for _, p := range response.Posters {
		if seen[p.Path] || !artworkPath.MatchString(p.Path) || p.Width <= 0 || p.Height <= 0 {
			continue
		}
		seen[p.Path] = true
		out = append(out, PosterOption{Ref: p.Path, Language: p.Language, Width: p.Width, Height: p.Height})
		if len(out) == 100 {
			break
		}
	}
	return out, nil
}
