package main

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/configuration"
	bundled "mosaune/extensionkit/providers/tmdb"
	"mosaune/extensionkit/transport"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type providerProfileCache struct {
	mu       sync.Mutex
	checksum string
	profile  transport.Profile
}

func (p *tmdbProvider) transportProfile(ctx context.Context) (transport.Profile, error) {
	legacy := func() transport.Profile {
		v := bundled.Transport()
		v.LanguageSetting = ""
		v.RegionSetting = ""
		return v
	}
	if p.db == nil {
		return legacy(), nil
	}
	c := &p.profileCache
	c.mu.Lock()
	defer c.mu.Unlock()
	var sum string
	if e := p.db.QueryRowContext(ctx, "SELECT checksum FROM provider_packages WHERE id='tmdb'").Scan(&sum); e != nil {
		return transport.Profile{}, e
	}
	if sum == c.checksum {
		return c.profile, nil
	}
	pack, e := p.installedPackage(ctx)
	if e != nil {
		return transport.Profile{}, e
	}
	var profile transport.Profile
	if pack.Manifest.Transport != nil {
		profile = *pack.Manifest.Transport
	} else {
		profile = legacy()
	}
	c.checksum = sum
	c.profile = profile
	return profile, nil
}

func (p *tmdbProvider) configuredProfile(ctx context.Context) (transport.Profile, error) {
	profile, e := p.transportProfile(ctx)
	if e != nil {
		return profile, e
	}
	if p.db == nil || profile.LanguageSetting == "" && profile.RegionSetting == "" {
		return profile, nil
	}
	state, e := tmdbConfiguration(p.db).Read(ctx)
	if e != nil {
		return profile, e
	}
	for key, target := range map[string]*string{profile.LanguageSetting: &profile.Language, profile.RegionSetting: &profile.Region} {
		if raw := state.Values[key]; len(raw) > 0 {
			if json.Unmarshal(raw, target) != nil {
				return profile, errors.New("Réglages du fournisseur invalides.")
			}
		}
	}
	return profile, nil
}

// The provider adapter supplies the already-validated plan, the host supplies
// its credential. Source data is authenticated at installation; it cannot grant
// itself a new origin or access the secret through the module protocol.
func providerSourceCall(client *http.Client, rate *providerRateGate, source transport.Source, path string, params url.Values, secret string) (providerHTTPCall, error) {
	endpoint, e := source.URL(path)
	if e != nil {
		return providerHTTPCall{}, e
	}
	params = cloneProviderParams(params)
	if source.QueryKey != "" {
		params.Del(source.QueryKey)
	}
	publicURL := endpoint
	if len(params) > 0 {
		publicURL += "?" + params.Encode()
	}
	bounded := *client
	bounded.Timeout = time.Duration(source.TimeoutMS) * time.Millisecond
	if client.Timeout > 0 && client.Timeout < bounded.Timeout {
		bounded.Timeout = client.Timeout
	}
	call := providerHTTPCall{Client: &bounded, URL: publicURL, CacheKey: publicURL, MaxBytes: source.MaxBytes, TTL: time.Duration(source.TTL(path)) * time.Second, Authorize: providerOrigins([]string{source.BaseURL}, nil)}
	if source.Format == "json" {
		call.Accept = "application/json"
	}
	if source.IntervalMS > 0 {
		call.Wait = func(ctx context.Context) error {
			return rate.wait(ctx, time.Duration(source.IntervalMS)*time.Millisecond)
		}
	}
	if source.Auth != "" {
		if secret == "" {
			return call, errors.New("Configurez la connexion du fournisseur.")
		}
		call.Prepare = func(req *http.Request) {
			if source.Auth == "query" || source.Auth == "key-or-bearer" && len(secret) == 32 && !strings.Contains(secret, ".") {
				q := req.URL.Query()
				q.Set(source.QueryKey, secret)
				req.URL.RawQuery = q.Encode()
			} else {
				req.Header.Set("Authorization", "Bearer "+secret)
			}
		}
	}
	return call, nil
}

// Keep dormant values during rollback/removal. Only fields declared by the
// currently installed package are exposed or writable through its settings UI.
func registerProviderSettings(ctx context.Context, store configuration.SQLStore, schema configuration.Schema) error {
	bound, e := store.For(tmdbConfigurationOwner)
	if e != nil {
		return e
	}
	previous, e := bound.Read(ctx)
	if e == nil {
		seen := map[string]bool{}
		for _, f := range schema.Fields {
			seen[f.ID] = true
		}
		for _, f := range previous.Schema.Fields {
			if !seen[f.ID] {
				schema.Fields = append(schema.Fields, f)
			}
		}
	}
	return store.Register(ctx, tmdbConfigurationOwner, schema)
}
func projectProviderSettings(state configuration.State, schema configuration.Schema) configuration.State {
	state.Schema = schema
	values := map[string]json.RawMessage{}
	secrets := map[string]bool{}
	for _, f := range schema.Fields {
		if v, ok := state.Values[f.ID]; ok {
			values[f.ID] = v
		}
		if v, ok := state.SecretsConfigured[f.ID]; ok {
			secrets[f.ID] = v
		}
	}
	state.Values = values
	state.SecretsConfigured = secrets
	return state
}
