// A separate WASI program: no database, HTTP client, credentials or host paths.
package main

import (
	"encoding/json"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	video "mosaune/extensionkit/capabilities/video/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Overridden when independently building a provider release.
var buildVersion = "0.4.0"

func limit(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
func handle(r protocol.Request) (any, *protocol.Failure) {
	if r.Capability == "system.health.v1" && r.Operation == "check" {
		return map[string]string{"version": buildVersion, "provider": "tmdb"}, nil
	}
	if r.Capability == video.DetailsCapability {
		return handleVideo(r)
	}
	fail := func(code string) (any, *protocol.Failure) {
		return nil, &protocol.Failure{Code: code, Message: "Recherche TMDB indisponible."}
	}
	if r.Capability != metadata.SearchCapability || r.Operation != "search" {
		return fail("unsupported_operation")
	}
	var q metadata.Query
	if protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil || (q.Kind != "films" && q.Kind != "series") {
		return fail("invalid_query")
	}
	path := "/search/movie"
	params := map[string]string{"query": q.Title, "include_adult": "false"}
	if q.Language != "" {
		params["language"] = q.Language
	}
	if q.Kind == "series" {
		path = "/search/tv"
	}
	if q.Year > 0 {
		key := "year"
		if q.Kind == "series" {
			key = "first_air_date_year"
		}
		params[key] = strconv.Itoa(q.Year)
	}
	data, err := sdk.ReadService("tmdb.api.v1", "get", map[string]any{"path": path, "query": params})
	if err != nil {
		return fail("provider_unavailable")
	}
	var response struct {
		Results []struct {
			ID            int    `json:"id"`
			Title         string `json:"title"`
			Name          string `json:"name"`
			OriginalTitle string `json:"original_title"`
			OriginalName  string `json:"original_name"`
			ReleaseDate   string `json:"release_date"`
			FirstAirDate  string `json:"first_air_date"`
			Overview      string `json:"overview"`
		} `json:"results"`
	}
	if json.Unmarshal(data, &response) != nil {
		return fail("invalid_provider_reply")
	}
	result := metadata.SearchResult{Items: []metadata.Candidate{}}
	seen := map[int]bool{}
	for _, item := range response.Results {
		title := item.Title
		if title == "" {
			title = item.Name
		}
		if item.ID <= 0 || item.ID > 2147483647 || seen[item.ID] || strings.TrimSpace(title) == "" {
			continue
		}
		seen[item.ID] = true
		original := item.OriginalTitle
		if original == "" {
			original = item.OriginalName
		}
		date := item.ReleaseDate
		if date == "" {
			date = item.FirstAirDate
		}
		year := 0
		if len(date) >= 4 {
			year, _ = strconv.Atoi(date[:4])
		}
		if year > 3000 {
			year = 0
		}
		result.Items = append(result.Items, metadata.Candidate{ID: strconv.Itoa(item.ID), Title: limit(title, 500), OriginalTitle: limit(original, 500), Year: year, Overview: limit(item.Overview, 600)})
		if len(result.Items) == 20 {
			break
		}
	}
	return result, nil
}
func main() { _ = sdk.Serve(os.Stdin, os.Stdout, handle) }
