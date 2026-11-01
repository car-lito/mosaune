package main

import (
	"context"
	discovery "mosaune/extensionkit/discovery/v1"
	"testing"
)

type surfaceFixture struct {
	discoveryFixture
	surface  discovery.Surface
	requests int
	selector string
	state    providerLifecycle
}

func (p *surfaceFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "custom-source", Name: "Source indépendante", Kinds: []string{"films", "music"}, ContractVersion: 1}
}
func (p *surfaceFixture) DiscoverySurface() discovery.Surface { return p.surface }
func (p *surfaceFixture) lifecycle() *providerLifecycle       { return &p.state }
func (p *surfaceFixture) Discover(context.Context, string, DiscoveryQuery) (DiscoveryPage, error) {
	p.requests++
	return DiscoveryPage{Page: 1, Pages: 1, Items: []Metadata{}}, nil
}
func (p *surfaceFixture) Shelf(_ context.Context, _ string, _ string, selector string) (DiscoveryPage, error) {
	p.requests++
	p.selector = selector
	return DiscoveryPage{Page: 1, Pages: 1, Items: []Metadata{}}, nil
}
func TestDiscoveryDeclaredSurfaceRouting(t *testing.T) {
	c, _ := testCatalog(t)
	p := &surfaceFixture{surface: discovery.Surface{Version: 1, Search: []discovery.Search{{Kind: "films", Label: "Films", Placeholder: "Find films"}}, Shelves: []discovery.Shelf{{ID: "festival", Kind: "films", Label: "Festival", Selector: "kind"}, {ID: "next", Kind: "films", Label: "Cast projects", Selector: "contributors"}}}}
	c.providers = []MetadataProvider{p}
	h, token := handlerFor(t, c, "video,music")
	request := func(route string, body map[string]any, status int) {
		t.Helper()
		r := call(t, h, "POST", route, token, body)
		if r.Code != status {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	search := map[string]any{"provider": "custom-source", "kind": "films", "query": "abc", "page": 1}
	request("/api/v1/discovery/browse", search, 200)
	search["kind"] = "music"
	request("/api/v1/discovery/browse", search, 400)
	search["kind"] = "films"
	search["query"] = ""
	search["seed"] = "not-needed"
	request("/api/v1/discovery/browse", search, 400)
	body := map[string]any{"provider": "custom-source", "kind": "films", "shelf": "festival"}
	request("/api/v1/shelves", body, 200)
	if p.selector != "films" {
		t.Fatal("wrong kind selector")
	}
	body["shelf"] = "next"
	request("/api/v1/shelves", body, 400)
	body["artist"] = "known-contributor"
	request("/api/v1/shelves", body, 200)
	if p.selector != "known-contributor" {
		t.Fatal("provider identity guessed")
	}
	body["shelf"] = "private"
	request("/api/v1/shelves", body, 400)
	if p.requests != 3 {
		t.Fatal("undeclared operation reached provider", p.requests)
	}
	p.state.disabled = true
	body["shelf"] = "festival"
	request("/api/v1/shelves", body, 409)
	if p.requests != 3 {
		t.Fatal("disabled provider called")
	}
	p.state.disabled = false
	p.surface.Version = 2
	request("/api/v1/shelves", body, 400)
	info := c.providerInfo(testContext(context.Background()))[0]
	if info.Discovery || info.Catalogue == nil || len(info.Catalogue.Search) != 0 {
		t.Fatal("invalid declaration regained search", info)
	}
	p.surface = discovery.Surface{Version: 1}
	search["query"] = "abc"
	delete(search, "seed")
	request("/api/v1/discovery/browse", search, 400)
}
func TestDiscoverySurfaceDefaultsAndIndependentFeed(t *testing.T) {
	for _, p := range []MetadataProvider{newTMDBProvider(nil), newMusicBrainzProvider(nil)} {
		s, e := providerDiscoverySurface(p)
		if e != nil || s == nil || len(s.Search) == 0 || len(s.Shelves) == 0 {
			t.Fatal(p.Info().ID, s, e)
		}
	}
	old := &discoveryFixture{}
	if !providerSearchAllowed(old, "films", false) {
		t.Fatal("legacy search lost")
	}
	if !validUIFeed(uiBlock{Type: "feed", Source: "remote", View: "films", Provider: "org.independent.source", Shelf: "festival"}) {
		t.Fatal("contributor requires host whitelist")
	}
	if validUIFeed(uiBlock{Type: "feed", Source: "remote", View: "films", Provider: "https://bad", Shelf: "festival"}) {
		t.Fatal("URL accepted")
	}
}
