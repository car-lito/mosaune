package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type tmdbProvider struct {
	state     providerLifecycle
	extension tmdbExtension
	db        *sql.DB
	client    *http.Client
	apiBase   string
	imageBase string
	mu        sync.Mutex
	next      time.Time
}

func newTMDBProvider(db *sql.DB) *tmdbProvider {
	return &tmdbProvider{extension: tmdbExtension{gate: make(chan struct{}, 1)}, db: db, apiBase: "https://api.themoviedb.org/3", imageBase: "https://image.tmdb.org/t/p/w500", client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection refusée") }}}
}
func (p *tmdbProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "tmdb", Name: "TMDB", ContractVersion: 1, Kinds: []string{"films", "series"}, RequiresCredential: true, NetworkHosts: []string{"api.themoviedb.org", "image.tmdb.org", "i.ytimg.com"}, Attribution: "This product uses the TMDB API but is not endorsed or certified by TMDB.", Website: "https://www.themoviedb.org"}
}
func (p *tmdbProvider) wait(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if delay := time.Until(p.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	p.next = time.Now().Add(300 * time.Millisecond)
	return nil
}
func (p *tmdbProvider) get(ctx context.Context, secret, endpoint string, params url.Values, out any) error {
	ctx, done, e := beginProvider(ctx, p)
	if e != nil {
		return e
	}
	defer done()
	if secret == "" {
		return errors.New("Configurez TMDB pour récupérer les fiches en ligne.")
	}
	if params == nil {
		params = url.Values{}
	}
	if !params.Has("language") {
		params.Set("language", appLanguage)
	}
	key := endpoint + "?" + params.Encode()
	var cached []byte
	if p.db != nil && p.db.QueryRowContext(ctx, `SELECT body FROM provider_cache WHERE provider='tmdb' AND key=? AND expires>?`, key, time.Now().Unix()).Scan(&cached) == nil {
		return json.Unmarshal(cached, out)
	}
	if err := p.wait(ctx); err != nil {
		return err
	}
	if len(secret) == 32 && !strings.Contains(secret, ".") {
		params.Set("api_key", secret)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.apiBase+endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return errors.New("Requête TMDB invalide.")
	}
	if params.Get("api_key") == "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mosaune/"+version)
	response, err := p.client.Do(req)
	if err != nil {
		return errors.New("TMDB est inaccessible. La bibliothèque locale reste disponible.")
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return errors.New("TMDB refuse cette clé. Vérifiez le jeton ou la clé API.")
	}
	if response.StatusCode == 429 {
		return errors.New("TMDB demande une pause. Réessayez plus tard.")
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("Fiche TMDB indisponible (HTTP %d).", response.StatusCode)
	}
	data, err := readBounded(response.Body, 2<<20)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return errors.New("Réponse TMDB illisible.")
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	if p.db != nil {
		unlock, e := providerCommit(ctx)
		if e != nil {
			return e
		}
		defer unlock()
		// Public metadata only: credentials are never part of the cache key or body.
		ttl := 7 * 24 * time.Hour
		if endpoint == "/movie/now_playing" || endpoint == "/movie/upcoming" || endpoint == "/tv/popular" {
			ttl = 6 * time.Hour
		}
		_, _ = p.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<? AND NOT (key LIKE 'visual-image:v2:%' AND length(body)>0)`, time.Now().Unix())
		_, _ = p.db.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT 'tmdb',?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+? < 16777216`, key, data, time.Now().Add(ttl).Unix(), len(data))
	}
	return nil
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
func (p *tmdbProvider) searchNative(ctx context.Context, secret string, q MetadataQuery) ([]MetadataCandidate, error) {
	params := url.Values{"query": {q.Title}, "include_adult": {"false"}}
	if q.Year > 0 {
		if q.Kind == "films" {
			params.Set("year", strconv.Itoa(q.Year))
		} else {
			params.Set("first_air_date_year", strconv.Itoa(q.Year))
		}
	}
	var response struct {
		Results []tmdbItem `json:"results"`
	}
	if err := p.get(ctx, secret, "/search/"+tmdbKind(q.Kind), params, &response); err != nil {
		return nil, err
	}
	result := []MetadataCandidate{}
	for _, item := range response.Results {
		original := item.OriginalTitle
		if original == "" {
			original = item.OriginalName
		}
		result = append(result, MetadataCandidate{ID: strconv.Itoa(item.ID), Title: truncate(item.title(), 500), OriginalTitle: truncate(original, 500), Year: item.year(), Overview: truncate(item.Overview, 600)})
		if len(result) == 20 {
			break
		}
	}
	return result, nil
}
func (p *tmdbProvider) detailsNative(ctx context.Context, secret string, s MetadataSelection) (Metadata, error) {
	id, err := strconv.Atoi(s.ID)
	if err != nil || id <= 0 || id > 2147483647 {
		return Metadata{}, errors.New("Identifiant TMDB invalide.")
	}
	var item tmdbItem
	endpoint := fmt.Sprintf("/%s/%d", tmdbKind(s.Kind), id)
	err = p.get(ctx, secret, endpoint, url.Values{"append_to_response": {"credits,videos"}, "include_video_language": {strings.Split(appLanguage, "-")[0] + ",en,null"}}, &item)
	if err != nil {
		return Metadata{}, err
	}
	m := Metadata{Title: truncate(item.title(), 500), Year: item.year(), Overview: truncate(item.Overview, 8000), Provider: "tmdb", ProviderID: s.ID, MatchState: "matched", ArtworkRef: item.Poster, Origins: map[string]string{"title": "tmdb", "year": "tmdb", "overview": "tmdb", "cast": "tmdb", "genres": "tmdb", "image": "tmdb"}}
	for _, g := range item.Genres {
		m.Genres = append(m.Genres, truncate(g.Name, 100))
		if len(m.Genres) == 12 {
			break
		}
	}
	for _, actor := range item.Credits.Cast {
		if actor.ID > 0 {
			m.Contributors = append(m.Contributors, Contributor{ID: strconv.Itoa(actor.ID), Name: truncate(actor.Name, 200), Role: truncate(actor.Character, 200), Image: assetURL("tmdb", actor.Portrait)})
		}
		m.Cast = append(m.Cast, truncate(actor.Name, 200))
		if len(m.Cast) == 10 {
			break
		}
	}
	m.Runtime = item.Runtime
	m.Tagline = truncate(item.Tagline, 500)
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
		m.Countries = append(m.Countries, truncate(country.Name, 100))
	}
	for _, language := range item.Languages {
		if len(m.Languages) >= 10 {
			break
		}
		m.Languages = append(m.Languages, truncate(language.Name, 100))
	}
	m.DetailVersion = 2
	for _, season := range item.Seasons {
		if len(m.Seasons) >= 100 {
			break
		}
		m.Seasons = append(m.Seasons, SeasonInfo{Number: season.Number, Name: truncate(season.Name, 200), Episodes: season.Episodes, Image: assetURL("tmdb", season.Poster)})
	}
	m.Trailers = selectTrailers(item.Videos.Results, strings.Split(appLanguage, "-")[0])
	if len(m.Trailers) == 0 {
		var fallback struct{ Results []tmdbVideo }
		if p.get(ctx, secret, endpoint+"/videos", url.Values{"language": {""}}, &fallback) == nil {
			m.Trailers = selectTrailers(fallback.Results, strings.Split(appLanguage, "-")[0])
		}
	}

	if s.Kind == "series" {
		m.Series = m.Title
		m.Season = s.Season
		m.Episode = s.Episode
		if s.Episode > 0 {
			var episode tmdbItem
			if err = p.get(ctx, secret, fmt.Sprintf("%s/season/%d/episode/%d", endpoint, s.Season, s.Episode), nil, &episode); err != nil {
				return Metadata{}, err
			}
			m.Title = truncate(episode.title(), 500)
			if episode.Overview != "" {
				m.Overview = truncate(episode.Overview, 8000)
			}
		}
	}
	return m, nil
}

var artworkPath = regexp.MustCompile(`^/[A-Za-z0-9_-]+\.(?:jpg|png)$`)

func (p *tmdbProvider) discoverNative(ctx context.Context, secret string, q DiscoveryQuery) (DiscoveryPage, error) {
	page := DiscoveryPage{Items: []Metadata{}, Page: q.Page}
	if !videoKind(q.Kind) || q.Page < 1 || q.Page > 50 {
		return page, errors.New("Recherche invalide.")
	}
	params := url.Values{"page": {strconv.Itoa(q.Page)}, "include_adult": {"false"}}
	endpoint := "/search/" + tmdbKind(q.Kind)
	if q.Seed != "" {
		id, err := strconv.Atoi(q.Seed)
		if err != nil || id <= 0 || id > 2147483647 {
			return page, errors.New("Identifiant TMDB invalide.")
		}
		endpoint = fmt.Sprintf("/%s/%d/recommendations", tmdbKind(q.Kind), id)
	} else {
		params.Set("query", q.Query)
	}
	var response struct {
		Results []tmdbItem `json:"results"`
		Pages   int        `json:"total_pages"`
	}
	if err := p.get(ctx, secret, endpoint, params, &response); err != nil {
		return page, err
	}
	page.Pages = max(1, min(50, response.Pages))
	for _, item := range response.Results {
		if item.ID <= 0 || item.title() == "" {
			continue
		}
		page.Items = append(page.Items, Metadata{Title: truncate(item.title(), 500), Year: item.year(), Overview: truncate(item.Overview, 8000), Provider: "tmdb", ProviderID: strconv.Itoa(item.ID), MatchState: "matched", ArtworkRef: item.Poster})
		if len(page.Items) == 20 {
			break
		}
	}
	return page, nil
}

func (p *tmdbProvider) activeArtwork(ctx context.Context, _ string, ref string) ([]byte, error) {
	if strings.HasPrefix(ref, "yt:") {
		key := strings.TrimPrefix(ref, "yt:")
		if !youtubeKey.MatchString(key) {
			return nil, errors.New("Miniature invalide.")
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://i.ytimg.com/vi/"+key+"/hqdefault.jpg", nil)
		response, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return nil, errors.New("Miniature indisponible.")
		}
		return readBounded(response.Body, 8<<20)
	}
	if !artworkPath.MatchString(ref) {
		return nil, errors.New("Image TMDB invalide.")
	}
	if err := p.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.imageBase+ref, nil)
	if err != nil {
		return nil, err
	}
	response, err := p.client.Do(req)
	if err != nil {
		return nil, errors.New("Affiche indisponible.")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New("Affiche indisponible.")
	}
	return readBounded(response.Body, 8<<20)
}

const appLanguage = "fr-FR" // Current interface language; shared with metadata and trailer selection.
type tmdbVideo struct {
	Name     string
	Key      string
	Site     string
	Type     string
	Language string `json:"iso_639_1"`
	Official bool
}

var frenchDub = regexp.MustCompile(`(?i)(\bVF\b|version fran[cç]aise)`)

func selectTrailers(videos []tmdbVideo, language string) []Trailer {
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
	out := []Trailer{}
	seen := map[string]bool{}
	for _, v := range valid {
		if seen[v.Key] {
			continue
		}
		seen[v.Key] = true
		out = append(out, Trailer{Name: truncate(v.Name, 200), URL: "https://www.youtube.com/watch?v=" + v.Key, Thumbnail: assetURL("tmdb", "yt:"+v.Key), Language: v.Language})
		if len(out) == 4 {
			break
		}
	}
	return out
}
