package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProviderHTTPRateCancellation(t *testing.T) {
	var gate providerRateGate
	if err := gate.wait(context.Background(), time.Hour); err != nil {
		t.Fatal(err)
	}
	// Another caller owns the gate while waiting for its turn. Cancellation must
	// interrupt queueing too, not merely the timer after acquiring the gate.
	gate.gate <- struct{}{}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("extension disabled")
	done := make(chan error, 1)
	go func() { done <- gate.wait(ctx, time.Hour) }()
	cancel(cause)
	select {
	case err := <-done:
		if !errors.Is(err, cause) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued request ignored cancellation")
	}
	<-gate.gate
	ctx, cancel = context.WithCancelCause(context.Background())
	go func() { done <- gate.wait(ctx, time.Hour) }()
	cancel(cause)
	select {
	case err := <-done:
		if !errors.Is(err, cause) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("rate timer ignored cancellation")
	}
	var other providerRateGate
	if err := other.wait(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestProviderHTTPCacheBoundaries(t *testing.T) {
	c, _ := testCatalog(t)
	calls := 0
	response := `{"title":"OK"}`
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { calls++; return visualResponse(200, response), nil })}
	call := providerHTTPCall{Client: client, DB: c.db, Provider: "transport-test", URL: "https://source.example/item", CacheKey: "item", TTL: time.Hour, MaxBytes: 32, Authorize: providerOrigins([]string{"https://source.example"}, nil), Validate: func(b []byte) error { var out map[string]string; return json.Unmarshal(b, &out) }}
	for i := 0; i < 2; i++ {
		if _, _, err := fetchProviderHTTP(context.Background(), call); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("cache missed", calls)
	}
	// A warm cache does not bypass source authorization.
	call.Authorize = providerOrigins([]string{"https://other.example"}, nil)
	if _, _, err := fetchProviderHTTP(context.Background(), call); err == nil || calls != 1 {
		t.Fatal("unauthorized cached result", err, calls)
	}
	call.Authorize = providerOrigins([]string{"https://source.example"}, nil)
	call.Provider = "other-provider"
	if _, _, err := fetchProviderHTTP(context.Background(), call); err != nil || calls != 2 {
		t.Fatal("cache namespace crossed", err, calls)
	}
	call.Provider = "transport-test"
	for _, body := range []string{`{"bad":`, strings.Repeat("x", 33)} {
		response = body
		call.CacheKey = "bad"
		if _, _, err := fetchProviderHTTP(context.Background(), call); err == nil {
			t.Fatal("invalid response accepted")
		}
		var n int
		if err := c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE provider=? AND key='bad'", call.Provider).Scan(&n); err != nil || n != 0 {
			t.Fatal("invalid response cached", err, n)
		}
	}
}

func TestProviderHTTPRevocationBeforeCommit(t *testing.T) {
	c, _ := testCatalog(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("permission revoked")
	call := providerHTTPCall{Client: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { return visualResponse(200, `{}`), nil })}, DB: c.db, Provider: "revoke-test", URL: "https://source.example/item", CacheKey: "item", TTL: time.Hour, MaxBytes: 32, Authorize: providerOrigins([]string{"https://source.example"}, nil), BeforeCommit: func(context.Context) (func(), error) { cancel(cause); return func() {}, cause }}
	if _, _, err := fetchProviderHTTP(ctx, call); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	var n int
	if err := c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE provider='revoke-test'").Scan(&n); err != nil || n != 0 {
		t.Fatal("revoked result cached", err, n)
	}
}

func TestProviderHTTPRedirectPolicy(t *testing.T) {
	calls := 0
	const origin = "https://source.example"
	const destination = "https://destination.example"
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return nil }, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "source.example" {
			response := visualResponse(http.StatusFound, "")
			response.Header.Set("Location", destination)
			return response, nil
		}
		calls++
		return visualResponse(200, "OK"), nil
	})}
	call := providerHTTPCall{Client: client, URL: origin, MaxBytes: 32, Authorize: providerOrigins([]string{origin}, nil), Prepare: func(r *http.Request) { r.Header.Set("Authorization", "Bearer private-test-token") }}
	if _, _, err := fetchProviderHTTP(context.Background(), call); err == nil || strings.Contains(err.Error(), "private-test-token") || calls != 0 {
		t.Fatal("redirect escaped policy", err, calls)
	}
	call.Prepare = nil
	call.Authorize = providerOrigins([]string{origin, destination}, nil)
	if body, _, err := fetchProviderHTTP(context.Background(), call); err != nil || string(body) != "OK" || calls != 1 {
		t.Fatal("approved redirect failed", err, calls)
	}
	// No redirect policy is not implicit permission to follow one.
	client.CheckRedirect = nil
	if _, _, err := fetchProviderHTTP(context.Background(), call); err == nil || calls != 1 {
		t.Fatal("missing redirect policy accepted", err, calls)
	}
}

func TestProviderHTTPTMDBCredentialsStayPrivate(t *testing.T) {
	c, _ := testCatalog(t)
	p := newTMDBProvider(c.db)
	t.Cleanup(p.closeExtension)
	secret := strings.Repeat("a", 32)
	params := url.Values{"query": {"example"}, "api_key": {"caller-secret"}}
	original := cloneProviderParams(params)
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("api_key") != secret || r.Header.Get("Authorization") != "" {
			t.Error("incorrect credential delivery")
		}
		return visualResponse(200, `{"results":[]}`), nil
	})
	for i := 0; i < 2; i++ {
		var out map[string]any
		if err := p.get(context.Background(), secret, "/search/movie", params, &out); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(params, original) || calls != 1 {
		t.Fatal("caller parameters mutated or cache missed", calls)
	}
	var key string
	if err := c.db.QueryRow("SELECT key FROM provider_cache WHERE provider='tmdb'").Scan(&key); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(key, "api_key") || strings.Contains(key, secret) || strings.Contains(key, "caller-secret") {
		t.Fatal("credential persisted in key")
	}
	// Disable must still reject the previously cached request.
	p.state.mu.Lock()
	p.state.disabled = true
	p.state.mu.Unlock()
	var out map[string]any
	if err := p.get(context.Background(), secret, "/search/movie", params, &out); !errors.Is(err, errProviderDisabled) || calls != 1 {
		t.Fatal("disabled provider used cache", err, calls)
	}
}

func TestProviderHTTPOrigins(t *testing.T) {
	authorize := providerOrigins([]string{"https://coverartarchive.org"}, []string{"archive.org"})
	for _, raw := range []string{"https://coverartarchive.org/item", "https://ia123.archive.org/item"} {
		u, _ := url.Parse(raw)
		if err := authorize(u); err != nil {
			t.Fatal(raw, err)
		}
	}
	for _, raw := range []string{"http://coverartarchive.org/item", "https://user:secret@coverartarchive.org/item", "https://coverartarchive.org:8443/item", "https://notarchive.org/item", "https://archive.org.evil.example/item", "https://ia123.archive.org/item#fragment"} {
		u, _ := url.Parse(raw)
		if err := authorize(u); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}
