package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Trusted host service. Extensions submit validated plans; they cannot choose
// credentials, a cache namespace, an HTTP client or their own permissions.
type providerHTTPCall struct {
	Client                          *http.Client
	DB                              *sql.DB
	Provider, URL, CacheKey, Accept string
	MaxBytes                        int64
	TTL                             time.Duration
	Authorize                       func(*url.URL) error
	Prepare                         func(*http.Request)
	Wait                            func(context.Context) error
	Validate                        func([]byte) error
	BeforeCommit                    func(context.Context) (func(), error)
}
type providerHTTPStatusError struct{ Status int }

func (e *providerHTTPStatusError) Error() string {
	return fmt.Sprintf("Source indisponible (HTTP %d).", e.Status)
}

func providerOrigins(origins []string, subdomains []string) func(*url.URL) error {
	return func(u *url.URL) error {
		bad := errors.New("Destination du fournisseur refusée.")
		if u == nil || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			return bad
		}
		for _, base := range origins {
			b, e := url.Parse(base)
			if e == nil && u.Scheme == b.Scheme && u.Host == b.Host {
				return nil
			}
		}
		if u.Scheme == "https" && u.Port() == "" {
			for _, domain := range subdomains {
				if strings.HasSuffix(u.Hostname(), "."+domain) {
					return nil
				}
			}
		}
		return bad
	}
}
func cloneProviderParams(in url.Values) url.Values {
	out := url.Values{}
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

type providerRateGate struct {
	once sync.Once
	gate chan struct{}
	next time.Time
}

func (g *providerRateGate) wait(ctx context.Context, interval time.Duration) error {
	g.once.Do(func() { g.gate = make(chan struct{}, 1) })
	select {
	case g.gate <- struct{}{}:
	case <-ctx.Done():
		return context.Cause(ctx)
	}
	defer func() { <-g.gate }()
	if e := context.Cause(ctx); e != nil {
		return e
	}
	if delay := time.Until(g.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
	if e := context.Cause(ctx); e != nil {
		return e
	}
	g.next = time.Now().Add(interval)
	return nil
}

func fetchProviderHTTP(ctx context.Context, c providerHTTPCall) ([]byte, int, error) {
	if e := context.Cause(ctx); e != nil {
		return nil, 0, e
	}
	if c.Client == nil || c.Authorize == nil || c.MaxBytes < 1 || c.MaxBytes > 8<<20 || c.TTL < 0 || c.TTL > 7*24*time.Hour {
		return nil, 0, errors.New("Configuration du transport invalide.")
	}
	u, e := url.Parse(c.URL)
	if e != nil {
		return nil, 0, errors.New("Requête du fournisseur invalide.")
	}
	if e = c.Authorize(u); e != nil {
		return nil, 0, e
	}
	validate := func(b []byte) error {
		if int64(len(b)) > c.MaxBytes {
			return errors.New("Réponse du fournisseur trop volumineuse.")
		}
		if c.Validate != nil {
			return c.Validate(b)
		}
		return nil
	}
	cache := c.DB != nil && c.CacheKey != "" && c.Provider != "" && c.TTL > 0
	if cache {
		var body []byte
		if c.DB.QueryRowContext(ctx, "SELECT body FROM provider_cache WHERE provider=? AND key=? AND expires>?", c.Provider, c.CacheKey, time.Now().Unix()).Scan(&body) == nil {
			if e = validate(body); e != nil {
				return nil, 0, e
			}
			if e = context.Cause(ctx); e != nil {
				return nil, 0, e
			}
			return body, 200, nil
		}
	}
	if c.Wait != nil {
		if e = c.Wait(ctx); e != nil {
			return nil, 0, e
		}
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if e != nil {
		return nil, 0, errors.New("Requête du fournisseur invalide.")
	}
	req.Header.Set("User-Agent", "Mosaune/"+version)
	if c.Accept != "" {
		req.Header.Set("Accept", c.Accept)
	}
	if c.Prepare != nil {
		c.Prepare(req)
	}
	if e = c.Authorize(req.URL); e != nil {
		return nil, 0, e
	}
	client := *c.Client
	priorRedirect := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) > 5 || priorRedirect == nil {
			return errors.New("Redirection du fournisseur refusée.")
		}
		if err := priorRedirect(next, via); err != nil {
			return err
		}
		return c.Authorize(next.URL)
	}
	response, e := client.Do(req)
	if e != nil {
		if cause := context.Cause(ctx); cause != nil {
			return nil, 0, cause
		}
		return nil, 0, errors.New("Source inaccessible. Vos fichiers restent disponibles.")
	}
	defer response.Body.Close()
	status := response.StatusCode
	if status != http.StatusOK {
		return nil, status, &providerHTTPStatusError{status}
	}
	body, e := readBounded(response.Body, c.MaxBytes)
	if e != nil {
		if cause := context.Cause(ctx); cause != nil {
			return nil, status, cause
		}
		return nil, status, e
	}
	if e = validate(body); e != nil {
		return nil, status, e
	}
	if e = context.Cause(ctx); e != nil {
		return nil, status, e
	}
	if cache {
		if c.BeforeCommit != nil {
			unlock, err := c.BeforeCommit(ctx)
			if err != nil {
				return nil, status, err
			}
			defer unlock()
		}
		if e = context.Cause(ctx); e != nil {
			return nil, status, e
		}
		_, _ = c.DB.ExecContext(ctx, `DELETE FROM provider_cache WHERE expires<? AND NOT (key LIKE 'visual-image:v2:%' AND length(body)>0)`, time.Now().Unix())
		_, _ = c.DB.ExecContext(ctx, `INSERT OR REPLACE INTO provider_cache(provider,key,body,expires) SELECT ?,?,?,? WHERE (SELECT COALESCE(SUM(length(body)),0) FROM provider_cache)+?<16777216`, c.Provider, c.CacheKey, body, time.Now().Add(c.TTL).Unix(), len(body))
	}
	return body, status, nil
}
