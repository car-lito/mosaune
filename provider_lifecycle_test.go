package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTMDBDisableAllCapabilitiesAndEngines(t *testing.T) {
	for _, wasm := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "wasm"}[wasm], func(t *testing.T) {
			c, _ := testCatalog(t)
			p := c.provider("tmdb").(*tmdbProvider)
			ctx := context.Background()
			var calls atomic.Int32
			p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"id":42,"title":"Voyage"}]}`)), Header: make(http.Header)}, nil
			})}
			if err := p.setExtension(ctx, wasm); err != nil {
				t.Fatal(err)
			}
			q := MetadataQuery{Kind: "films", Title: "Voyage"}
			if rows, err := p.Search(ctx, "fixture", q); err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
			if err := p.setProviderEnabled(ctx, false); err != nil {
				t.Fatal(err)
			}
			before := calls.Load()
			checks := []func() error{
				func() error { _, e := p.Search(ctx, "fixture", q); return e },
				func() error { _, e := p.Details(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}); return e },
				func() error {
					_, e := p.Discover(ctx, "fixture", DiscoveryQuery{Kind: "films", Query: "Voyage", Page: 1})
					return e
				},
				func() error { _, e := p.Artwork(ctx, "fixture", "/picture.jpg"); return e },
				func() error { _, e := p.Artwork(ctx, "fixture", "yt:abcdefghijk"); return e },
				func() error { _, e := p.Shelf(ctx, "fixture", "cinema", "films"); return e },
				func() error { _, e := p.Profile(ctx, "fixture", "42", 1); return e },
				func() error {
					_, e := p.Posters(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}, "fr")
					return e
				},
			}
			for i, check := range checks {
				if e := check(); !errors.Is(e, errProviderDisabled) {
					t.Fatalf("operation %d: %v", i, e)
				}
			}
			if _, e := p.AcquisitionID(ctx, "fixture", "series", "42"); !errors.Is(e, errProviderDisabled) {
				t.Fatal("series ID lookup bypass", e)
			}
			if id, e := p.AcquisitionID(ctx, "fixture", "films", "42"); e != nil || id != 42 {
				t.Fatal("local film ID unavailable", e)
			}
			if calls.Load() != before {
				t.Fatal("disabled provider contacted network")
			}
			if p.extensionStatus()["loaded"] || p.extensionStatus()["enabled"] != wasm {
				t.Fatal("engine memory/preference")
			}
			// Engine selection, safe mode and cached responses cannot reactivate a provider.
			if err := p.setExtension(ctx, !wasm); err != nil {
				t.Fatal(err)
			}
			c.extensionSafeMode = true
			if _, e := p.Search(ctx, "fixture", q); !errors.Is(e, errProviderDisabled) {
				t.Fatal(e)
			}
			c.extensionSafeMode = false
			if err := p.setProviderEnabled(ctx, true); err != nil {
				t.Fatal(err)
			}
			if rows, err := p.Search(ctx, "fixture", q); err != nil || len(rows) != 1 {
				t.Fatal("reenable", rows, err)
			}
			if calls.Load() != before {
				t.Fatal("cache lost")
			}
			if !providerEnabled(c.provider("musicbrainz")) {
				t.Fatal("other provider disabled")
			}
		})
	}
}

func TestTMDBDisableInterruptsWorkerWithoutCountingFailure(t *testing.T) {
	for _, wasm := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "wasm"}[wasm], func(t *testing.T) {
			c, _ := testCatalog(t)
			p := c.provider("tmdb").(*tmdbProvider)
			ctx := testContext(context.Background())
			dir := t.TempDir()
			putFile(t, dir, "Voyage.2024.mp4", "fixture")
			if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
				t.Fatal(e)
			}
			if e := c.setProviderCredential(ctx, "tmdb", "fixture-token-123456789"); e != nil {
				t.Fatal(e)
			}
			if e := p.setExtension(ctx, wasm); e != nil {
				t.Fatal(e)
			}
			entered := make(chan struct{}, 1)
			p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
				entered <- struct{}{}
				<-r.Context().Done()
				return nil, r.Context().Err()
			})}
			done := make(chan struct{})
			go func() { c.metadataStep(ctx, allKinds); close(done) }()
			select {
			case <-entered:
			case <-time.After(15 * time.Second):
				t.Fatal("worker did not start")
			}
			if e := p.setProviderEnabled(ctx, false); e != nil {
				t.Fatal(e)
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("worker did not stop")
			}
			s := queueState(t, c)
			if s.Libraries[0].Pending != 1 || s.Libraries[0].Failed != 0 || s.RetryAt != 0 || s.Current != "" {
				t.Fatal("disable consumed an attempt", s)
			}
			// A later activation can identify the same file without clearing errors.
			p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
				body := `{"results":[{"id":42,"title":"Voyage","release_date":"2024-01-01"}]}`
				if strings.Contains(r.URL.Path, "/movie/42") {
					body = `{"id":42,"title":"Voyage","release_date":"2024-01-01","videos":{"results":[]}}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			if e := p.setProviderEnabled(ctx, true); e != nil {
				t.Fatal(e)
			}
			c.metadataStep(ctx, allKinds)
			s = queueState(t, c)
			if s.Libraries[0].Matched != 1 || s.Libraries[0].Failed != 0 {
				t.Fatal("queue did not resume", s)
			}
		})
	}
}

func TestTMDBDisableCancelsLateResultsAndPersistence(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := testContext(context.Background())
	entered := make(chan context.Context, 1)
	release := make(chan struct{})
	defer close(release)
	p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		entered <- r.Context()
		<-release
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"id":42,"title":"Voyage"}]}`)), Header: make(http.Header)}, nil
	})}
	result := make(chan error, 1)
	go func() { _, e := p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Voyage"}); result <- e }()
	var request context.Context
	select {
	case request = <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}
	if e := p.setProviderEnabled(ctx, false); e != nil {
		t.Fatal(e)
	}
	select {
	case <-request.Done():
	case <-time.After(time.Second):
		t.Fatal("request not cancelled")
	}
	if e := p.setProviderEnabled(ctx, true); e != nil {
		t.Fatal(e)
	}
	// Release the deliberately uncooperative transport after reactivation.
	release <- struct{}{}
	select {
	case e := <-result:
		if !errors.Is(e, errProviderDisabled) {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("late call stuck")
	}
	var count int
	if e := c.db.QueryRow(`SELECT COUNT(*) FROM provider_cache WHERE provider='tmdb'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("late response cached", count, e)
	}
	lease, done, e := beginProvider(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	defer done()
	if e = p.setProviderEnabled(ctx, false); e != nil {
		t.Fatal(e)
	}
	if e = p.setProviderEnabled(ctx, true); e != nil {
		t.Fatal(e)
	}
	if _, e = c.rememberWork(lease, "films", Metadata{Provider: "tmdb", ProviderID: "42", Title: "Late"}, true); !errors.Is(e, errProviderDisabled) {
		t.Fatal("late discovery committed", e)
	}
}

func TestTMDBDisableKeepsCachedImagesAndPendingQueue(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "Voyage.2024.mp4", "fixture")
	if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	imageID, e := storeArtwork(ctx, c.db, testPicture())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.db.Exec(`INSERT INTO provider_cache(provider,key,body,expires) VALUES('tmdb','visual-image:v2:/picture.jpg',?,1)`, []byte(imageID)); e != nil {
		t.Fatal(e)
	}
	if e = p.setProviderEnabled(ctx, false); e != nil {
		t.Fatal(e)
	}
	p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		t.Error("disabled network access")
		return nil, errors.New("unexpected network")
	})}
	if data, e := c.visualImage(ctx, p, "/picture.jpg"); e != nil || len(data) == 0 {
		t.Fatal("stored image lost", e)
	}
	if _, e := c.visualImage(ctx, p, "/missing.jpg"); !errors.Is(e, errProviderDisabled) {
		t.Fatal("missing image", e)
	}
	c.metadataStep(ctx, func(string) bool { return true })
	status, e := c.metadataStatus(ctx, func(string) bool { return true })
	if e != nil || len(status.Libraries) != 1 || status.Libraries[0].Pending != 1 || status.Libraries[0].Failed != 0 {
		t.Fatal(status, e)
	}
}

func TestTMDBDisablePermissionsMigrationAndRestart(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	endpoint := "/api/v1/metadata/providers/tmdb/state"
	if w := reader.request("PUT", endpoint, map[string]bool{"enabled": false}); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := admin.request("PUT", endpoint, map[string]bool{}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := admin.request("PUT", endpoint, map[string]bool{"enabled": false}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if providerEnabled(c.provider("tmdb")) {
		t.Fatal("API did not disable")
	}
	old, dir := testCatalog(t)
	removeRecognitionForLegacyFixture(t, old)
	if _, e := old.db.Exec(`DROP TABLE provider_states; PRAGMA user_version=13`); e != nil {
		t.Fatal(e)
	}
	old.close()
	migrated, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	p := migrated.provider("tmdb").(*tmdbProvider)
	if !providerEnabled(p) {
		t.Fatal("migration confused native engine with disabled provider")
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if len(backups) != 1 {
		t.Fatal("migration backup missing")
	}
	if e = p.setProviderEnabled(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	migrated.close()
	restarted, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer restarted.close()
	if providerEnabled(restarted.provider("tmdb")) {
		t.Fatal("disable lost after restart")
	}
}
