package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestModuleSelection(t *testing.T) {
	modules, err := enabledModules(" music, music ")
	if err != nil || len(modules) != 1 || modules[0].ID != "music" {
		t.Fatalf("selection: %v %v", modules, err)
	}
	if _, err := enabledModules("unknown"); err == nil {
		t.Fatal("unknown module accepted")
	}
	modules, err = enabledModules("")
	if err != nil || modules == nil || len(modules) != 0 {
		t.Fatal("empty module list must serialize as []")
	}
}

func TestRoutes(t *testing.T) {
	modules, _ := enabledModules("music")
	c, err := openCatalog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	handler := newHandler(fstest.MapFS{"index.html": {Data: []byte("prototype")}}, modules, c)
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/", 200}, {"GET", "/healthz", 200}, {"GET", "/api/v1/status", 401},
		{"GET", "/api/missing", 401}, {"POST", "/", 405}, {"GET", "/main.go", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, "http://127.0.0.1"+tc.path, nil))
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
			if w.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("missing policy")
			}
		})
	}
	if e := c.createFirstAccount(httptest.NewRequest("GET", "/", nil).Context(), account{ID: "test-user", Name: "TestAdmin", Admin: true}, "unused-test-hash"); e != nil {
		t.Fatal(e)
	}
	authenticated, _ := handlerFor(t, c, "music")
	w := httptest.NewRecorder()
	authenticated.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/status", nil))
	var status struct {
		Modules          []module `json:"modules"`
		LibraryConnected bool     `json:"libraryConnected"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Modules) != 1 || status.Modules[0].ID != "music" || !status.LibraryConnected {
		t.Fatal("incorrect module capability response")
	}
}
