package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProviderIdentifiersInstalledRulesAndLifecycle(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	p := c.provider("tmdb").(*tmdbProvider)
	pack, e := p.installedPackage(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(pack.Manifest.Identifiers) != 1 {
		t.Fatal("fresh install missing routes")
	}
	removeTMDBPackageFixture(t, p)
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	p.extension.key = pub
	pack.Manifest.Identifiers[0].Path = "/replacement/{id}"
	pack.Manifest.Identifiers[0].Field = "replacement_id"
	raw, e := packageformat.Create(pack.Manifest, pack.Code, key)
	if e != nil {
		t.Fatal(e)
	}
	pack, e = packageformat.Parse(raw, pub, version)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.installPackage(ctx, pack, "", false); e != nil {
		t.Fatal(e)
	}
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/3/replacement/42" || r.Header.Get("Authorization") != "Bearer private-fixture-token" {
			t.Error("route/credential not supplied by installed profile", r.URL.Path)
		}
		return visualResponse(200, `{"tvdb_id":999,"replacement_id":84}`), nil
	})
	for i := 0; i < 2; i++ {
		v, e := acquisitionIdentifier(ctx, p, "private-fixture-token", "series", "42")
		if e != nil || v != 84 {
			t.Fatal(v, e)
		}
	}
	if calls != 1 {
		t.Fatal("cache not reused", calls)
	}
	if e = p.setExtension(ctx, false); e != nil {
		t.Fatal(e)
	}
	if _, e = acquisitionIdentifier(ctx, p, "private-fixture-token", "series", "42"); !errors.Is(e, errProviderModuleUnavailable) {
		t.Fatal("disabled cached lookup", e)
	}
	if v, e := acquisitionIdentifier(ctx, p, "", "films", "42"); e != nil || v != 42 {
		t.Fatal("stored film identity lost", v, e)
	}
	if e = p.removePackage(ctx, pack.Checksum); e != nil {
		t.Fatal(e)
	}
	if _, e = acquisitionIdentifier(ctx, p, "private-fixture-token", "series", "42"); !errors.Is(e, errProviderModuleUnavailable) {
		t.Fatal("removed fallback", e)
	}
	if calls != 1 {
		t.Fatal("disabled/removed network", calls)
	}
}

func TestProviderIdentifiersLegacyAndMalformed(t *testing.T) {
	c, _ := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	removeTMDBPackageFixture(t, p)
	old, e := packageformat.Parse(providerTransportExample, trustedProviderKey(), version)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.installPackage(ctx, old, "", false); e != nil {
		t.Fatal(e)
	}
	if e = p.setExtension(ctx, true); e != nil {
		t.Fatal(e)
	}
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return visualResponse(200, `{"tvdb_id":1.5}`), nil
	})
	if _, e = acquisitionIdentifier(ctx, p, "fixture", "series", "42"); e == nil || !strings.Contains(e.Error(), "mise à jour") {
		t.Fatal(e)
	}
	if calls != 0 {
		t.Fatal("legacy fallback")
	}
	pack, e := packageformat.Parse(providerIdentifiersExample, trustedProviderKey(), version)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.installPackage(ctx, pack, old.Checksum, false); e != nil {
		t.Fatal(e)
	}
	if _, e = acquisitionIdentifier(ctx, p, "fixture", "series", "42"); e == nil {
		t.Fatal("fraction accepted")
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	if _, e = acquisitionIdentifier(ctx, p, "fixture", "series", "../42"); e == nil {
		t.Fatal("unsafe ID")
	}
	if _, e = acquisitionIdentifier(ctx, p, "fixture", "films", "999999999999999999999"); e == nil {
		t.Fatal("oversized connector ID")
	}
	if calls != 1 {
		t.Fatal("invalid input reached network")
	}
	if e = p.installPackage(ctx, old, pack.Checksum, true); e != nil {
		t.Fatal(e)
	}
	if _, e = acquisitionIdentifier(ctx, p, "fixture", "series", "42"); e == nil {
		t.Fatal("rollback kept removed mapping")
	}
	if calls != 1 {
		t.Fatal("rollback fallback")
	}
}

type alternateIdentifierProvider struct{}

func (alternateIdentifierProvider) ResolveIdentifier(_ context.Context, _ string, q metadata.IdentifierQuery) (string, error) {
	if q.ID != "opaque-work" || q.Namespace != "tvdb" {
		return "", errors.New("wrong query")
	}
	return "123", nil
}
func TestAcquisitionUsesGenericIdentifierProvider(t *testing.T) {
	v, e := acquisitionIdentifier(context.Background(), alternateIdentifierProvider{}, "", "series", "opaque-work")
	if e != nil || v != 123 {
		t.Fatal(v, e)
	}
}

func TestProviderIdentifiersCancelAndRestart(t *testing.T) {
	c, dir := testCatalog(t)
	p := c.provider("tmdb").(*tmdbProvider)
	ctx := context.Background()
	entered := make(chan struct{})
	finished := make(chan error, 1)
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	go func() { _, e := acquisitionIdentifier(ctx, p, "fixture", "series", "42"); finished <- e }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}
	if e := p.setProviderEnabled(ctx, false); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-finished:
		if !errors.Is(e, errProviderDisabled) {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request not cancelled")
	}
	c.close()
	c, e := openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("tmdb").(*tmdbProvider)
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		t.Error("paused network after restart")
		return nil, errors.New("unexpected network")
	})
	if _, e := acquisitionIdentifier(ctx, p, "fixture", "series", "42"); !errors.Is(e, errProviderDisabled) {
		t.Fatal(e)
	}
	pack, e := p.installedPackage(ctx)
	if e != nil || len(pack.Manifest.Identifiers) != 1 {
		t.Fatal("routes lost", e)
	}
	for _, change := range []func(*packageformat.Manifest){
		func(m *packageformat.Manifest) { m.MinCore = "0.0.70" },
		func(m *packageformat.Manifest) { m.Identifiers[0].Path = "https://evil.invalid/{id}" },
		func(m *packageformat.Manifest) { m.Identifiers[0].Source = "trailers" },
	} {
		fresh, e := packageformat.Parse(providerIdentifiersExample, trustedProviderKey(), version)
		if e != nil {
			t.Fatal(e)
		}
		change(&fresh.Manifest)
		if fresh.Manifest.Validate(version) == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
}
