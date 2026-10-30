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
