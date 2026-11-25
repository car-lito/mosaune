package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// A single explicit origin is the authority. Forwarded headers never enable TLS
// or change the accepted Host. Reverse-proxy termination is not supported yet.
type networkConfig struct {
	Listen, PublicURL, Cert, Key, CA string
	Port                             int
}

func (n networkConfig) validate() error {
	if n.Port < 1 || n.Port > 65535 {
		return errors.New("port invalide")
	}
	ip := net.ParseIP(n.Listen)
	if ip == nil {
		return errors.New("listen doit être une adresse IP, par exemple 127.0.0.1 ou 0.0.0.0")
	}
	if n.PublicURL == "" {
		if !ip.IsLoopback() || n.Cert != "" || n.Key != "" {
			return errors.New("une adresse public-url HTTPS et un certificat sont requis pour l’accès réseau")
		}
		return nil
	}
	u, err := url.Parse(n.PublicURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(n.PublicURL, "\\ \t\r\n") {
		return errors.New("public-url doit être une origine HTTPS complète, sans chemin, identifiant, requête ni fragment")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && ip.IsUnspecified() {
		return errors.New("public-url doit désigner l’adresse utilisée par les clients, pas une adresse d’écoute générique")
	}
	if p := u.Port(); p != "" {
		value, err := strconv.Atoi(p)
		if err != nil || value < 1 || value > 65535 {
			return errors.New("port public-url invalide")
		}
	}
	if strings.HasSuffix(u.Host, ":") || n.Cert == "" || n.Key == "" {
		return errors.New("tls-cert et tls-key sont requis avec public-url")
	}
	return nil
}

func (n networkConfig) address() string { return net.JoinHostPort(n.Listen, strconv.Itoa(n.Port)) }
func (n networkConfig) origin() string {
	if n.PublicURL != "" {
		return n.PublicURL
	}
	return "http://" + n.address()
}

func (n networkConfig) loadTLS() (*tls.Config, error) {
	if n.PublicURL == "" {
		return nil, nil
	}
	pair, err := tls.LoadX509KeyPair(n.Cert, n.Key)
	if err != nil {
		return nil, fmt.Errorf("certificat HTTPS : %w", err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse(n.PublicURL)
	if err = leaf.VerifyHostname(u.Hostname()); err != nil {
		return nil, fmt.Errorf("le certificat ne couvre pas l’adresse public-url : %w", err)
	}
	now := time.Now()
	if now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) {
		return nil, errors.New("certificat HTTPS expiré ou pas encore valide")
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}}, nil
}

func localAccess(next http.Handler, token string) http.Handler {
	return (networkConfig{}).access(next, token)
}

func (n networkConfig) access(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := "http://" + r.Host
		if n.PublicURL == "" {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if host != "127.0.0.1" && host != "localhost" && host != "::1" {
				problem(w, 403, "Adresse locale requise.")
				return
			}
			if r.TLS != nil {
				problem(w, 403, "Adresse non configurée.")
				return
			}
		} else {
			u, _ := url.Parse(n.PublicURL) // validated before constructing the server
			if u == nil || r.TLS == nil || !strings.EqualFold(strings.TrimSuffix(r.Host, ":443"), strings.TrimSuffix(u.Host, ":443")) {
				problem(w, 403, "Adresse HTTPS non autorisée.")
				return
			}
			origin = strings.TrimSuffix(n.PublicURL, ":443")
		}
		if supplied := r.Header.Get("Origin"); supplied != "" && !strings.EqualFold(supplied, origin) {
			problem(w, 403, "Origine refusée.")
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			problem(w, 403, "Origine refusée.")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && (r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/setup" || r.URL.Path == "/api/v1/auth/recover" || r.URL.Path == "/api/v1/auth/activate") {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Mosaune-Token")), []byte(token)) != 1 {
				problem(w, 403, "Rechargez la page pour reconnecter Mosaune.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Shutdown must finish before run closes SQLite. SIGTERM also cancels requests
// and background jobs; a slow stream cannot keep the container alive forever.
func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener, tlsConfig *tls.Config) error {
	finished, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if server.Shutdown(shutdown) != nil {
				_ = server.Close()
			}
		case <-finished:
		}
	}()
	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
	}
	err := server.Serve(listener)
	close(finished)
	<-stopped
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (n networkConfig) healthcheck() error {
	u, _ := url.Parse(n.origin())
	u.Path = "/healthz"
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if n.CA != "" {
		pem, err := os.ReadFile(n.CA)
		if err != nil {
			return err
		}
		if !roots.AppendCertsFromPEM(pem) {
			return errors.New("autorité TLS illisible")
		}
	}
	address := n.address()
	if ip := net.ParseIP(n.Listen); ip.IsUnspecified() {
		local := "127.0.0.1"
		if ip.To4() == nil {
			local = "::1"
		}
		address = net.JoinHostPort(local, strconv.Itoa(n.Port))
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", address)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection de santé refusée") }}
	response, err := client.Get(u.String())
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var health struct{ Status, Version string }
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 1024)).Decode(&health) != nil || health.Status != "ok" || health.Version != version {
		return errors.New("serveur indisponible ou version inattendue")
	}
	return nil
}
