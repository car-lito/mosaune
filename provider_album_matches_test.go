package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type albumMatcherFixture struct {
	peopleProviderFixture
	malformed bool
}

func (p *albumMatcherFixture) MatchAlbums(ctx context.Context, _ string, q AlbumMatchQuery) (AlbumMatchResult, error) {
	p.calls++
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	if q.ID == "invalid" {
		return AlbumMatchResult{}, errAlbumMatchInput
	}
	id := "release-42"
	title := "Alternate album"
	if q.ID != "" {
		id = q.ID
		title = "Detailed album"
	}
	result := AlbumMatchResult{Page: DiscoveryPage{Page: q.Page, Pages: q.Page, Items: []Metadata{{Provider: "musicbrainz", ProviderID: id, Title: title, Tracks: []CatalogTrack{{ID: "track-1", Title: "Track", Number: 1}}}}}, Detailed: q.ID != ""}
	if p.malformed {
		result.Page.Pages = 1000
	}
	return result, nil
}

type claimedAlbumMatcher struct{ MetadataProvider }

func (p *claimedAlbumMatcher) Info() ProviderInfo {
	return ProviderInfo{ID: "claim-albums", Kinds: []string{"music"}, AlbumMatches: true}
}

func TestProviderAlbumMatchesIndependentSource(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := &albumMatcherFixture{}
	c.providers = append(c.providers, p, &claimedAlbumMatcher{})
	w := admin.request("GET", "/api/v1/metadata/providers", nil)
	var infos []ProviderInfo
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &infos) != nil {
		t.Fatal(w.Code, w.Body)
	}
	found := false
	for _, info := range infos {
		if info.ID == "other-people" {
			found = info.AlbumMatches
		}
		if info.ID == "claim-albums" && info.AlbumMatches {
			t.Fatal("unimplemented matcher exposed")
		}
	}
	if !found {
		t.Fatal("matcher missing")
	}
	route := "/api/v1/music/matches/other-people"
	w = admin.request("POST", route, map[string]any{"title": "Album", "artist": "Artist", "page": 2})
	var result struct {
		Items []discoveryWork `json:"items"`
		Page  int             `json:"page"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || len(result.Items) != 1 || result.Page != 2 {
		t.Fatal(w.Code, w.Body)
	}
	work := result.Items[0]
	if work.Metadata.Provider != "other-people" || work.Metadata.ProviderID != "release-42" {
		t.Fatal("foreign provider claimed", work)
	}
	w = admin.request("POST", route, map[string]string{"id": "release-42"})
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Detailed album") {
		t.Fatal("direct lookup failed to replace cached summary", w.Code, w.Body)
	}
	saved, err := c.work(context.Background(), work.ID)
	if err != nil || !saved.detailed || len(saved.Metadata.Tracks) != 1 {
		t.Fatal(saved, err)
	}
	for _, q := range []map[string]any{{"title": "A"}, {"title": "Album", "page": 51}, {"id": strings.Repeat("a", 201)}} {
		before := p.calls
		if w = admin.request("POST", route, q); w.Code != 400 || p.calls != before {
			t.Fatal("invalid input invoked matcher", w.Code)
		}
	}
	if w = admin.request("POST", route, map[string]string{"id": "invalid"}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	for _, id := range []string{"claim-albums", "missing"} {
		if w = admin.request("POST", "/api/v1/music/matches/"+id, map[string]string{"title": "Album"}); w.Code != 404 {
			t.Fatal(id, w.Code)
		}
	}
	p.malformed = true
	if w = admin.request("POST", route, map[string]string{"title": "Album"}); w.Code != 502 {
		t.Fatal("invalid page accepted", w.Code)
	}
	p.malformed = false
	w = admin.request("POST", "/api/v1/accounts", map[string]any{"account": account{Name: "Music", Kinds: []string{"music"}, Libraries: map[string]libraryGrant{}, Theme: "light"}, "password": accountTestPassword})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Music", "password": accountTestPassword})
	if w = reader.request("POST", route, map[string]string{"title": "Album"}); w.Code != 200 {
		t.Fatal("music account rejected", w.Code, w.Body)
	}
	w = admin.request("POST", "/api/v1/accounts", map[string]any{"account": account{Name: "Films", Kinds: []string{"films"}, Libraries: map[string]libraryGrant{}, Theme: "light"}, "password": accountTestPassword})
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	film := &accountClient{t: t, h: h}
	film.enter("login", map[string]string{"name": "Films", "password": accountTestPassword})
	before := p.calls
	if w = film.request("POST", route, map[string]string{"title": "Album"}); w.Code != 403 || p.calls != before {
		t.Fatal("music permissions bypassed", w.Code)
	}
	_ = p.setProviderEnabled(context.Background(), false)
	if w = admin.request("POST", route, map[string]string{"title": "Album"}); w.Code != 409 || p.calls != before {
		t.Fatal("disabled matcher called", w.Code)
	}
	_ = p.setProviderEnabled(context.Background(), true)
	p.entered, p.release = make(chan struct{}), make(chan struct{})
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() { completed <- admin.request("POST", route, map[string]string{"id": "late-release"}) }()
	select {
	case <-p.entered:
	case <-time.After(3 * time.Second):
		close(p.release)
		t.Fatal("matcher not reached")
	}
	_ = p.setProviderEnabled(context.Background(), false)
	_ = p.setProviderEnabled(context.Background(), true)
	close(p.release)
	select {
	case w = <-completed:
		if w.Code != 409 {
			t.Fatal("late result accepted", w.Code, w.Body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("matcher stuck")
	}
	var n int
	if err = c.db.QueryRow(`SELECT COUNT(*) FROM works WHERE id=?`, workID("music", "other-people", "late-release")).Scan(&n); err != nil || n != 0 {
		t.Fatal("late result persisted", n, err)
	}
}
