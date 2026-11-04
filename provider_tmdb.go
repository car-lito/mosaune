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
	profileCache   providerProfileCache
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
	return &tmdbProvider{state: providerLifecycle{moduleUnavailable: !installed}, extension: tmdbExtension{installed: installed, enabled: installed, gate: make(chan struct{}, 1)}, db: db, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection refusée") }}}
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
	profile, e := p.configuredProfile(ctx)
	if e != nil {
		return e
	}
	source, e := profile.Source("api")
	if e != nil {
		return e
	}
	if p.apiBase != "" {
		source.BaseURL = p.apiBase
	} // host-only fixture override
	params = cloneProviderParams(params)
	if !params.Has("language") {
		params.Set("language", profile.Language)
	}
	call, e := providerSourceCall(p.client, &p.rate, source, endpoint, params, secret)
	if e != nil {
		return e
	}
	call.DB = p.db
	call.Provider = "tmdb"
	call.BeforeCommit = providerCommit
	call.Validate = func(data []byte) error {
		if json.Unmarshal(data, out) != nil {
			return errors.New("Réponse TMDB illisible.")
		}
		return nil
	}
	_, status, err := fetchProviderHTTP(ctx, call)
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
	profile, e := p.transportProfile(ctx)
	if e != nil {
		return nil, e
	}
	source, e := profile.Source("images")
	if e != nil {
		return nil, e
	}
	if p.imageBase != "" {
		source.BaseURL = p.imageBase
	}
	path := ref
	if strings.HasPrefix(ref, "yt:") {
		key := strings.TrimPrefix(ref, "yt:")
		if !youtubeKey.MatchString(key) {
			return nil, errors.New("Miniature invalide.")
		}
		source, e = profile.Source("trailers")
		if e != nil {
			return nil, e
		}
		path = "/" + key + "/hqdefault.jpg"
	} else if !artworkPath.MatchString(ref) {
		return nil, errors.New("Image TMDB invalide.")
	}
	call, e := providerSourceCall(p.client, &p.rate, source, path, nil, "")
	if e != nil {
		return nil, e
	}
	body, _, e := fetchProviderHTTP(ctx, call)
	return body, e
}
