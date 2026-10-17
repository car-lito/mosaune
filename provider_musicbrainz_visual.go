package main

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type commonsPhoto struct{ File, URL, Source, Credit, License string }

var wikidataPath = regexp.MustCompile(`^/wiki/(Q[1-9][0-9]*)$`)
var htmlTag = regexp.MustCompile(`<[^>]*>`)

func plainCredit(s string) string {
	return truncate(strings.TrimSpace(html.UnescapeString(htmlTag.ReplaceAllString(s, ""))), 500)
}
func safeCommonsFile(file string) bool {
	low := strings.ToLower(file)
	return len(file) > 0 && len(file) <= 500 && !strings.ContainsAny(file, "/\\\x00\r\n") && (strings.HasSuffix(low, ".jpg") || strings.HasSuffix(low, ".jpeg") || strings.HasSuffix(low, ".png"))
}
func safeUpload(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && (u.Host == "upload.wikimedia.org" || u.Host == "thumb.wikimedia.org") && u.User == nil
}

func (p *musicBrainzProvider) publicJSON(ctx context.Context, endpoint string, out any) error {
	key := "visual-json:" + endpoint
	var data []byte
	if p.db != nil && p.db.QueryRowContext(ctx, `SELECT body FROM provider_cache WHERE provider='musicbrainz' AND key=? AND expires>?`, key, time.Now().Unix()).Scan(&data) == nil {
		return json.Unmarshal(data, out)
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	req.Header.Set("User-Agent", "Mosaune/"+version+" (self-hosted media catalog)")
	req.Header.Set("Accept", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("Source d’image indisponible.")
	}
	data, err = readBounded(response.Body, 2<<20)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return err
	}
	if p.db != nil {
		_, _ = p.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<?`, time.Now().Unix())
		_, _ = p.db.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT 'musicbrainz',?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+?<16777216`, key, data, time.Now().Add(7*24*time.Hour).Unix(), len(data))
	}
	return nil
}
func (p *musicBrainzProvider) commonsPhoto(ctx context.Context, file string) (commonsPhoto, error) {
	out := commonsPhoto{File: file}
	if !safeCommonsFile(file) {
		return out, errors.New("Photo invalide.")
	}
	var response struct {
		Query struct {
			Pages map[string]struct {
				Info []struct {
					Thumb    string `json:"thumburl"`
					URL      string
					Metadata map[string]struct{ Value any } `json:"extmetadata"`
				} `json:"imageinfo"`
			}
		}
	}
	params := url.Values{"action": {"query"}, "format": {"json"}, "prop": {"imageinfo"}, "titles": {"File:" + file}, "iiprop": {"url|extmetadata"}, "iiextmetadatafilter": {"Artist|LicenseShortName"}, "iiurlwidth": {"320"}}
	if err := p.publicJSON(ctx, "https://commons.wikimedia.org/w/api.php?"+params.Encode(), &response); err != nil {
		return out, err
	}
	for _, page := range response.Query.Pages {
		for _, info := range page.Info {
			out.URL = info.Thumb
			if out.URL == "" {
				out.URL = info.URL
			}
			if !safeUpload(out.URL) {
				continue
			}
			out.Source = "https://commons.wikimedia.org/wiki/File:" + url.PathEscape(file)
			credit, _ := info.Metadata["Artist"].Value.(string)
			license, _ := info.Metadata["LicenseShortName"].Value.(string)
			out.Credit = plainCredit(credit)
			out.License = plainCredit(license)
			return out, nil
		}
	}
	return out, errors.New("Photo non renseignée.")
}
func (p *musicBrainzProvider) artistPhoto(ctx context.Context, id string) (commonsPhoto, error) {
	var artist struct {
		Relations []struct {
			Type string
			URL  struct{ Resource string }
		}
	}
	if !mbID.MatchString(id) {
		return commonsPhoto{}, errors.New("Artiste invalide.")
	}
	if err := p.get(ctx, "/artist/"+id, url.Values{"inc": {"url-rels"}}, &artist); err != nil {
		return commonsPhoto{}, err
	}
	entity := ""
	for _, rel := range artist.Relations {
		u, err := url.Parse(rel.URL.Resource)
		if err != nil || rel.Type != "wikidata" || u.User != nil || u.Port() != "" || (u.Hostname() != "www.wikidata.org" && u.Hostname() != "wikidata.org") {
			continue
		}
		if m := wikidataPath.FindStringSubmatch(u.Path); len(m) == 2 {
			entity = m[1]
			break
		}
	}
	if entity == "" {
		return commonsPhoto{}, errors.New("Photo non renseignée.")
	}
	var claims struct {
		Claims map[string][]struct {
			Rank string
			Main struct {
				Data struct{ Value string } `json:"datavalue"`
			} `json:"mainsnak"`
		}
	}
	params := url.Values{"action": {"wbgetclaims"}, "format": {"json"}, "entity": {entity}, "property": {"P18"}}
	if err := p.publicJSON(ctx, "https://www.wikidata.org/w/api.php?"+params.Encode(), &claims); err != nil {
		return commonsPhoto{}, err
	}
	for _, claim := range claims.Claims["P18"] {
		if claim.Rank != "deprecated" && safeCommonsFile(claim.Main.Data.Value) {
			return p.commonsPhoto(ctx, claim.Main.Data.Value)
		}
	}
	return commonsPhoto{}, errors.New("Photo non renseignée.")
}
func (p *musicBrainzProvider) fetchVisual(ctx context.Context, endpoint string) ([]byte, int, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	req.Header.Set("User-Agent", "Mosaune/"+version)
	response, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, response.StatusCode, errors.New("Image indisponible.")
	}
	data, err := readBounded(response.Body, 8<<20)
	return data, 200, err
}
func (p *musicBrainzProvider) visualArtwork(ctx context.Context, ref string) ([]byte, error) {
	if strings.HasPrefix(ref, "commons:") {
		photo, err := p.commonsPhoto(ctx, strings.TrimPrefix(ref, "commons:"))
		if err != nil {
			return nil, err
		}
		data, _, err := p.fetchVisual(ctx, photo.URL)
		return data, err
	}
	parts := strings.Split(ref, "/")
	if len(parts) > 2 || !mbID.MatchString(parts[0]) || (len(parts) == 2 && !mbID.MatchString(parts[1])) {
		return nil, errors.New("Pochette invalide.")
	}
	data, status, err := p.fetchVisual(ctx, "https://coverartarchive.org/release/"+parts[0]+"/front-500")
	if err == nil || status != 404 {
		return data, err
	}
	group := ""
	if len(parts) == 2 {
		group = parts[1]
	} else {
		var release mbRelease
		if p.get(ctx, "/release/"+parts[0], url.Values{"inc": {"release-groups"}}, &release) == nil {
			group = release.Group.ID
		}
	}
	if mbID.MatchString(group) {
		data, _, err = p.fetchVisual(ctx, "https://coverartarchive.org/release-group/"+group+"/front-500")
		return data, err
	}
	return nil, errors.New("Aucune pochette renseignée pour cette édition.")
}
func (p *musicBrainzProvider) ResolveArtist(ctx context.Context, name string) ([]Contributor, error) {
	var response struct {
		Artists []struct {
			ID             string
			Name           string
			Disambiguation string
		}
	}
	err := p.get(ctx, "/artist/", url.Values{"query": {"artist:" + mbQuote(name)}, "limit": {"8"}}, &response)
	out := []Contributor{}
	for _, artist := range response.Artists {
		if mbID.MatchString(artist.ID) {
			out = append(out, Contributor{ID: artist.ID, Name: truncate(artist.Name, 200), Role: truncate(artist.Disambiguation, 300)})
		}
	}
	return out, err
}
