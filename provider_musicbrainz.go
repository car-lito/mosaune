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
	"strconv"
	"strings"
	"sync"
	"time"
)

type musicBrainzProvider struct {
	owner       *catalog
	resultsGate chan struct{}
	db          *sql.DB
	client      *http.Client
	base        string
	mu          sync.Mutex
	next        time.Time
}

var mbID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newMusicBrainzProvider(db *sql.DB) *musicBrainzProvider {
	return &musicBrainzProvider{owner: &catalog{db: db}, resultsGate: make(chan struct{}, 1), db: db, base: "https://musicbrainz.org/ws/2", client: &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		host := r.URL.Hostname()
		if len(via) > 5 || r.URL.User != nil || (r.URL.Scheme != "https" && r.URL.Scheme != "http") || r.URL.Port() != "" || !(host == "coverartarchive.org" || host == "archive.org" || strings.HasSuffix(host, ".archive.org") || host == "www.wikidata.org" || host == "commons.wikimedia.org" || host == "upload.wikimedia.org" || host == "thumb.wikimedia.org") {
			return errors.New("Redirection refusée.")
		}
		r.URL.Scheme = "https"
		return nil
	}}}
}
func (p *musicBrainzProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "musicbrainz", Name: "MusicBrainz", ContractVersion: 1, Kinds: []string{"music"}, Website: "https://musicbrainz.org", NetworkHosts: []string{"musicbrainz.org", "coverartarchive.org", "archive.org", "www.wikidata.org", "commons.wikimedia.org", "upload.wikimedia.org", "thumb.wikimedia.org"}, Attribution: "MusicBrainz / MetaBrainz — core data CC0; supplementary data CC BY-NC-SA 3.0. Cover Art Archive : droits des images détenus par leurs ayants droit."}
}
func (p *musicBrainzProvider) get(ctx context.Context, endpoint string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("fmt", "json")
	key := endpoint + "?" + params.Encode()
	var cached []byte
	if p.db != nil && p.db.QueryRowContext(ctx, `SELECT body FROM provider_cache WHERE provider='musicbrainz' AND key=? AND expires>?`, key, time.Now().Unix()).Scan(&cached) == nil {
		return json.Unmarshal(cached, out)
	}
	p.mu.Lock()
	delay := time.Until(p.next)
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			p.mu.Unlock()
			return ctx.Err()
		case <-timer.C:
		}
	}
	p.next = time.Now().Add(1100 * time.Millisecond)
	p.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, "GET", p.base+key, nil)
	if err != nil {
		return errors.New("Requête musicale invalide.")
	}
	req.Header.Set("User-Agent", "Mosaune/"+version+" (personal self-hosted media catalog)")
	req.Header.Set("Accept", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return errors.New("MusicBrainz est inaccessible. Vos fichiers restent disponibles.")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("MusicBrainz ne peut pas répondre pour le moment (HTTP %d). Réessayez plus tard.", response.StatusCode)
	}
	data, err := readBounded(response.Body, 2<<20)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return errors.New("Réponse musicale illisible.")
	}
	if p.db != nil {
		_, _ = p.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<? AND NOT (key LIKE 'visual-image:v2:%' AND length(body)>0)`, time.Now().Unix())
		_, _ = p.db.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT 'musicbrainz',?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+?<16777216`, key, data, time.Now().Add(24*time.Hour).Unix(), len(data))
	}
	return nil
}

func mbQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
func (p *musicBrainzProvider) searchReleases(ctx context.Context, q string, page int) (DiscoveryPage, error) {
	ctx, done, e := p.openMusicResults(ctx)
	if e != nil {
		return DiscoveryPage{}, e
	}
	defer done()
	out := DiscoveryPage{Items: []Metadata{}, Page: page}
	var raw json.RawMessage
	if e := p.get(ctx, "/release/", url.Values{"query": {q}, "limit": {"20"}, "offset": {strconv.Itoa((page - 1) * 20)}}, &raw); e != nil {
		return out, e
	}
	decoded, e := p.decodeMusic(ctx, "releases", "", page, raw)
	if e != nil {
		return out, e
	}
	out.Pages = decoded.Pages
	for _, m := range decoded.Items {
		out.Items = append(out.Items, musicResultMetadata(m))
	}
	return out, nil
}

func (p *musicBrainzProvider) Discover(ctx context.Context, _ string, q DiscoveryQuery) (DiscoveryPage, error) {
	ctx, done, openErr := p.openMusicResults(ctx)
	if openErr != nil {
		return DiscoveryPage{}, openErr
	}
	defer done()
	if q.Kind != "music" || q.Page < 1 || q.Page > 50 {
		return DiscoveryPage{}, errors.New("Recherche musicale invalide.")
	}
	query := "release:" + mbQuote(q.Query) + " OR artist:" + mbQuote(q.Query)
	if q.Seed != "" {
		if !mbID.MatchString(q.Seed) {
			return DiscoveryPage{}, errors.New("Album invalide.")
		}
		var raw json.RawMessage
		if e := p.get(ctx, "/release/"+q.Seed, url.Values{"inc": {"artist-credits"}}, &raw); e != nil {
			return DiscoveryPage{}, e
		}
		decoded, e := p.decodeMusic(ctx, "seed", q.Seed, 1, raw)
		if e != nil {
			return DiscoveryPage{}, e
		}
		if decoded.ArtistID == "" {
			return DiscoveryPage{Items: []Metadata{}, Page: 1, Pages: 1}, nil
		}
		query = "arid:" + decoded.ArtistID
	}
	return p.searchReleases(ctx, query, q.Page)
}
func (p *musicBrainzProvider) Search(ctx context.Context, secret string, q MetadataQuery) ([]MetadataCandidate, error) {
	page, err := p.Discover(ctx, secret, DiscoveryQuery{Kind: q.Kind, Query: q.Title, Page: 1})
	out := []MetadataCandidate{}
	for _, m := range page.Items {
		out = append(out, MetadataCandidate{ID: m.ProviderID, Title: m.Title, Year: m.Year, Overview: m.Artist + " · " + m.Edition})
	}
	return out, err
}
func (p *musicBrainzProvider) Details(ctx context.Context, _ string, q MetadataSelection) (Metadata, error) {
	ctx, done, openErr := p.openMusicResults(ctx)
	if openErr != nil {
		return Metadata{}, openErr
	}
	defer done()
	if q.Kind != "music" || !mbID.MatchString(q.ID) {
		return Metadata{}, errors.New("Édition musicale invalide.")
	}
	var raw json.RawMessage
	if e := p.get(ctx, "/release/"+q.ID, url.Values{"inc": {"recordings+artist-credits+labels+release-groups"}}, &raw); e != nil {
		return Metadata{}, e
	}
	decoded, e := p.decodeMusic(ctx, "release", q.ID, 1, raw)
	if e != nil {
		return Metadata{}, e
	}
	return musicResultMetadata(decoded.Items[0]), nil
}

func (p *musicBrainzProvider) Artwork(ctx context.Context, _ string, ref string) ([]byte, error) {
	return p.visualArtwork(ctx, ref)
}

func (p *musicBrainzProvider) Shelf(ctx context.Context, _ string, shelf, artist string) (DiscoveryPage, error) {
	ctx, done, openErr := p.openMusicResults(ctx)
	if openErr != nil {
		return DiscoveryPage{}, openErr
	}
	defer done()
	if shelf != "upcoming" || !mbID.MatchString(artist) {
		return DiscoveryPage{}, errors.New("Choisissez d’abord un album identifié pour connaître son artiste.")
	}
	return p.searchReleases(ctx, "arid:"+artist+" AND date:["+time.Now().UTC().Format("2006-01-02")+" TO *]", 1)
}
func (p *musicBrainzProvider) Profile(ctx context.Context, _ string, id string, page int) (PersonProfile, error) {
	ctx, done, openErr := p.openMusicResults(ctx)
	if openErr != nil {
		return PersonProfile{}, openErr
	}
	defer done()
	out := PersonProfile{ID: id, Page: page}
	if !mbID.MatchString(id) {
		return out, errors.New("Artiste invalide.")
	}
	var raw json.RawMessage
	if e := p.get(ctx, "/artist/"+id, nil, &raw); e != nil {
		return out, e
	}
	decoded, e := p.decodeMusic(ctx, "artist", id, page, raw)
	if e != nil {
		return out, e
	}
	out.Name, out.Detail = decoded.Name, decoded.Detail
	releases, err := p.searchReleases(ctx, "arid:"+id, page)
	if err != nil {
		return out, err
	}
	out.Pages = releases.Pages
	for _, m := range releases.Items {
		out.Items = append(out.Items, ProfileItem{Kind: "music", Metadata: m})
	}
	// A slow photo source must not make an otherwise usable discography fail.
	photoCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	if photo, err := p.artistPhoto(photoCtx, id); err == nil {
		out.Image = assetURL("musicbrainz", "commons:"+photo.File)
		out.ImageSource = photo.Source
		out.ImageCredit = photo.Credit
		out.ImageLicense = photo.License
	}
	return out, nil
}
