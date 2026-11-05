package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type musicBrainzProvider struct {
	owner       *catalog
	resultsGate chan struct{}
	db          *sql.DB
	client      *http.Client
	base        string
	rate        providerRateGate
}

var mbID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newMusicBrainzProvider(db *sql.DB) *musicBrainzProvider {
	return &musicBrainzProvider{owner: &catalog{db: db}, resultsGate: make(chan struct{}, 1), db: db, client: &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
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
	params = cloneProviderParams(params)
	params.Set("fmt", "json")
	call, err := p.musicCall(ctx, "musicbrainz", endpoint, params)
	if err != nil {
		return err
	}
	call.Validate = func(data []byte) error {
		if json.Unmarshal(data, out) != nil {
			return errors.New("Réponse musicale illisible.")
		}
		return nil
	}
	_, status, err := fetchProviderHTTP(ctx, call)
	if status != 0 && status != 200 {
		return fmt.Errorf("MusicBrainz ne peut pas répondre pour le moment (HTTP %d). Réessayez plus tard.", status)
	}
	return err
}

func (p *musicBrainzProvider) searchMusic(ctx context.Context, q d.Request) (DiscoveryPage, error) {
	ctx, done, e := p.openMusicResults(ctx)
	if e != nil {
		return DiscoveryPage{}, e
	}
	defer done()
	out := DiscoveryPage{Items: []Metadata{}, Page: q.Page}
	decoded, e := p.requestMusic(ctx, q)
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
	query := d.Request{Operation: "discover", Title: q.Query, Page: q.Page}
	if q.Seed != "" {
		if !mbID.MatchString(q.Seed) {
			return DiscoveryPage{}, errors.New("Album invalide.")
		}
		decoded, e := p.requestMusic(ctx, d.Request{Operation: "seed", ID: q.Seed, Page: 1})
		if e != nil {
			return DiscoveryPage{}, e
		}
		if decoded.ArtistID == "" {
			return DiscoveryPage{Items: []Metadata{}, Page: 1, Pages: 1}, nil
		}
		query = d.Request{Operation: "artist-releases", ID: decoded.ArtistID, Page: q.Page}
	}
	return p.searchMusic(ctx, query)
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
	decoded, e := p.requestMusic(ctx, d.Request{Operation: "release", ID: q.ID, Page: 1})
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
	return p.searchMusic(ctx, d.Request{Operation: "upcoming", ID: artist, Date: time.Now().UTC().Format("2006-01-02"), Page: 1})
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
	decoded, e := p.requestMusic(ctx, d.Request{Operation: "artist", ID: id, Page: page})
	if e != nil {
		return out, e
	}
	out.Name, out.Detail = decoded.Name, decoded.Detail
	releases, err := p.searchMusic(ctx, d.Request{Operation: "artist-releases", ID: id, Page: page})
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
