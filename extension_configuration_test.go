package main

import (
	"context"
	"strings"
	"testing"
)

func TestExtensionConfigurationSharesHostSecretAndChecksAllWrites(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	path := "/api/v1/extensions/bundled/mosaune.tmdb-search/configuration"
	key := "fixture-secret-never-returned-123456"
	if w := admin.request("PUT", "/api/v1/metadata/providers/tmdb", map[string]string{"secret": key}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	read := admin.request("GET", path, nil)
	if read.Code != 200 || strings.Contains(read.Body.String(), key) || !strings.Contains(read.Body.String(), `"apiToken":true`) || !strings.Contains(read.Body.String(), `"services"`) {
		t.Fatal(read.Code, read.Body.String())
	}
	// Missing or empty patch keeps the native credential; no duplicate store.
	for _, body := range []any{map[string]any{"values": map[string]any{}}, map[string]any{}} {
		if w := admin.request("PUT", path, body); w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if c.credential(context.Background(), "tmdb") != key {
		t.Fatal("empty patch lost credential")
	}
	for _, body := range []any{
		map[string]any{"values": map[string]any{"apiToken": "short"}},
		map[string]any{"values": map[string]any{"apiToken": true}},
		map[string]any{"values": map[string]any{"apiToken": nil}},
		map[string]any{"values": map[string]any{"evil": "injected"}, "clearSecrets": []string{"apiToken"}},
		map[string]any{"values": map[string]any{"apiToken": key}, "clearSecrets": []string{"apiToken"}},
	} {
		w := admin.request("PUT", path, body)
		if w.Code != 400 || strings.Contains(w.Body.String(), key) {
			t.Fatal(w.Code, w.Body.String())
		}
		if c.credential(context.Background(), "tmdb") != key {
			t.Fatal("invalid patch partly applied")
		}
	}
	newer := "fixture-replacement-key-123456789"
	if w := admin.request("PUT", path, map[string]any{"values": map[string]string{"apiToken": newer}}); w.Code != 204 || strings.Contains(w.Body.String(), newer) {
		t.Fatal(w.Code)
	}
	if c.credential(context.Background(), "tmdb") != newer {
		t.Fatal("native provider cannot see new key")
	}
	if w := admin.request("PUT", "/api/v1/metadata/tmdb-runtime", map[string]bool{"enabled": false}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if c.credential(context.Background(), "tmdb") != newer {
		t.Fatal("disable lost key")
	}
	if w := admin.request("PUT", path, map[string]any{"clearSecrets": []string{"apiToken"}}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if c.credential(context.Background(), "tmdb") != "" {
		t.Fatal("clear not shared")
	}
	if w := admin.request("GET", path, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"apiToken":false`) {
		t.Fatal(w.Code, w.Body.String())
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "PUT"} {
		if w := reader.request(method, path, map[string]any{}); w.Code != 403 {
			t.Fatal("non-admin", w.Code)
		}
	}
	if w := admin.request("GET", "/api/v1/extensions/bundled/arbitrary.module/configuration", nil); w.Code != 404 {
		t.Fatal("unknown module", w.Code)
	}
}
