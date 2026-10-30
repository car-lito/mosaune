package main

import (
	"context"
	"encoding/json"
	"errors"
	videocatalog "mosaune/extensionkit/capabilities/videocatalog/v1"
	"mosaune/extensionkit/protocol"
	bundled "mosaune/extensionkit/providers/tmdb"
	runtime "mosaune/extensionkit/runtime"
	"net/url"
	"strconv"
)

func validTMDBCatalogueID(id string) bool {
	n, e := strconv.Atoi(id)
	return e == nil && n > 0 && n <= 2147483647 && strconv.Itoa(n) == id
}

// The host bounds the only network operation; the guest cannot change destination,
// query, page, language or credentials. No generic HTTP capability is granted.
func tmdbCatalogueRequest(op string, q videocatalog.Query) (string, url.Values, error) {
	deny := errors.New("Requête de catalogue TMDB invalide.")
	if q.Validate(op) != nil || q.ID != "" && !validTMDBCatalogueID(q.ID) {
		return "", nil, deny
	}
	params := url.Values{"language": {q.Language}}
	base := "/" + tmdbKind(q.Kind)
	switch op {
	case "discover":
		params.Set("page", strconv.Itoa(q.Page))
		params.Set("include_adult", "false")
		if q.ID != "" {
			return base + "/" + q.ID + "/recommendations", params, nil
		}
		params.Set("query", q.Text)
		return "/search" + base, params, nil
	case "shelf":
		if !(q.Kind == "films" && (q.Shelf == "now_playing" || q.Shelf == "upcoming") || q.Kind == "series" && q.Shelf == "popular") {
			return "", nil, deny
		}
		params.Set("region", q.Region)
		params.Set("page", "1")
		return base + "/" + q.Shelf, params, nil
	case "profile":
		params.Set("append_to_response", "combined_credits")
		return "/person/" + q.ID, params, nil
	case "posters":
		params.Set("language", "")
		return base + "/" + q.ID + "/images", params, nil
	}
	return "", nil, deny
}
func tmdbCatalogueGrant(operation string, raw json.RawMessage, op string, q videocatalog.Query) (string, url.Values, error) {
	path, params, e := tmdbCatalogueRequest(op, q)
	if e != nil {
		return "", nil, e
	}
	var in struct {
		Path  string            `json:"path"`
		Query map[string]string `json:"query"`
	}
	deny := errors.New("TMDB catalogue request denied")
	if operation != "get" || protocol.DecodeOne(raw, &in) != nil || in.Path != path || len(in.Query) != len(params) {
		return "", nil, deny
	}
	for k, vs := range params {
		v, ok := in.Query[k]
		if !ok || v != vs[0] {
			return "", nil, deny
		}
	}
	return path, params, nil
}
func decodeTMDBCatalogue(raw json.RawMessage, op string, q videocatalog.Query) (videocatalog.Result, error) {
	var result videocatalog.Result
	bad := errors.New("Réponse du catalogue TMDB invalide.")
	if protocol.DecodeOne(raw, &result) != nil || result.Validate(op, q) != nil {
		return result, bad
	}
	image := func(ref string) bool { return ref == "" || artworkPath.MatchString(ref) }
	for _, i := range result.Items {
		if !validTMDBCatalogueID(i.ID) || !image(i.ArtworkRef) {
			return result, bad
		}
	}
	if p := result.Person; p != nil {
		if !validTMDBCatalogueID(p.ID) || !image(p.ArtworkRef) {
			return result, bad
		}
	}
	for _, p := range result.Posters {
		if !image(p.Ref) {
			return result, bad
		}
	}
	return result, nil
}
func (p *tmdbProvider) catalogueExtension(ctx context.Context, secret, op string, q videocatalog.Query) (videocatalog.Result, error) {
	if _, _, err := tmdbCatalogueRequest(op, q); err != nil {
		return videocatalog.Result{}, err
	}
	engine, release, err := p.extensionEngine(ctx)
	if err != nil {
		return videocatalog.Result{}, err
	}
	defer release()
	raw, _ := json.Marshal(q)
	used := false
	var serviceErr error
	result, err := engine.CallWithServices(ctx, bundled.Descriptor().ID, protocol.Request{Protocol: 1, ID: "tmdb-catalogue", Capability: videocatalog.Capability, Operation: op, Payload: raw}, runtime.ServiceGrants{
		"tmdb.api.v1": func(call context.Context, operation string, raw json.RawMessage) (json.RawMessage, error) {
			if used {
				return nil, errors.New("request budget exhausted")
			}
			used = true
			path, params, e := tmdbCatalogueGrant(operation, raw, op, q)
			if e != nil {
				return nil, e
			}
			var body json.RawMessage
			serviceErr = p.get(call, secret, path, params, &body)
			return body, serviceErr
		}})
	if serviceErr != nil {
		return videocatalog.Result{}, serviceErr
	}
	if err != nil || result.Response.Error != nil {
		return videocatalog.Result{}, errors.New("Cette fonction nécessite le module TMDB 0.5.0 ou supérieur en état de fonctionner. Mettez-le à jour dans Extensions. Aucun autre moteur n’a été utilisé.")
	}
	out, err := decodeTMDBCatalogue(result.Response.Result, op, q)
	return out, err
}
func catalogueMetadata(i videocatalog.Item) Metadata {
	return Metadata{Title: i.Title, Year: i.Year, Overview: i.Overview, ReleaseDate: i.ReleaseDate, ArtworkRef: i.ArtworkRef, Provider: "tmdb", ProviderID: i.ID, MatchState: "matched"}
}
func cataloguePage(r videocatalog.Result) DiscoveryPage {
	out := DiscoveryPage{Items: []Metadata{}, Page: r.Page, Pages: r.Pages}
	for _, i := range r.Items {
		out.Items = append(out.Items, catalogueMetadata(i))
	}
	return out
}
func (p *tmdbProvider) activeDiscover(ctx context.Context, secret string, q DiscoveryQuery) (DiscoveryPage, error) {
	r, e := p.catalogueExtension(ctx, secret, "discover", videocatalog.Query{Kind: q.Kind, Text: q.Query, ID: q.Seed, Page: q.Page, Language: appLanguage})
	return cataloguePage(r), e
}
func (p *tmdbProvider) activeShelf(ctx context.Context, secret, shelf, kind string) (DiscoveryPage, error) {
	r, e := p.catalogueExtension(ctx, secret, "shelf", videocatalog.Query{Kind: kind, Shelf: shelf, Page: 1, Language: appLanguage, Region: "BE"})
	return cataloguePage(r), e
}
func (p *tmdbProvider) activeProfile(ctx context.Context, secret, id string, page int) (PersonProfile, error) {
	r, e := p.catalogueExtension(ctx, secret, "profile", videocatalog.Query{ID: id, Page: page, Language: appLanguage})
	if e != nil {
		return PersonProfile{}, e
	}
	person := r.Person
	out := PersonProfile{ID: id, Name: person.Name, Biography: person.Biography, Detail: person.Detail, Image: assetURL("tmdb", person.ArtworkRef), Page: r.Page, Pages: r.Pages, Works: []discoveryWork{}}
	for _, i := range r.Items {
		out.Items = append(out.Items, ProfileItem{Kind: i.Kind, Metadata: catalogueMetadata(i)})
	}
	return out, nil
}
func (p *tmdbProvider) activePosters(ctx context.Context, secret string, s MetadataSelection, language string) ([]PosterOption, error) {
	if language == "" {
		language = appLanguage
	}
	r, e := p.catalogueExtension(ctx, secret, "posters", videocatalog.Query{Kind: s.Kind, ID: s.ID, Page: 1, Language: language})
	if e != nil {
		return nil, e
	}
	out := []PosterOption{}
	for _, i := range r.Posters {
		out = append(out, PosterOption{Ref: i.Ref, Language: i.Language, Width: i.Width, Height: i.Height})
	}
	return out, nil
}
