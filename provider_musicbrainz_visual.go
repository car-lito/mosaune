package main

import (
	"context"
	"encoding/json"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/url"
)

type commonsPhoto = d.Photo

func (p *musicBrainzProvider) publicJSON(ctx context.Context, source, path string, params url.Values, out any) error {
	call, err := p.musicCall(ctx, source, path, params)
	if err != nil {
		return err
	}
	call.Validate = func(data []byte) error { return json.Unmarshal(data, out) }
	_, _, err = fetchProviderHTTP(ctx, call)
	return err
}
func (p *musicBrainzProvider) fetchVisual(ctx context.Context, source, path string) ([]byte, int, error) {
	call, err := p.musicCall(ctx, source, path, nil)
	if err != nil {
		return nil, 0, err
	}
	return fetchProviderHTTP(ctx, call)
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
