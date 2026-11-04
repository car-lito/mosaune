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
	"strings"
	"time"
)

type tmdbProvider struct {
	discoveryCache tmdbDiscoveryCache
	state          providerLifecycle
	extension      tmdbExtension
	db             *sql.DB
	client         *http.Client
	apiBase        string
	imageBase      string
	rate           providerRateGate
}

func newTMDBProvider(db *sql.DB) *tmdbProvider {
	installed := false
	if db != nil {
		_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM provider_packages WHERE id='tmdb')").Scan(&installed)
	}
	return &tmdbProvider{state: providerLifecycle{moduleUnavailable: !installed}, extension: tmdbExtension{installed: installed, enabled: installed, gate: make(chan struct{}, 1)}, db: db, apiBase: "https://api.themoviedb.org/3", imageBase: "https://image.tmdb.org/t/p/w500", client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection refusée") }}}
}
func (p *tmdbProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "tmdb", Name: "TMDB", ContractVersion: 1, Kinds: []string{"films", "series"}, RequiresCredential: true, NetworkHosts: []string{"api.themoviedb.org", "image.tmdb.org", "i.ytimg.com"}, Attribution: "This product uses the TMDB API but is not endorsed or certified by TMDB.", Website: "https://www.themoviedb.org"}
}
func (p *tmdbProvider) wait(ctx context.Context) error {
	return p.rate.wait(ctx, 300*time.Millisecond)
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
	params = cloneProviderParams(params)
	// Caller-owned values and public cache keys must never contain credentials.
	params.Del("api_key")
	if !params.Has("language") {
		params.Set("language", appLanguage)
	}
	key := endpoint + "?" + params.Encode()
	ttl := 7 * 24 * time.Hour
	if endpoint == "/movie/now_playing" || endpoint == "/movie/upcoming" || endpoint == "/tv/popular" {
		ttl = 6 * time.Hour
	}
	_, status, err := fetchProviderHTTP(ctx, providerHTTPCall{
		Client: p.client, DB: p.db, Provider: "tmdb", URL: p.apiBase + key, CacheKey: key,
		Accept: "application/json", MaxBytes: 2 << 20, TTL: ttl,
		Authorize: providerOrigins([]string{p.apiBase}, nil), Wait: p.wait, BeforeCommit: providerCommit,
		Prepare: func(req *http.Request) {
			if len(secret) == 32 && !strings.Contains(secret, ".") {
				q := req.URL.Query()
				q.Set("api_key", secret)
				req.URL.RawQuery = q.Encode()
			} else {
				req.Header.Set("Authorization", "Bearer "+secret)
			}
		},
		Validate: func(data []byte) error {
			if json.Unmarshal(data, out) != nil {
				return errors.New("Réponse TMDB illisible.")
			}
			return nil
		},
	})
	if status == 401 || status == 403 {
		return errors.New("TMDB refuse cette clé. Vérifiez le jeton ou la clé API.")
	}
	if status == 429 {
		return errors.New("TMDB demande une pause. Réessayez plus tard.")
	}
	if status != 0 && status != 200 {
		return fmt.Errorf("Fiche TMDB indisponible (HTTP %d).", status)
	}
	return err
}

func tmdbKind(kind string) string {
	if kind == "series" {
		return "tv"
	}
	return "movie"
}

var artworkPath = regexp.MustCompile(`^/[A-Za-z0-9_-]+\.(?:jpg|png)$`)
var youtubeKey = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

const appLanguage = "fr-FR"

func (p *tmdbProvider) activeArtwork(ctx context.Context, _ string, ref string) ([]byte, error) {
	if err := p.moduleReady(); err != nil {
		return nil, err
	}
	endpoint := p.imageBase + ref
	wait := p.wait
	origin := p.imageBase
	if strings.HasPrefix(ref, "yt:") {
		key := strings.TrimPrefix(ref, "yt:")
		if !youtubeKey.MatchString(key) {
			return nil, errors.New("Miniature invalide.")
		}
		origin = "https://i.ytimg.com"
		endpoint = origin + "/vi/" + key + "/hqdefault.jpg"
		wait = nil
	} else if !artworkPath.MatchString(ref) {
		return nil, errors.New("Image TMDB invalide.")
	}
	body, _, err := fetchProviderHTTP(ctx, providerHTTPCall{Client: p.client, URL: endpoint, MaxBytes: 8 << 20, Authorize: providerOrigins([]string{origin}, nil), Wait: wait})
	return body, err
}
