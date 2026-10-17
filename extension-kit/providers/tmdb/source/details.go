package main

import (
	"encoding/json"
	"errors"
	"fmt"
	video "mosaune/extensionkit/capabilities/video/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var youtubeKey = regexp.MustCompile("^[a-zA-Z0-9_-]{11}$")
var imagePath = regexp.MustCompile("^/[A-Za-z0-9_-]+\\.(?:jpg|png)$")

func imageRef(ref string) string {
	if imagePath.MatchString(ref) || (strings.HasPrefix(ref, "yt:") && youtubeKey.MatchString(strings.TrimPrefix(ref, "yt:"))) {
		return ref
	}
	return ""
}
func handleVideo(r protocol.Request) (any, *protocol.Failure) {
	var s video.Selection
	if r.Operation != "details" || protocol.DecodeOne(r.Payload, &s) != nil || s.Validate() != nil {
		return nil, &protocol.Failure{Code: "invalid_selection", Message: "Sélection vidéo invalide."}
	}
	result, err := details(s)
	if err != nil {
		return nil, &protocol.Failure{Code: "provider_unavailable", Message: "Fiche TMDB indisponible."}
	}
	if result.Validate() != nil {
		return nil, &protocol.Failure{Code: "invalid_details", Message: "Fiche TMDB invalide."}
	}
	return result, nil
}

type tmdbItem struct {
	Runtime   int     `json:"runtime"`
	Tagline   string  `json:"tagline"`
	Rating    float64 `json:"vote_average"`
	VoteCount int     `json:"vote_count"`
	Status    string  `json:"status"`
	MediaType string  `json:"media_type"`
	Countries []struct {
		Name string `json:"name"`
	} `json:"production_countries"`
	Languages []struct {
		Name string `json:"english_name"`
	} `json:"spoken_languages"`
	Videos  struct{ Results []tmdbVideo } `json:"videos"`
	Seasons []struct {
		Number   int `json:"season_number"`
		Name     string
		Poster   string `json:"poster_path"`
		Episodes int    `json:"episode_count"`
	} `json:"seasons"`
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
	Overview      string `json:"overview"`
	Poster        string `json:"poster_path"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Credits struct {
		Cast []struct {
			ID        int    `json:"id"`
			Character string `json:"character"`
			Portrait  string `json:"profile_path"`
			Name      string `json:"name"`
		} `json:"cast"`
	} `json:"credits"`
}

func (i tmdbItem) title() string {
	if i.Title != "" {
		return i.Title
	}
	return i.Name
}
func (i tmdbItem) year() int {
	date := i.ReleaseDate
	if date == "" {
		date = i.FirstAirDate
	}
	if len(date) < 4 {
		return 0
	}
	v, _ := strconv.Atoi(date[:4])
	return v
}
func tmdbKind(kind string) string {
	if kind == "series" {
		return "tv"
	}
	return "movie"
}
func details(s video.Selection) (video.Details, error) {
	get := func(endpoint string, params url.Values, out any) error {
		if params == nil {
			params = url.Values{}
		}
		if !params.Has("language") {
			params.Set("language", s.Language)
		}
		query := map[string]string{}
		for k, v := range params {
			if len(v) == 1 {
				query[k] = v[0]
			}
		}
		raw, err := sdk.ReadService("tmdb.api.v1", "get", map[string]any{"path": endpoint, "query": query})
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, out)
	}
	id, err := strconv.Atoi(s.ID)
	if err != nil || id <= 0 || id > 2147483647 {
		return video.Details{}, errors.New("Identifiant TMDB invalide.")
	}
	var item tmdbItem
	endpoint := fmt.Sprintf("/%s/%d", tmdbKind(s.Kind), id)
	err = get(endpoint, url.Values{"append_to_response": {"credits,videos"}, "include_video_language": {strings.Split(s.Language, "-")[0] + ",en,null"}}, &item)
	if err != nil {
		return video.Details{}, err
	}
	m := video.Details{Title: limit(item.title(), 500), Year: item.year(), Overview: limit(item.Overview, 8000), ArtworkRef: item.Poster}
	for _, g := range item.Genres {
		m.Genres = append(m.Genres, limit(g.Name, 100))
		if len(m.Genres) == 12 {
			break
		}
	}
	for _, actor := range item.Credits.Cast {
		if actor.ID > 0 {
			m.Contributors = append(m.Contributors, video.Contributor{ID: strconv.Itoa(actor.ID), Name: limit(actor.Name, 200), Role: limit(actor.Character, 200), Image: imageRef(actor.Portrait)})
		}
		m.Cast = append(m.Cast, limit(actor.Name, 200))
		if len(m.Cast) == 10 {
			break
		}
	}
	m.Runtime = item.Runtime
	m.Tagline = limit(item.Tagline, 500)
	m.Rating = item.Rating
	m.VoteCount = item.VoteCount
	m.Status = item.Status
	m.ReleaseDate = item.ReleaseDate
	if m.ReleaseDate == "" {
		m.ReleaseDate = item.FirstAirDate
	}
	for _, country := range item.Countries {
		if len(m.Countries) >= 10 {
			break
		}
		m.Countries = append(m.Countries, limit(country.Name, 100))
	}
	for _, language := range item.Languages {
		if len(m.Languages) >= 10 {
			break
		}
		m.Languages = append(m.Languages, limit(language.Name, 100))
	}

	for _, season := range item.Seasons {
		if len(m.Seasons) >= 100 {
			break
		}
		m.Seasons = append(m.Seasons, video.SeasonInfo{Number: season.Number, Name: limit(season.Name, 200), Episodes: season.Episodes, Image: imageRef(season.Poster)})
	}
	m.Trailers = selectTrailers(item.Videos.Results, strings.Split(s.Language, "-")[0])
	if len(m.Trailers) == 0 {
		var fallback struct{ Results []tmdbVideo }
		if get(endpoint+"/videos", url.Values{"language": {""}}, &fallback) == nil {
			m.Trailers = selectTrailers(fallback.Results, strings.Split(s.Language, "-")[0])
		}
	}

	if s.Kind == "series" {
		m.Series = m.Title
		m.Season = s.Season
		m.Episode = s.Episode
		if s.Episode > 0 {
			var episode tmdbItem
			if err = get(fmt.Sprintf("%s/season/%d/episode/%d", endpoint, s.Season, s.Episode), nil, &episode); err != nil {
				return video.Details{}, err
			}
			m.Title = limit(episode.title(), 500)
			if episode.Overview != "" {
				m.Overview = limit(episode.Overview, 8000)
			}
		}
	}
	return m, nil
}

type tmdbVideo struct {
	Name     string
	Key      string
	Site     string
	Type     string
	Language string `json:"iso_639_1"`
	Official bool
}

var frenchDub = regexp.MustCompile(`(?i)(\bVF\b|version fran[cç]aise)`)

func selectTrailers(videos []tmdbVideo, language string) []video.Trailer {
	valid := []tmdbVideo{}
	for _, v := range videos {
		if v.Site == "YouTube" && v.Type == "Trailer" && youtubeKey.MatchString(v.Key) {
			valid = append(valid, v)
		}
	}
	sort.SliceStable(valid, func(i, j int) bool {
		a, b := valid[i], valid[j]
		if (a.Language == language) != (b.Language == language) {
			return a.Language == language
		}
		if language == "fr" && frenchDub.MatchString(a.Name) != frenchDub.MatchString(b.Name) {
			return frenchDub.MatchString(a.Name)
		}
		if a.Official != b.Official {
			return a.Official
		}
		return false
	})
	out := []video.Trailer{}
	seen := map[string]bool{}
	for _, v := range valid {
		if seen[v.Key] {
			continue
		}
		seen[v.Key] = true
		out = append(out, video.Trailer{Name: limit(v.Name, 200), URL: "https://www.youtube.com/watch?v=" + v.Key, Thumbnail: imageRef("yt:" + v.Key), Language: v.Language})
		if len(out) == 4 {
			break
		}
	}
	return out
}
