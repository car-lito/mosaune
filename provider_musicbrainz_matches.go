package main

import (
	"context"
	musicrequests "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/http"
	"net/url"
	"strings"
)

func legacyMusicBrainzMatchRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/music/matches", func(w http.ResponseWriter, r *http.Request) { c.matchAlbums(w, r, allowed, c.provider("musicbrainz")) })
}

func (p *musicBrainzProvider) MatchAlbums(ctx context.Context, secret string, q AlbumMatchQuery) (AlbumMatchResult, error) {
	if q.ID == "" {
		page, err := p.searchMusic(ctx, musicrequests.Request{Operation: "match", Title: q.Title, Artist: q.Artist, Page: q.Page})
		return AlbumMatchResult{Page: page}, err
	}
	id := q.ID
	if u, e := url.Parse(id); e == nil && u.Scheme == "https" && u.Host == "musicbrainz.org" && u.User == nil && strings.HasPrefix(u.Path, "/release/") {
		id = strings.TrimPrefix(strings.TrimSuffix(u.Path, "/"), "/release/")
	}
	if !mbID.MatchString(id) {
		return AlbumMatchResult{}, errAlbumMatchInput
	}
	md, err := p.Details(ctx, secret, MetadataSelection{Kind: "music", ID: id})
	return AlbumMatchResult{Page: DiscoveryPage{Page: 1, Pages: 1, Items: []Metadata{md}}, Detailed: true}, err
}
