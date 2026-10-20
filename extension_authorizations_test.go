package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mosaune/extensionkit/authorization"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
)

func TestServiceAuthorizationIsolationUpdateAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "access.db")
	db := registryDB(t, path)
	defer func() { db.Close() }()
	s := authorization.SQLStore{DB: db}
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	d := protocol.Descriptor{ID: "test.first", Version: "1.0.0", Capabilities: []string{"test.reader.v1"}, Services: []protocol.ServiceRequirement{{Name: "test.storage.v1", Operations: []string{"read"}}}}
	if err := s.Register(ctx, d); err != nil {
		t.Fatal(err)
	}
	other := d
	other.ID = "test.second"
	if err := s.Register(ctx, other); err != nil {
		t.Fatal(err)
	}
	a, _ := s.Read(ctx, d.ID)
	if a.Allows("test.storage.v1", "read") {
		t.Fatal("manifest granted access")
	}
	if err := s.Apply(ctx, d.ID, authorization.Patch{Revision: &a.Revision, Granted: d.Services}); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Read(ctx, other.ID)
	if b.Allows("test.storage.v1", "read") {
		t.Fatal("owner isolation")
	}
	if err := s.Apply(ctx, d.ID, authorization.Patch{Revision: &a.Revision}); !errors.Is(err, authorization.ErrConflict) {
		t.Fatal("stale edit accepted", err)
	}
	d.Version = "1.1.0"
	d.Services = []protocol.ServiceRequirement{{Name: "test.storage.v1", Operations: []string{"read", "write"}}}
	if err := s.Register(ctx, d); err != nil {
		t.Fatal(err)
	}
	a, _ = s.Read(ctx, d.ID)
	if !a.Allows("test.storage.v1", "read") || a.Allows("test.storage.v1", "write") {
		t.Fatal("update increased authority", a)
	}
	db.Close()
	db = registryDB(t, path)
	s.DB = db
	a, err := s.Read(ctx, d.ID)
	if err != nil || !a.Allows("test.storage.v1", "read") || a.Allows("test.storage.v1", "write") {
		t.Fatal("restart", err, a)
	}
	for _, p := range []authorization.Patch{{Granted: d.Services}, {Revision: &a.Revision, Granted: []protocol.ServiceRequirement{{Name: "test.other.v1", Operations: []string{"read"}}}}, {Revision: &a.Revision, Granted: []protocol.ServiceRequirement{{Name: "test.storage.v1", Operations: []string{"read", "read"}}}}} {
		if err = s.Apply(ctx, d.ID, p); !errors.Is(err, authorization.ErrInvalid) {
			t.Fatal("invalid access accepted", err)
		}
	}
	d.Services = nil
	if err = s.Register(ctx, d); err != nil {
		t.Fatal(err)
	}
	d.Services = other.Services
	if err = s.Register(ctx, d); err != nil {
		t.Fatal(err)
	}
	a, _ = s.Read(ctx, d.ID)
	if a.Allows("test.storage.v1", "read") {
		t.Fatal("reintroduced access automatically reapproved")
	}
	// Two administrators writing the same revision cannot overwrite each other.
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.Apply(ctx, d.ID, authorization.Patch{Revision: &a.Revision, Granted: d.Services})
		}()
	}
	wg.Wait()
	close(errs)
	ok, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			ok++
		} else if errors.Is(err, authorization.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Fatal("concurrent approval", ok, conflicts)
	}
	if _, err = db.Exec(`UPDATE extension_authorizations SET granted='[{"name":"test.storage.v1","operations":["delete"]}]' WHERE owner='test.first'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Read(ctx, d.ID); !errors.Is(err, authorization.ErrStored) {
		t.Fatal("corrupt authority trusted", err)
	}
	if err = s.Register(ctx, d); !errors.Is(err, authorization.ErrStored) {
		t.Fatal("update reset corrupt authority", err)
	}
	if b, err = s.Read(ctx, other.ID); err != nil || len(b.Granted) != 0 {
		t.Fatal("other owner damaged", err)
	}
}

func changeTMDBAccess(t *testing.T, p *tmdbProvider, allow bool) {
	t.Helper()
	ctx := context.Background()
	s, err := (authorization.SQLStore{DB: p.db}).Read(ctx, tmdbConfigurationOwner)
	if err != nil {
		t.Fatal(err)
	}
	patch := authorization.Patch{Revision: &s.Revision}
	if allow {
		patch.Granted = s.Requested
	}
	if err = p.setServiceAuthorization(ctx, patch); err != nil {
		t.Fatal(err)
	}
}
func TestServiceAuthorizationRevokesBothEnginesAndLateResults(t *testing.T) {
	for _, wasm := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "wasm"}[wasm], func(t *testing.T) {
			c, _ := testCatalog(t)
			p := c.provider("tmdb").(*tmdbProvider)
			ctx := testContext(context.Background())
			if err := p.setExtension(ctx, wasm); err != nil {
				t.Fatal(err)
			}
			entered := make(chan context.Context, 1)
			release := make(chan struct{})
			defer close(release)
			p.client = &http.Client{Transport: tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
				entered <- r.Context()
				<-release
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"results":[{"id":42,"title":"Voyage"}]}`))}, nil
			})}
			finished := make(chan error, 1)
			go func() {
				_, err := p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Voyage"})
				finished <- err
			}()
			var request context.Context
			select {
			case request = <-entered:
			case <-time.After(20 * time.Second):
				t.Fatal("call did not start")
			}
			lease, done, err := beginProvider(ctx, p)
			if err != nil {
				t.Fatal(err)
			}
			defer done()
			changeTMDBAccess(t, p, false)
			select {
			case <-request.Done():
			case <-time.After(time.Second):
				t.Fatal("revocation did not cancel request")
			}
			checks := []func() error{
				func() error { _, e := p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Other"}); return e },
				func() error { _, e := p.Details(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}); return e },
				func() error { _, e := p.Artwork(ctx, "fixture", "/image.jpg"); return e },
				func() error { _, e := p.Artwork(ctx, "fixture", "yt:abcdefghijk"); return e },
				func() error {
					_, e := p.Discover(ctx, "fixture", DiscoveryQuery{Kind: "films", Query: "Voyage", Page: 1})
					return e
				},
				func() error { _, e := p.Shelf(ctx, "fixture", "cinema", "films"); return e },
				func() error { _, e := p.Profile(ctx, "fixture", "42", 1); return e },
				func() error {
					_, e := p.Posters(ctx, "fixture", MetadataSelection{Kind: "films", ID: "42"}, "fr")
					return e
				},
			}
			for i, check := range checks {
				if err = check(); !errors.Is(err, errProviderAccessDenied) {
					t.Fatal("operation", i, err)
				}
			}
			// Provider activation and safe fallback cannot grant network permission.
			if err = p.setProviderEnabled(ctx, true); err != nil {
				t.Fatal(err)
			}
			c.extensionSafeMode = true
			if _, err = p.Search(ctx, "fixture", MetadataQuery{Kind: "films", Title: "Voyage"}); !errors.Is(err, errProviderAccessDenied) {
				t.Fatal("native fallback bypassed consent", err)
			}
			c.extensionSafeMode = false
			changeTMDBAccess(t, p, true)
			release <- struct{}{}
			select {
			case err = <-finished:
				if !errors.Is(err, errProviderAccessDenied) {
					t.Fatal("late result escaped", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("call stuck")
			}
			if _, err = c.rememberWork(lease, "films", Metadata{Provider: "tmdb", ProviderID: "42", Title: "Late"}, true); !errors.Is(err, errProviderAccessDenied) {
				t.Fatal("late catalogue commit", err)
			}
			var count int
			if err = c.db.QueryRow(`SELECT COUNT(*) FROM provider_cache WHERE provider='tmdb'`).Scan(&count); err != nil || count != 0 {
				t.Fatal("late cache commit", count, err)
			}
			if !providerAvailable(c.provider("musicbrainz")) {
				t.Fatal("other provider affected")
			}
		})
	}
}
func TestServiceAuthorizationHTTPRevisionAndRoles(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	path := "/api/v1/extensions/" + tmdbConfigurationOwner + "/authorizations"
	for _, method := range []string{"GET", "PUT"} {
		if w := reader.request(method, path, map[string]any{"revision": 1}); w.Code != 403 {
			t.Fatal("non-admin access", w.Code)
		}
	}
	response := admin.request("GET", path, nil)
	var state authorization.State
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &state) != nil {
		t.Fatal("read state", response.Code)
	}
	if w := admin.request("PUT", path, map[string]any{"granted": state.Requested}); w.Code != 400 {
		t.Fatal("missing revision", w.Code)
	}
	if w := admin.request("PUT", path, map[string]any{"revision": state.Revision, "owner": "other.module"}); w.Code != 400 {
		t.Fatal("owner override", w.Code)
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision}); w.Code != 204 {
		t.Fatal("revoke", w.Code, w.Body.String())
	}
	if w := admin.request("PUT", path, authorization.Patch{Revision: &state.Revision, Granted: state.Requested}); w.Code != 409 {
		t.Fatal("stale grant", w.Code)
	}
	if w := admin.request("GET", "/api/v1/extensions/other.module/authorizations", nil); w.Code != 404 {
		t.Fatal("unknown adapter", w.Code)
	}
	if providerAvailable(c.provider("tmdb")) {
		t.Fatal("API ineffective")
	}
}
func TestServiceAuthorizationRestartRollbackAndImages(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := testContext(context.Background())
	key := "fixture-kept-credential-1234567"
	if err := c.setProviderCredential(ctx, "tmdb", key); err != nil {
		t.Fatal(err)
	}
	imageID, err := storeArtwork(ctx, c.db, testPicture())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.db.Exec(`INSERT INTO provider_cache(provider,key,body,expires) VALUES('tmdb','visual-image:v2:/picture.jpg',?,1)`, []byte(imageID)); err != nil {
		t.Fatal(err)
	}
	lease, done, err := beginProvider(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	before, _ := (authorization.SQLStore{DB: c.db}).Read(ctx, tmdbConfigurationOwner)
	if _, err = c.db.Exec(`CREATE TEMP TRIGGER fail_access BEFORE UPDATE ON extension_authorizations BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = p.setServiceAuthorization(ctx, authorization.Patch{Revision: &before.Revision}); err == nil {
		t.Fatal("failed DB write succeeded")
	}
	if context.Cause(lease) != nil || !providerAvailable(p) {
		t.Fatal("failed write revoked running operation")
	}
	if _, err = c.db.Exec(`DROP TRIGGER fail_access`); err != nil {
		t.Fatal(err)
	}
	changeTMDBAccess(t, p, false)
	if data, err := c.visualImage(ctx, p, "/picture.jpg"); err != nil || len(data) == 0 {
		t.Fatal("saved image lost", err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	if providerAvailable(p) || c.credential(ctx, "tmdb") != key {
		t.Fatal("restart lost denial or key")
	}
	// Install, update, rollback and removal preserve consent shared with native.
	first, err := packageformat.Parse(providerExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	next, err := packageformat.Parse(providerUpdateExample, trustedProviderKey(), version)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, first, "", false); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, next, first.Checksum, false); err != nil {
		t.Fatal(err)
	}
	if err = p.installPackage(ctx, first, next.Checksum, true); err != nil {
		t.Fatal(err)
	}
	if err = p.removePackage(ctx, first.Checksum); err != nil {
		t.Fatal(err)
	}
	state, err := (authorization.SQLStore{DB: c.db}).Read(ctx, tmdbConfigurationOwner)
	if err != nil || state.Allows("tmdb.api.v1", "get") || providerAvailable(p) {
		t.Fatal("package lifecycle restored denied access", err)
	}
}

func TestServiceAuthorizationInterruptedScanStaysPending(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	putFile(t, dir, "Voyage.2024.mp4", "fixture")
	if _, err := c.scan(ctx, library{Path: dir, Kind: "films"}, true); err != nil {
		t.Fatal(err)
	}
	if err := c.setProviderCredential(ctx, "tmdb", "fixture-token-123456789"); err != nil {
		t.Fatal(err)
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
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not start")
	}
	changeTMDBAccess(t, p, false)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not stop")
	}
	s := queueState(t, c)
	if s.Libraries[0].Pending != 1 || s.Libraries[0].Failed != 0 || s.RetryAt != 0 || s.Current != "" {
		t.Fatal("revocation counted as identification failure", s)
	}
	c.metadataStep(ctx, allKinds)
	if len(c.queueProviders(ctx, allKinds)) != 0 {
		t.Fatal("revoked provider selected again")
	}
}

func TestServiceAuthorizationMigrationKeepsDisabledProvider(t *testing.T) {
	c, dir := testCatalog(t)
	if err := c.provider("tmdb").(*tmdbProvider).setProviderEnabled(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec(`DROP TABLE extension_authorizations; PRAGMA user_version=16`); err != nil {
		t.Fatal(err)
	}
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	state, err := (authorization.SQLStore{DB: next.db}).Read(context.Background(), tmdbConfigurationOwner)
	if err != nil || !state.Allows("tmdb.api.v1", "get") || providerAvailable(next.provider("tmdb")) {
		t.Fatal("migration changed disabled provider preference", err)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if err != nil || len(backups) == 0 {
		t.Fatal("missing migration backup", err)
	}
}
