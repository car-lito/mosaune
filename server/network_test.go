package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"
	"time"
)

func testTLSFiles(t *testing.T, host string, expires time.Time) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Mosaune ephemeral test"}, NotBefore: time.Now().Add(-24 * time.Hour), NotAfter: expires, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	if ip := net.ParseIP(host); ip != nil {
		cert.IPAddresses = []net.IP{ip}
	} else {
		cert.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey}), 0600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestNetworkConfiguration(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "::1"} {
		if err := (networkConfig{Listen: ip, Port: 47831}).validate(); err != nil {
			t.Fatal(err)
		}
	}
	base := networkConfig{Listen: "0.0.0.0", Port: 47831, PublicURL: "https://nas.example:47831", Cert: "cert.pem", Key: "key.pem"}
	if err := base.validate(); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"", "http://nas.example", "https://user:password@nas.example", "https://nas.example/path", "https://nas.example?", "https://nas.example?q=1", "https://nas.example#fragment", "https://nas.example:0", "https://nas.example:65536", "https://nas.example:", "https://0.0.0.0", "https://[::]"} {
		n := base
		n.PublicURL = origin
		if err := n.validate(); err == nil {
			t.Errorf("accepted origin %q", origin)
		}
	}
	for _, change := range []func(*networkConfig){func(n *networkConfig) { n.Cert = "" }, func(n *networkConfig) { n.Key = "" }, func(n *networkConfig) { n.Listen = "nas.example" }, func(n *networkConfig) { n.Port = 0 }} {
		n := base
		change(&n)
		if err := n.validate(); err == nil {
			t.Fatal("accepted incomplete configuration")
		}
	}
	cert, key := testTLSFiles(t, "nas.example", time.Now().Add(time.Hour))
	base.Cert, base.Key = cert, key
	if config, err := base.loadTLS(); err != nil || config.MinVersion != tls.VersionTLS12 {
		t.Fatal(config, err)
	}
	base.PublicURL = "https://wrong.example"
	if _, err := base.loadTLS(); err == nil {
		t.Fatal("accepted wrong hostname")
	}
	base.PublicURL = "https://nas.example"
	base.Cert, base.Key = testTLSFiles(t, "nas.example", time.Now().Add(-time.Hour))
	if _, err := base.loadTLS(); err == nil {
		t.Fatal("accepted expired certificate")
	}
}

func TestHTTPSOriginGuard(t *testing.T) {
	n := networkConfig{PublicURL: "https://nas.example:47831"}
	h := n.access(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "setup-token")
	for _, tc := range []struct {
		scheme, host, origin, site, method, token string
		want                                      int
	}{
		{"https", "nas.example:47831", "https://nas.example:47831", "same-origin", "POST", "setup-token", 204},
		{"https", "NAS.EXAMPLE:47831", "", "", "GET", "", 204},
		{"http", "nas.example:47831", "", "", "GET", "", 403},
		{"https", "attacker.example", "", "", "GET", "", 403},
		{"https", "127.0.0.1:47831", "", "", "GET", "", 403},
		{"https", "nas.example:47831", "http://nas.example:47831", "", "GET", "", 403},
		{"https", "nas.example:47831", "https://nas.example:47832", "", "GET", "", 403},
		{"https", "nas.example:47831", "https://nas.example:47831/path", "", "GET", "", 403},
		{"https", "nas.example:47831", "null", "", "GET", "", 403},
		{"https", "nas.example:47831", "", "cross-site", "GET", "", 403},
		{"https", "nas.example:47831", "", "same-site", "GET", "", 403},
		{"https", "nas.example:47831", "", "", "POST", "", 403},
	} {
		r := httptest.NewRequest(tc.method, tc.scheme+"://"+tc.host+"/api/v1/auth/login", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.site)
		r.Header.Set("X-Mosaune-Token", tc.token)
		r.Header.Set("X-Forwarded-Proto", "https")
		r.Header.Set("X-Forwarded-Host", "nas.example:47831")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("%+v: %d", tc, w.Code)
		}
	}
	h = (networkConfig{PublicURL: "https://nas.example:443"}).access(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "")
	r := httptest.NewRequest("GET", "https://nas.example/", nil)
	r.Header.Set("Origin", "https://nas.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("default HTTPS port rejected", w.Code)
	}
}

func TestHTTPSAccountsAndHealth(t *testing.T) {
	c, err := openCatalog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portString, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	cert, key := testTLSFiles(t, "nas.example", time.Now().Add(time.Hour))
	n := networkConfig{Listen: "127.0.0.1", Port: port, PublicURL: "https://nas.example:" + portString, Cert: cert, Key: key, CA: cert}
	config, err := n.loadTLS()
	if err != nil {
		t.Fatal(err)
	}
	modules, _ := enabledModules("music")
	server := &http.Server{Handler: newNetworkHandler(fstest.MapFS{"index.html": {Data: []byte("Mosaune")}}, modules, c, n)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveHTTP(ctx, server, listener, config) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	if err := n.healthcheck(); err != nil {
		t.Fatal(err)
	}
	untrusted := n
	untrusted.CA = ""
	if err := untrusted.healthcheck(); err == nil {
		t.Fatal("untrusted certificate accepted")
	}
	roots := x509.NewCertPool()
	raw, _ := os.ReadFile(cert)
	roots.AppendCertsFromPEM(raw)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", listener.Addr().String())
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, err := client.Get(n.PublicURL + "/api/v1/auth/state")
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Token string
		Setup bool
	}
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !state.Setup {
		t.Fatal("expected initial setup")
	}
	data, _ := json.Marshal(map[string]string{"name": "NetworkAdmin", "password": "test-password-for-https", "code": c.auth.code})
	r, _ := http.NewRequest("POST", n.PublicURL+"/api/v1/auth/setup", bytes.NewReader(data))
	r.Header.Set("Origin", n.PublicURL)
	r.Header.Set("X-Mosaune-Token", state.Token)
	response, err = client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		t.Fatal(response.StatusCode, string(body))
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe session cookie")
	}
	var login struct{ Token string }
	if err := json.NewDecoder(response.Body).Decode(&login); err != nil {
		t.Fatal(err)
	}
	r, _ = http.NewRequest("POST", n.PublicURL+"/api/v1/auth/logout", nil)
	r.AddCookie(cookies[0])
	r.Header.Set("X-Mosaune-Token", login.Token)
	response, err = client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 || len(response.Cookies()) != 1 || !response.Cookies()[0].Secure || response.Cookies()[0].MaxAge != -1 {
		t.Fatal("logout cookie not securely cleared")
	}
}

func TestShutdownWaitsForActiveRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; w.WriteHeader(204) })}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serveHTTP(ctx, server, listener, nil) }()
	defer server.Close()
	clientDone := make(chan struct{})
	clientError := make(chan error, 1)
	go func() {
		defer close(clientDone)
		client := &http.Client{Timeout: 5 * time.Second}
		response, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			response.Body.Close()
		} else {
			clientError <- err
		}
	}()
	select {
	case <-entered:
	case err := <-clientError:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("test request did not reach the server")
	}
	cancel()
	select {
	case <-done:
		t.Fatal("returned before active handler finished")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete")
	}
	<-clientDone
}
