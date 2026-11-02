package main

import (
	"context"
	"encoding/json"
	"errors"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/http"
	"time"
)

type commonsPhoto = d.Photo

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
	data, err = readBounded(response.Body, d.MaxBody)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return err
	}
	if p.db != nil {
		_, _ = p.db.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<? AND NOT (key LIKE 'visual-image:v2:%' AND length(body)>0)`, time.Now().Unix())
		_, _ = p.db.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT 'musicbrainz',?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+?<16777216`, key, data, time.Now().Add(7*24*time.Hour).Unix(), len(data))
	}
	return nil
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
func (p *musicBrainzProvider) ResolveArtist(ctx context.Context, name string) ([]Contributor, error) {
	ctx, done, e := p.openMusicResults(ctx)
	if e != nil {
		return nil, e
	}
	defer done()
	decoded, e := p.requestMusic(ctx, d.Request{Operation: "artists", Title: name, Page: 1})
	if e != nil {
		return nil, e
	}
	out := []Contributor{}
	for _, a := range decoded.Artists {
		out = append(out, Contributor{ID: a.ID, Name: a.Name, Role: a.Role})
	}
	return out, nil
}
