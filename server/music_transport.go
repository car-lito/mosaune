package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/configuration"
	legacy "mosaune/extensionkit/providers/musicbrainz"
	"mosaune/extensionkit/transport"
	"net/url"
)

// Origins are the host's existing network grant. Module settings can narrow
// limits and change API paths, but cannot grant access to another service.
func validateMusicTransport(profile transport.Profile) error {
	bad := errors.New("Déclaration des sources musicales refusée.")
	if profile.Validate(configuration.Schema{}) != nil || len(profile.Sources) != 6 {
		return bad
	}
	hosts := map[string]string{
		"musicbrainz": "musicbrainz.org", "wikidata": "www.wikidata.org",
		"commons": "commons.wikimedia.org", "coverart": "coverartarchive.org",
		"upload": "upload.wikimedia.org", "thumb": "thumb.wikimedia.org",
	}
	for _, source := range profile.Sources {
		u, err := url.Parse(source.BaseURL)
		if err != nil || u.Host != hosts[source.ID] || source.Auth != "" || source.TimeoutMS > 12000 {
			return bad
		}
		switch source.ID {
		case "musicbrainz":
			if source.Format != "json" || source.IntervalMS < 1100 {
				return bad
			}
		case "wikidata", "commons":
			if source.Format != "json" || source.MaxBytes > 512<<10 {
				return bad
			}
		default:
			if source.Format != "image" || u.Path != "" || source.CacheSeconds != 0 || len(source.CachePaths) != 0 {
				return bad
			}
		}
	}
	return nil
}

func loadMusicTransport(ctx context.Context, reader *cataloguer) (transport.Profile, error) {
	var info struct {
		Version          int
		TransportVersion int
	}
	if err := reader.call(ctx, "describe-requests", struct{}{}, &info); err != nil || info.Version != 1 {
		return transport.Profile{}, errors.New("Mettez à jour le composant musical pour préparer les recherches.")
	}
	// Old installed modules use the original bounded policy until replaced.
	if info.TransportVersion == 0 {
		return legacy.Transport(), nil
	}
	var profile transport.Profile
	if info.TransportVersion != 1 {
		return profile, errors.New("Version des sources musicales incompatible.")
	}
	if err := reader.call(ctx, "describe-transport", struct{}{}, &profile); err != nil {
		return profile, err
	}
	return profile, validateMusicTransport(profile)
}

func (p *musicBrainzProvider) musicCall(ctx context.Context, id, path string, params url.Values) (providerHTTPCall, error) {
	reader, ok := ctx.Value(musicResultsKey{}).(*musicResultsReader)
	if !ok || reader.owner != p {
		return providerHTTPCall{}, errors.New("Composant musical absent.")
	}
	// Image URLs have already passed the visual contract. Match their origin
	// against declared sources and preserve escaped filenames exactly.
	absolute := ""
	if id == "image" {
		u, err := url.Parse(path)
		if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return providerHTTPCall{}, errors.New("Adresse d’image refusée.")
		}
		id = ""
		for _, candidate := range reader.profile.Sources {
			if (candidate.ID == "upload" || candidate.ID == "thumb") && candidate.BaseURL == u.Scheme+"://"+u.Host {
				id = candidate.ID
				break
			}
		}
		absolute, path = path, "/"
	}
	source, err := reader.profile.Source(id)
	if err != nil {
		return providerHTTPCall{}, err
	}
	// Host-only endpoint override used by local fixture servers.
	if id == "musicbrainz" && p.base != "" {
		source.BaseURL = p.base
	}
	call, err := providerSourceCall(p.client, &p.rate, source, path, params, "")
	if err != nil {
		return call, err
	}
	if absolute != "" {
		call.URL, call.CacheKey = absolute, absolute
	}
	if id == "coverart" {
		call.Authorize = providerOrigins([]string{source.BaseURL, "https://archive.org"}, []string{"archive.org"})
	}
	call.DB, call.Provider = p.db, "musicbrainz"
	call.BeforeCommit = p.owner.recognitionCommit
	return call, nil
}
