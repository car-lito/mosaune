package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	video "mosaune/extensionkit/capabilities/video/v1"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	runtime "mosaune/extensionkit/runtime"
	"net/url"
	"strconv"
	"strings"
)

func (p *tmdbProvider) activeDetails(ctx context.Context, secret string, s MetadataSelection) (Metadata, error) {
	p.extension.mu.Lock()
	enabled := p.extension.enabled && (p.extension.blocked == nil || !p.extension.blocked())
	p.extension.mu.Unlock()
	if !enabled {
		return p.detailsNative(ctx, secret, s)
	}
	selection := video.Selection{ID: s.ID, Kind: s.Kind, Season: s.Season, Episode: s.Episode, Language: appLanguage}
	id, err := strconv.Atoi(s.ID)
	if selection.Validate() != nil || err != nil || id <= 0 || id > 2147483647 {
		return Metadata{}, errors.New("Sélection TMDB invalide.")
	}
	engine, release, err := p.extensionEngine(ctx)
	if err != nil {
		return Metadata{}, err
	}
	defer release()
	if engine == nil {
		return p.detailsNative(ctx, secret, s)
	}
	payload, _ := json.Marshal(selection)
	used := map[string]bool{}
	var requiredErr error
	out, err := engine.CallWithServices(ctx, bundled.Descriptor().ID, protocol.Request{Protocol: 1, ID: "tmdb-details", Capability: video.DetailsCapability, Operation: "details", Payload: payload}, runtime.ServiceGrants{
		"tmdb.api.v1": func(call context.Context, operation string, raw json.RawMessage) (json.RawMessage, error) {
			params, path, e := tmdbDetailsGrant(operation, raw, selection)
			if e != nil {
				return nil, e
			}
			if used[path] {
				return nil, errors.New("duplicate details request")
			}
			used[path] = true
			var body json.RawMessage
			e = p.get(call, secret, path, params, &body)
			if e != nil && !strings.HasSuffix(path, "/videos") {
				requiredErr = e
			}
			return body, e
		},
	})
	if requiredErr != nil {
		return Metadata{}, requiredErr
	}
	if err != nil {
		return Metadata{}, errors.New("Fiche interrompue ou module TMDB indisponible. Réessayez ou désactivez le module dans Extensions.")
	}
	if out.Response.Error != nil {
		return Metadata{}, errors.New("Le module TMDB n’a pas pu préparer cette fiche.")
	}
	return tmdbExtensionMetadata(out.Response.Result, selection)
}

func tmdbDetailsGrant(operation string, raw json.RawMessage, s video.Selection) (url.Values, string, error) {
	var request struct {
		Path  string            `json:"path"`
		Query map[string]string `json:"query"`
	}
	deny := errors.New("TMDB details request denied")
	id, err := strconv.Atoi(s.ID)
	if s.Validate() != nil || err != nil || id <= 0 || id > 2147483647 || operation != "get" || protocol.DecodeOne(raw, &request) != nil {
		return nil, "", deny
	}
	base := fmt.Sprintf("/%s/%d", tmdbKind(s.Kind), id)
	expected := map[string]string{"language": s.Language}
	switch request.Path {
	case base:
		expected["append_to_response"] = "credits,videos"
		expected["include_video_language"] = strings.Split(s.Language, "-")[0] + ",en,null"
	case base + "/videos":
		expected["language"] = ""
	default:
		if s.Kind != "series" || s.Episode <= 0 || request.Path != fmt.Sprintf("%s/season/%d/episode/%d", base, s.Season, s.Episode) {
			return nil, "", deny
		}
	}
	if len(request.Query) != len(expected) {
		return nil, "", deny
	}
	values := url.Values{}
	for k, v := range expected {
		actual, exists := request.Query[k]
		if !exists || actual != v {
			return nil, "", deny
		}
		values.Set(k, v)
	}
	return values, request.Path, nil
}

func tmdbExtensionMetadata(raw json.RawMessage, s video.Selection) (Metadata, error) {
	var detail video.Details
	invalid := errors.New("Fiche ou références du module TMDB invalides.")
	if protocol.DecodeOne(raw, &detail) != nil || detail.Validate() != nil {
		return Metadata{}, invalid
	}
	if detail.Season != s.Season || detail.Episode != s.Episode || (s.Kind == "series" && detail.Series == "") || (s.Kind == "films" && detail.Series != "") {
		return Metadata{}, invalid
	}
	validImage := func(ref string) bool { return ref == "" || artworkPath.MatchString(ref) }
	if !validImage(detail.ArtworkRef) {
		return Metadata{}, invalid
	}
	var m Metadata
	// Same field names for the common document, but all provenance and asset URLs
	// are created below by the host, never trusted from the extension.
	if json.Unmarshal(raw, &m) != nil {
		return Metadata{}, invalid
	}
	m.ArtworkRef = detail.ArtworkRef
	for i, person := range detail.Contributors {
		id, e := strconv.Atoi(person.ID)
		if e != nil || id <= 0 || id > 2147483647 || !validImage(person.Image) {
			return Metadata{}, invalid
		}
		m.Contributors[i].Image = assetURL("tmdb", person.Image)
	}
	for i, season := range detail.Seasons {
		if !validImage(season.Image) {
			return Metadata{}, invalid
		}
		m.Seasons[i].Image = assetURL("tmdb", season.Image)
	}
	for i, trailer := range detail.Trailers {
		key := strings.TrimPrefix(trailer.URL, "https://www.youtube.com/watch?v=")
		if !youtubeKey.MatchString(key) || trailer.URL != "https://www.youtube.com/watch?v="+key || trailer.Thumbnail != "yt:"+key {
			return Metadata{}, invalid
		}
		m.Trailers[i].Thumbnail = assetURL("tmdb", trailer.Thumbnail)
	}
	m.Provider = "tmdb"
	m.ProviderID = s.ID
	m.MatchState = "matched"
	m.DetailVersion = 2
	m.Origins = map[string]string{"title": "tmdb", "year": "tmdb", "overview": "tmdb", "cast": "tmdb", "genres": "tmdb", "image": "tmdb"}
	return m, nil
}
