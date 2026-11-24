package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

type peopleProviderFixture struct {
	MetadataProvider
	state            providerLifecycle
	calls            int
	entered, release chan struct{}
}

func (p *peopleProviderFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "other-people", Name: "Other people", Kinds: []string{"films", "music"}}
}
func (p *peopleProviderFixture) lifecycle() *providerLifecycle { return &p.state }
func (p *peopleProviderFixture) setProviderEnabled(_ context.Context, enabled bool) error {
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	p.state.disabled = !enabled
	if !enabled {
		for _, cancel := range p.state.calls {
			cancel(errProviderDisabled)
		}
	}
	return nil
}
func (p *peopleProviderFixture) ResolveArtist(ctx context.Context, name string) ([]Contributor, error) {
	p.calls++
	if p.entered != nil {
		close(p.entered)
		<-p.release
	} // Deliberately ignores cancellation: the host must discard late results.
	return []Contributor{{ID: "artist-42", Name: name}}, nil
}
func (p *peopleProviderFixture) Profile(_ context.Context, _ string, id string, page int) (PersonProfile, error) {
	return PersonProfile{ID: id, Name: "Fixture artist", Page: page, Pages: 1}, nil
}

type claimedPeopleFixture struct{ MetadataProvider }

func (p *claimedPeopleFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "claim-only", Profiles: true, ResolveArtists: true, Kinds: []string{"music"}}
}

func TestProviderPeopleCapabilitiesPermissionsAndLifecycle(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := &peopleProviderFixture{}
	c.providers = append(c.providers, p, &claimedPeopleFixture{})
	w := admin.request("GET", "/api/v1/metadata/providers", nil)
	var infos []ProviderInfo
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &infos) != nil {
		t.Fatal(w.Code)
	}
	found := false
	for _, info := range infos {
		if info.ID == "other-people" {
			found = true
			if !info.Profiles || !info.ResolveArtists || !info.Available {
				t.Fatal(info)
			}
		}
		if info.ID == "claim-only" && (info.Profiles || info.ResolveArtists) {
			t.Fatal("unimplemented capability advertised")
		}
	}
	if !found {
		t.Fatal("alternate resolver missing")
	}
	route := "/api/v1/people/other-people/resolve"
	w = admin.request("POST", route, map[string]string{"name": "Fixture artist"})
	if w.Code != 200 || p.calls != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	var people []Contributor
	if json.Unmarshal(w.Body.Bytes(), &people) != nil || len(people) != 1 || people[0].ID != "artist-42" {
		t.Fatal(w.Body.String())
	}
	w = admin.request("GET", "/api/v1/people/other-people/artist-42?page=1", nil)
	var profile PersonProfile
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &profile) != nil || profile.ID != people[0].ID {
		t.Fatal(w.Code, w.Body.String())
	}
	// The provider supports films as well, but resolving artists still requires music access.
	w = admin.request("POST", "/api/v1/accounts", map[string]any{"account": account{Name: "FilmOnly", Kinds: []string{"films"}, Libraries: map[string]libraryGrant{}, Theme: "light"}, "password": accountTestPassword})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "FilmOnly", "password": accountTestPassword})
	if w = reader.request("POST", route, map[string]string{"name": "Fixture artist"}); w.Code != 403 || p.calls != 1 {
		t.Fatal("music rights bypassed", w.Code)
	}
	if err := p.setProviderEnabled(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if w = admin.request("POST", route, map[string]string{"name": "Fixture artist"}); w.Code != 409 || p.calls != 1 {
		t.Fatal("disabled provider invoked", w.Code)
	}
	_ = p.setProviderEnabled(context.Background(), true)
	p.entered, p.release = make(chan struct{}), make(chan struct{})
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() { completed <- admin.request("POST", route, map[string]string{"name": "Fixture artist"}) }()
	select {
	case <-p.entered:
	case <-time.After(3 * time.Second):
		close(p.release)
		t.Fatal("resolver not reached")
	}
	_ = p.setProviderEnabled(context.Background(), false)
	_ = p.setProviderEnabled(context.Background(), true)
	close(p.release)
	select {
	case result := <-completed:
		if result.Code != 409 {
			t.Fatal("old result revived after reactivation", result.Code, result.Body.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("resolver did not finish")
	}
}
