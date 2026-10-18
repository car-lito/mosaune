package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	presentation "mosaune/extensionkit/presentation/v1"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type musicBrainzProvider struct {
	db     *sql.DB
	client *http.Client
	base   string
	mu     sync.Mutex
	next   time.Time
}

var mbID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newMusicBrainzProvider(db *sql.DB) *musicBrainzProvider {
	return &musicBrainzProvider{db: db, base: "https://musicbrainz.org/ws/2", client: &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
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

type mbCredit struct {
	Name   string
	Join   string `json:"joinphrase"`
	Artist struct {
		ID   string
		Name string
	}
}
type mbRelease struct {
	ID             string
	Title          string
	Date           string
	Country        string
	Disambiguation string
	Group          struct {
		ID          string
		PrimaryType string `json:"primary-type"`
	} `json:"release-group"`
	Credits []mbCredit `json:"artist-credit"`
	Media   []struct {
		Position int
		Format   string
		Tracks   []struct {
			ID        string
			Title     string
			Position  int
			Length    int
			Credits   []mbCredit `json:"artist-credit"`
			Recording struct {
				Title   string
				Length  int
				Credits []mbCredit `json:"artist-credit"`
			}
		}
	}
	Cover struct{ Front bool } `json:"cover-art-archive"`
}

func mbPeople(credits []mbCredit) []Contributor {
	out := []Contributor{}
	for _, c := range credits {
		if len(out) >= 30 {
			break
		}
		if mbID.MatchString(c.Artist.ID) {
			out = append(out, Contributor{ID: c.Artist.ID, Name: truncate(c.Artist.Name, 200), Role: "Artiste"})
		}
	}
	return out
}
func mbArtist(credits []mbCredit) string {
	var b strings.Builder
	for _, c := range credits {
		name := c.Name
		if name == "" {
			name = c.Artist.Name
		}
		b.WriteString(name + c.Join)
	}
	return truncate(b.String(), 500)
}
func mbMetadata(r mbRelease) Metadata {
	year := 0
	if len(r.Date) >= 4 {
		year, _ = strconv.Atoi(r.Date[:4])
	}
	ref := r.ID
	if mbID.MatchString(r.Group.ID) {
		ref += "/" + r.Group.ID
	}
	label := map[string]string{"Album": "Album", "Single": "Single", "EP": "EP", "Broadcast": "Émission"}[r.Group.PrimaryType]
	if label == "" {
		label = "Sortie musicale"
	}
	return Metadata{Presentation: &presentation.Result{Version: 1, Label: label, Preview: &presentation.Preview{AspectRatio: 1, Shape: "rectangle", Fit: "cover"}}, Title: truncate(r.Title, 500), Album: truncate(r.Title, 500), Artist: mbArtist(r.Credits), AlbumArtist: mbArtist(r.Credits), Year: year, ReleaseDate: r.Date, Provider: "musicbrainz", ProviderID: r.ID, MatchState: "matched", Contributors: mbPeople(r.Credits), ArtworkRef: ref, Edition: truncate(strings.Trim(r.Country+" · "+r.Date+" · "+r.Disambiguation, " ·"), 400)}
}
func mbQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
func (p *musicBrainzProvider) searchReleases(ctx context.Context, q string, page int) (DiscoveryPage, error) {
	out := DiscoveryPage{Items: []Metadata{}, Page: page}
	var response struct {
		Count    int
		Releases []mbRelease
	}
	if err := p.get(ctx, "/release/", url.Values{"query": {q}, "limit": {"20"}, "offset": {strconv.Itoa((page - 1) * 20)}}, &response); err != nil {
		return out, err
	}
	out.Pages = max(1, min(50, (response.Count+19)/20))
	for _, r := range response.Releases {
		if mbID.MatchString(r.ID) && r.Title != "" {
			out.Items = append(out.Items, mbMetadata(r))
		}
		if len(out.Items) >= 20 {
			break
		}
	}
	return out, nil
}
func (p *musicBrainzProvider) Discover(ctx context.Context, _ string, q DiscoveryQuery) (DiscoveryPage, error) {
	if q.Kind != "music" || q.Page < 1 || q.Page > 50 {
		return DiscoveryPage{}, errors.New("Recherche musicale invalide.")
	}
	query := "release:" + mbQuote(q.Query) + " OR artist:" + mbQuote(q.Query)
	if q.Seed != "" {
		if !mbID.MatchString(q.Seed) {
			return DiscoveryPage{}, errors.New("Album invalide.")
		}
		var release mbRelease
		if err := p.get(ctx, "/release/"+q.Seed, url.Values{"inc": {"artist-credits"}}, &release); err != nil {
			return DiscoveryPage{}, err
		}
		if len(release.Credits) == 0 || !mbID.MatchString(release.Credits[0].Artist.ID) {
			return DiscoveryPage{Items: []Metadata{}, Page: 1, Pages: 1}, nil
		}
		query = "arid:" + release.Credits[0].Artist.ID
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
	if q.Kind != "music" || !mbID.MatchString(q.ID) {
		return Metadata{}, errors.New("Édition musicale invalide.")
	}
	var r mbRelease
	if err := p.get(ctx, "/release/"+q.ID, url.Values{"inc": {"recordings+artist-credits+labels+release-groups"}}, &r); err != nil {
		return Metadata{}, err
	}
	m := mbMetadata(r)
	m.DetailVersion = 2
	m.Tracks = []CatalogTrack{}
	for _, disc := range r.Media {
		for _, track := range disc.Tracks {
			if len(m.Tracks) >= 200 {
				m.Edition += " · liste limitée à 200 pistes"
				return m, nil
			}
			title := track.Title
			if title == "" {
				title = track.Recording.Title
			}
			credits := track.Credits
			if len(credits) == 0 {
				credits = track.Recording.Credits
			}
			if len(credits) == 0 {
				credits = r.Credits
			}
			length := track.Length
			if length == 0 {
				length = track.Recording.Length
			}
			m.Tracks = append(m.Tracks, CatalogTrack{ID: track.ID, Title: truncate(title, 500), Disc: disc.Position, Number: track.Position, Duration: length / 1000, Artists: mbPeople(credits)})
		}
	}
	return m, nil
}
func (p *musicBrainzProvider) Artwork(ctx context.Context, _ string, ref string) ([]byte, error) {
	return p.visualArtwork(ctx, ref)
}

func (p *musicBrainzProvider) Shelf(ctx context.Context, _ string, shelf, artist string) (DiscoveryPage, error) {
	if shelf != "upcoming" || !mbID.MatchString(artist) {
		return DiscoveryPage{}, errors.New("Choisissez d’abord un album identifié pour connaître son artiste.")
	}
	return p.searchReleases(ctx, "arid:"+artist+" AND date:["+time.Now().UTC().Format("2006-01-02")+" TO *]", 1)
}
func (p *musicBrainzProvider) Profile(ctx context.Context, _ string, id string, page int) (PersonProfile, error) {
	out := PersonProfile{ID: id, Page: page}
	if !mbID.MatchString(id) {
		return out, errors.New("Artiste invalide.")
	}
	var artist struct {
		Name           string
		Disambiguation string
		Country        string
	}
	if err := p.get(ctx, "/artist/"+id, nil, &artist); err != nil {
		return out, err
	}
	out.Name = truncate(artist.Name, 300)
	out.Detail = truncate(strings.Trim(artist.Country+" · "+artist.Disambiguation, " ·"), 500)
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
