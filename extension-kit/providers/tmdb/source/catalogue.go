package main

import (
	"encoding/json"
	catalog "mosaune/extensionkit/capabilities/videocatalog/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var catalogueImage = regexp.MustCompile(`^/[a-zA-Z0-9_-]+\.(jpg|png)$`)

type catalogueItem struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Name      string `json:"name"`
	Date      string `json:"release_date"`
	FirstAir  string `json:"first_air_date"`
	Overview  string `json:"overview"`
	Poster    string `json:"poster_path"`
	MediaType string `json:"media_type"`
}

func catalogueLight(i catalogueItem, kind string) catalog.Item {
	title := i.Title
	if title == "" {
		title = i.Name
	}
	date := i.Date
	if date == "" {
		date = i.FirstAir
	}
	year := 0
	if len(date) >= 4 {
		year, _ = strconv.Atoi(date[:4])
	}
	if year < 0 || year > 3000 {
		year = 0
	}
	ref := i.Poster
	if !catalogueImage.MatchString(ref) {
		ref = ""
	}
	return catalog.Item{ID: strconv.Itoa(i.ID), Kind: kind, Title: limit(strings.TrimSpace(title), 500), Year: year, Overview: limit(i.Overview, 8000), ArtworkRef: ref, ReleaseDate: limit(date, 40)}
}
func handleCatalogue(r protocol.Request) (any, *protocol.Failure) {
	fail := func(code string) (any, *protocol.Failure) {
		return nil, &protocol.Failure{Code: code, Message: "Catalogue TMDB indisponible."}
	}
	var q catalog.Query
	if protocol.DecodeOne(r.Payload, &q) != nil || q.Validate(r.Operation) != nil {
		return fail("invalid_query")
	}
	kind := "movie"
	if q.Kind == "series" {
		kind = "tv"
	}
	params := map[string]string{"language": q.Language}
	path := ""
	if q.ID != "" {
		n, e := strconv.Atoi(q.ID)
		if e != nil || n <= 0 || n > 2147483647 || strconv.Itoa(n) != q.ID {
			return fail("invalid_query")
		}
	}
	switch r.Operation {
	case "discover":
		params["page"] = strconv.Itoa(q.Page)
		params["include_adult"] = "false"
		path = "/search/" + kind
		if q.ID != "" {
			path = "/" + kind + "/" + q.ID + "/recommendations"
		} else {
			params["query"] = q.Text
		}
	case "shelf":
		if !(q.Kind == "films" && (q.Shelf == "now_playing" || q.Shelf == "upcoming") || q.Kind == "series" && q.Shelf == "popular") {
			return fail("invalid_query")
		}
		path = "/" + kind + "/" + q.Shelf
		params["region"] = q.Region
		params["page"] = "1"
	case "profile":
		path = "/person/" + q.ID
		params["append_to_response"] = "combined_credits"
	case "posters":
		path = "/" + kind + "/" + q.ID + "/images"
		params["language"] = ""
	}
	raw, err := sdk.ReadService("tmdb.api.v1", "get", map[string]any{"path": path, "query": params})
	if err != nil {
		return fail("provider_unavailable")
	}
	out := catalog.Result{Items: []catalog.Item{}, Page: q.Page, Pages: 1}
	var candidates []catalogueItem
	if r.Operation == "posters" {
		var response struct {
			Posters []struct {
				Path     string `json:"file_path"`
				Language string `json:"iso_639_1"`
				Width    int
				Height   int
				Votes    int `json:"vote_count"`
			}
		}
		if json.Unmarshal(raw, &response) != nil {
			return fail("invalid_provider_reply")
		}
		preferred := strings.Split(q.Language, "-")[0]
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
		seen := map[string]bool{}
		for _, p := range response.Posters {
			if seen[p.Path] || !catalogueImage.MatchString(p.Path) || p.Width <= 0 || p.Height <= 0 || p.Width > 100000 || p.Height > 100000 || len(p.Language) > 35 {
				continue
			}
			seen[p.Path] = true
			out.Posters = append(out.Posters, catalog.Poster{Ref: p.Path, Language: p.Language, Width: p.Width, Height: p.Height})
			if len(out.Posters) == 100 {
				break
			}
		}
	} else if r.Operation == "profile" {
		var response struct {
			Name      string
			Portrait  string `json:"profile_path"`
			Biography string
			Birthday  string
			Place     string                         `json:"place_of_birth"`
			Credits   struct{ Cast []catalogueItem } `json:"combined_credits"`
		}
		if json.Unmarshal(raw, &response) != nil {
			return fail("invalid_provider_reply")
		}
		ref := response.Portrait
		if !catalogueImage.MatchString(ref) {
			ref = ""
		}
		out.Person = &catalog.Person{ID: q.ID, Name: limit(strings.TrimSpace(response.Name), 300), Biography: limit(response.Biography, 12000), Detail: limit(strings.Trim(response.Birthday+" · "+response.Place, " ·"), 600), ArtworkRef: ref}
		candidates = response.Credits.Cast
	} else {
		var response struct {
			Results []catalogueItem
			Pages   int `json:"total_pages"`
		}
		if json.Unmarshal(raw, &response) != nil {
			return fail("invalid_provider_reply")
		}
		candidates = response.Results
		if r.Operation == "discover" {
			out.Pages = max(1, min(50, response.Pages))
		}
	}
	seen := map[string]bool{}
	all := []catalog.Item{}
	for _, i := range candidates {
		k := q.Kind
		if r.Operation == "profile" {
			if i.MediaType == "movie" {
				k = "films"
			} else if i.MediaType == "tv" {
				k = "series"
			} else {
				continue
			}
		}
		item := catalogueLight(i, k)
		key := k + ":" + item.ID
		if i.ID <= 0 || i.ID > 2147483647 || item.Title == "" || seen[key] {
			continue
		}
		seen[key] = true
		// Search results historically omit release date. Keep the host document stable.
		if r.Operation == "discover" {
			item.ReleaseDate = ""
		}
		all = append(all, item)
		if r.Operation != "profile" && len(all) == 20 || len(all) == 1000 {
			break
		}
	}
	if r.Operation == "profile" {
		out.Pages = max(1, (len(all)+19)/20)
		start := (q.Page - 1) * 20
		if start < len(all) {
			out.Items = all[start:min(start+20, len(all))]
		}
	} else {
		out.Items = all
	}
	if out.Validate(r.Operation, q) != nil {
		return fail("invalid_result")
	}
	return out, nil
}
