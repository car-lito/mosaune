package main

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var youtubeKey = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func tmdbLight(item tmdbItem) Metadata {
	return Metadata{Title: truncate(item.title(), 500), Year: item.year(), Overview: truncate(item.Overview, 8000), Provider: "tmdb", ProviderID: strconv.Itoa(item.ID), ArtworkRef: item.Poster, ReleaseDate: item.ReleaseDate, MatchState: "matched"}
}
func (p *tmdbProvider) Shelf(ctx context.Context, secret, shelf, kind string) (DiscoveryPage, error) {
	page := DiscoveryPage{Items: []Metadata{}, Page: 1, Pages: 1}
	endpoint := ""
	if kind == "films" && (shelf == "now_playing" || shelf == "upcoming") {
		endpoint = "/movie/" + shelf
	} else if kind == "series" && shelf == "popular" {
		endpoint = "/tv/popular"
	} else {
		return page, errors.New("Sélection inconnue.")
	}
	var response struct{ Results []tmdbItem }
	if err := p.get(ctx, secret, endpoint, url.Values{"region": {"BE"}, "page": {"1"}}, &response); err != nil {
		return page, err
	}
	for _, item := range response.Results {
		if len(page.Items) >= 20 {
			break
		}
		if item.ID > 0 && item.title() != "" {
			page.Items = append(page.Items, tmdbLight(item))
		}
	}
	return page, nil
}
func (p *tmdbProvider) Profile(ctx context.Context, secret, id string, page int) (PersonProfile, error) {
	out := PersonProfile{ID: id, Page: page, Works: []discoveryWork{}}
	number, err := strconv.Atoi(id)
	if err != nil || number <= 0 || number > 2147483647 {
		return out, errors.New("Profil invalide.")
	}
	var response struct {
		Name      string
		Portrait  string `json:"profile_path"`
		Biography string
		Birthday  string
		Place     string `json:"place_of_birth"`
		Credits   struct {
			Cast []tmdbItem `json:"cast"`
		} `json:"combined_credits"`
	}
	if err = p.get(ctx, secret, "/person/"+id, url.Values{"append_to_response": {"combined_credits"}}, &response); err != nil {
		return out, err
	}
	out.Image = assetURL("tmdb", response.Portrait)
	out.Name = truncate(response.Name, 300)
	out.Biography = truncate(response.Biography, 12000)
	out.Detail = strings.Trim(response.Birthday+" · "+response.Place, " ·")
	all := []ProfileItem{}
	seen := map[string]bool{}
	for _, item := range response.Credits.Cast {
		kind := "films"
		if item.MediaType == "tv" {
			kind = "series"
		} else if item.MediaType != "movie" {
			continue
		}
		key := kind + strconv.Itoa(item.ID)
		if seen[key] || item.ID <= 0 || item.title() == "" {
			continue
		}
		seen[key] = true
		all = append(all, ProfileItem{Kind: kind, Metadata: tmdbLight(item)})
	}
	out.Pages = max(1, min(50, (len(all)+19)/20))
	start := (page - 1) * 20
	if start < len(all) {
		out.Items = all[start:min(start+20, len(all))]
	}
	return out, nil
}
