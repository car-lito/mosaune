package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	_ "image/jpeg"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type imageProviderFixture struct {
	peopleProviderFixture
	id     string
	reject bool
}

func (p *imageProviderFixture) Info() ProviderInfo {
	return ProviderInfo{ID: p.id, Name: "Image source", Kinds: []string{"films", "music"}}
}
func (p *imageProviderFixture) ImageReferenceKinds(ref string) []string {
	if p.reject {
		return nil
	}
	switch ref {
	case "album:1", "album:2", "album:late", "album:bad":
		return []string{"music"}
	case "film:1":
		return []string{"films"}
	case "shared:1":
		return []string{"music", "films"}
	case "undeclared":
		return []string{"music", "books"}
	case "duplicate":
		return []string{"music", "music"}
	}
	return nil
}
func (p *imageProviderFixture) Artwork(_ context.Context, _, ref string) ([]byte, error) {
	p.calls++
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	if ref == "album:bad" {
		return []byte("not an image"), nil
	}
	return testPicture(), nil
}

type claimedImageFixture struct{ MetadataProvider }

func (*claimedImageFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "claim-images", Kinds: []string{"music"}, ImageReferences: true}
}

func TestProviderImagesReferencesPermissionsAndCache(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := &imageProviderFixture{id: "alternate-images"}
	other := &imageProviderFixture{id: "second-images"}
	c.providers = append(c.providers, p, other, &claimedImageFixture{})
	w := admin.request("GET", "/api/v1/metadata/providers", nil)
	var infos []ProviderInfo
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &infos) != nil {
		t.Fatal(w.Code)
	}
	found := false
	for _, info := range infos {
		if info.ID == p.id {
			found = info.ImageReferences
		}
		if info.ID == "claim-images" && info.ImageReferences {
			t.Fatal("unimplemented image contract advertised")
		}
	}
	if !found {
		t.Fatal("image contract missing")
	}
	for _, name := range []string{"Music", "Films"} {
		kind := "music"
		if name == "Films" {
			kind = "films"
		}
		w = admin.request("POST", "/api/v1/accounts", map[string]any{"account": account{Name: name, Kinds: []string{kind}, Libraries: map[string]libraryGrant{}, Theme: "light"}, "password": accountTestPassword})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	music := &accountClient{t: t, h: h}
	music.enter("login", map[string]string{"name": "Music", "password": accountTestPassword})
	film := &accountClient{t: t, h: h}
	film.enter("login", map[string]string{"name": "Films", "password": accountTestPassword})
	route := assetURL(p.id, "album:1")
	w = music.request("GET", route, nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" || !strings.Contains(w.Header().Get("Cache-Control"), "private") {
		t.Fatal(w.Code, w.Header())
	}
	if _, format, err := image.Decode(bytes.NewReader(w.Body.Bytes())); err != nil || format != "jpeg" {
		t.Fatal("not a normalized image", format, err)
	}
	before := p.calls
	if w = film.request("GET", route, nil); w.Code != 404 || p.calls != before {
		t.Fatal("cached music image exposed to film-only account", w.Code)
	}
	if w = music.request("GET", assetURL(p.id, "film:1"), nil); w.Code != 404 || p.calls != before {
		t.Fatal("film image exposed to music-only account", w.Code)
	}
	if w = film.request("GET", assetURL(p.id, "film:1"), nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	for _, client := range []*accountClient{music, film} {
		if w = client.request("GET", assetURL(p.id, "shared:1"), nil); w.Code != 200 {
			t.Fatal("shared image refused", w.Code)
		}
	}
	before = p.calls
	for _, ref := range []string{"", "https://127.0.0.1/private", "file:///secret", "../secret", "undeclared", "duplicate", strings.Repeat("a", 601)} {
		if w = admin.request("GET", "/api/v1/metadata/images/"+p.id+"?ref="+url.QueryEscape(ref), nil); w.Code != 404 || p.calls != before {
			t.Fatal("invalid reference invoked artwork", ref, w.Code)
		}
	}
	if w = admin.request("GET", assetURL("claim-images", "album:1"), nil); w.Code != 404 {
		t.Fatal("claim-only provider exposed", w.Code)
	}
	p.reject = true
	if w = music.request("GET", route, nil); w.Code != 404 || p.calls != before {
		t.Fatal("cache bypassed changed reference policy", w.Code)
	}
	p.reject = false
	if _, err := c.db.Exec("UPDATE provider_cache SET expires=0 WHERE provider=? AND key=?", p.id, "visual-image:v2:album:1"); err != nil {
		t.Fatal(err)
	}
	_ = p.setProviderEnabled(context.Background(), false)
	if w = music.request("GET", route, nil); w.Code != 200 || p.calls != before {
		t.Fatal("disabled provider lost cached image", w.Code)
	}
	if w = music.request("GET", assetURL(p.id, "album:2"), nil); w.Code != 404 || p.calls != before {
		t.Fatal("disabled provider fetched missing image", w.Code)
	}
	if w = music.request("GET", assetURL(other.id, "album:1"), nil); w.Code != 200 || other.calls != 1 {
		t.Fatal("provider caches mixed", w.Code, other.calls)
	}
	_ = p.setProviderEnabled(context.Background(), true)
	if w = music.request("GET", assetURL(p.id, "album:bad"), nil); w.Code != 404 {
		t.Fatal("invalid bytes served as an image", w.Code)
	}
	if providerImageAllowed(testContext(context.Background()), p, "album:1", func(kind string) bool { return kind == "films" }) {
		t.Fatal("inactive kind accepted")
	}
}

func TestProviderImagesLateResponseIsNotStored(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := &imageProviderFixture{id: "late-images"}
	p.entered, p.release = make(chan struct{}), make(chan struct{})
	c.providers = append(c.providers, p)
	complete := make(chan *httptest.ResponseRecorder, 1)
	go func() { complete <- admin.request("GET", assetURL(p.id, "album:late"), nil) }()
	select {
	case <-p.entered:
	case <-time.After(3 * time.Second):
		close(p.release)
		t.Fatal("artwork not reached")
	}
	_ = p.setProviderEnabled(context.Background(), false)
	_ = p.setProviderEnabled(context.Background(), true)
	close(p.release)
	select {
	case w := <-complete:
		if w.Code != 404 {
			t.Fatal("late image served", w.Code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request stuck")
	}
	var n int
	if err := c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE provider=?", p.id).Scan(&n); err != nil || n != 0 {
		t.Fatal("late image cached", n, err)
	}
}

func TestProviderImagesBuiltInReferencePolicies(t *testing.T) {
	tmdb := &tmdbProvider{}
	mb := &musicBrainzProvider{}
	for _, ref := range []string{"/picture.jpg", "/season_1.png", "yt:abcdefghijk"} {
		if len(tmdb.ImageReferenceKinds(ref)) == 0 {
			t.Fatal(ref)
		}
	}
	for _, ref := range []string{visualRelease, visualRelease + "/" + visualGroup, "commons:Portrait de l’artiste.jpg"} {
		if len(mb.ImageReferenceKinds(ref)) == 0 {
			t.Fatal(ref)
		}
	}
	for _, ref := range []string{"https://localhost/p.jpg", "//localhost/p.jpg", "/../picture.jpg", "yt:bad", "commons:../file.jpg", "", strings.Repeat("x", 601)} {
		if len(tmdb.ImageReferenceKinds(ref)) != 0 || len(mb.ImageReferenceKinds(ref)) != 0 {
			t.Fatal("invalid reference accepted", ref)
		}
	}
}
