package main

import (
	"context"
	"encoding/json"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"time"
)

type commonsPhoto = d.Photo

func (p *musicBrainzProvider) publicJSON(ctx context.Context, endpoint string, out any) error {
	_, _, err := fetchProviderHTTP(ctx, providerHTTPCall{
		Client: p.client, DB: p.db, Provider: "musicbrainz", URL: endpoint, CacheKey: "visual-json:" + endpoint,
		TTL: 7 * 24 * time.Hour, Accept: "application/json", MaxBytes: d.MaxBody,
		Authorize:    providerOrigins([]string{"https://www.wikidata.org", "https://commons.wikimedia.org"}, nil),
		BeforeCommit: p.owner.recognitionCommit,
		Validate:     func(data []byte) error { return json.Unmarshal(data, out) },
	})
	return err
}
func (p *musicBrainzProvider) fetchVisual(ctx context.Context, endpoint string) ([]byte, int, error) {
	return fetchProviderHTTP(ctx, providerHTTPCall{
		Client: p.client, URL: endpoint, MaxBytes: 8 << 20,
		Authorize: providerOrigins([]string{"https://coverartarchive.org", "https://archive.org", "https://upload.wikimedia.org", "https://thumb.wikimedia.org"}, []string{"archive.org"}),
	})
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
