package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func recognitionRepositoryEntry(p packageformat.Executable) repositoryEntry {
	m := p.Manifest
	kind := "recognition"
	if m.Contributions[0].Role == "media.detail-list.v1" {
		kind = "detail-list"
	}
	if m.Contributions[0].Role == "media.catalogue.v1" {
		kind = "catalogue"
	}
	if m.Contributions[0].Role == "media.audio-tags.v1" {
		kind = "audio-tags"
	}
	return repositoryEntry{ID: m.Module.ID, Name: m.Name, Publisher: m.Publisher, Description: m.Description, Version: m.Module.Version, Kind: kind, MinCore: m.MinCore, Path: "packages/module-" + m.Module.Version + ".zip", SHA256: p.Checksum}
}

func TestRepositoryRecognitionLifecycle(t *testing.T) {
	testRepositoryExtractionLifecycle(t, recognitionExample)
}
func TestRepositoryDetailListLifecycle(t *testing.T) {
	testRepositoryExtractionLifecycle(t, detailExample)
}
func TestRepositoryCatalogueLifecycle(t *testing.T) {
	testRepositoryExtractionLifecycle(t, catalogueExample)
}
func TestRepositoryAudioTagsLifecycle(t *testing.T) {
	testRepositoryExtractionLifecycle(t, audioTagsExample)
}
func testRepositoryExtractionLifecycle(t *testing.T, example func(*testing.T, string) ([]byte, packageformat.Executable)) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	raw, p := example(t, "1.0.0")
	newer, p2 := example(t, "1.1.0")
	e, e2 := recognitionRepositoryEntry(p), recognitionRepositoryEntry(p2)
	index := repositoryIndex{Schema: 1, Name: "Recognition fixture", Packages: []repositoryEntry{e}}
	files := map[string][]byte{e.Path: raw, e2.Path: newer}
	files[repositoryIndexFile], _ = json.Marshal(index)
	calls := 0
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Mosaune-Token") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("private headers leaked")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(files[strings.TrimPrefix(r.URL.Path, "/repos/author/recognition/contents/")]))}, nil
	})
	root := "/api/v1/extension-repositories"
	w := admin.request("POST", root, map[string]string{"url": "https://github.com/author/recognition"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var repo extensionRepository
	_ = json.Unmarshal(w.Body.Bytes(), &repo)
	endpoint := root + "/" + repo.ID
	upload := func(action string, body []byte, checksum string, revision int64) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/recognition-packages/"+action, bytes.NewReader(body))
		req.Header.Set("X-Mosaune-Token", admin.token)
		req.Header.Set("X-Mosaune-Package-Checksum", checksum)
		req.Header.Set("X-Mosaune-Registry-Revision", strconv.FormatInt(revision, 10))
		req.AddCookie(admin.cookie)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		return response
	}
	inspect := func(entry repositoryEntry) ([]byte, int64) {
		t.Helper()
		response := admin.request("POST", endpoint+"/package", map[string]string{"id": entry.ID, "version": entry.Version, "sha256": entry.SHA256})
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		body := bytes.Clone(response.Body.Bytes())
		response = upload("preview", body, entry.SHA256, 0)
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		var preview struct {
			Revision int64 `json:"revision"`
		}
		_ = json.Unmarshal(response.Body.Bytes(), &preview)
		return body, preview.Revision
	}
	installedVersion := func(want string) {
		t.Helper()
		response := admin.request("GET", root, nil)
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		var state struct {
			Installed []struct{ ID, Version, Kind string }
		}
		_ = json.Unmarshal(response.Body.Bytes(), &state)
		got := ""
		for _, v := range state.Installed {
			if v.ID == e.ID {
				if v.Kind != e.Kind {
					t.Fatal(v)
				}
				got = v.Version
			}
		}
		if got != want {
			t.Fatalf("installed version: got %q want %q", got, want)
		}
	}
	body, rev := inspect(e)
	installedVersion("") // A download and its preview do not install or execute it.
	if w = upload("import", body, e.SHA256, rev); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	installedVersion("1.0.0")
	modules, rev, err := c.recognitionPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range modules {
		if m.ID == e.ID && (m.Enabled || m.Selected) {
			t.Fatal("implicit activation", m)
		}
	}
	if err = c.changeRecognition(context.Background(), e.ID, "enable", rev); err != nil {
		t.Fatal(err)
	}
	_, rev, _ = c.recognitionPackages(context.Background())
	if err = c.changeRecognition(context.Background(), e.ID, "select", rev); err != nil {
		t.Fatal(err)
	}
	index.Packages = append(index.Packages, e2)
	files[repositoryIndexFile], _ = json.Marshal(index)
	if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	installedVersion("1.0.0") // Refresh advertises, never updates.
	body, rev = inspect(e2)
	// A concurrent lifecycle operation invalidates the administrator's preview.
	if err = c.changeRecognition(context.Background(), e.ID, "disable", rev); err != nil {
		t.Fatal(err)
	}
	if w = upload("import", body, e2.SHA256, rev); w.Code != 409 {
		t.Fatal("stale revision accepted", w.Code)
	}
	installedVersion("1.0.0")
	body, rev = inspect(e2)
	if w = upload("import", body, e2.SHA256, rev); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	installedVersion("1.1.0")
	modules, rev, err = c.recognitionPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range modules {
		if m.ID == e.ID && (m.Enabled || !m.Selected || m.PreviousVersion != "1.0.0") {
			t.Fatal("update lost state", m)
		}
	}
	// Removing a source does not uninstall its code or its rollback archive.
	if w = admin.request("DELETE", endpoint, nil); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	before := calls
	if err = c.restoreRecognition(context.Background(), e.ID, rev); err != nil {
		t.Fatal(err)
	}
	installedVersion("1.0.0")
	if calls != before {
		t.Fatal("offline restore contacted source")
	}
	_, rev, _ = c.recognitionPackages(context.Background())
	if err = c.changeRecognition(context.Background(), e.ID, "remove", rev); err != nil {
		t.Fatal(err)
	}
	installedVersion("")
}

func TestRepositoryRecognitionManifestPolicyAndLimits(t *testing.T) {
	c, _, _ := accountFixture(t)
	raw, p := recognitionExample(t, "1.0.0")
	entry := recognitionRepositoryEntry(p)
	repo := extensionRepository{URL: "https://github.com/author/repo", Index: repositoryIndex{Schema: 1, Name: "Fixture", Packages: []repositoryEntry{entry}}}
	served := raw
	c.repositoryTransport = tmdbFixtureTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: -1, Body: io.NopCloser(bytes.NewReader(served))}, nil
	})
	// The executable's bound is larger than the theme bound; legitimate modules fit.
	if len(raw) <= extensionLimit {
		t.Fatal("fixture no longer exercises executable download bound")
	}
	if _, err := c.repositoryPackage(context.Background(), repo, entry.ID, entry.Version); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"name", "publisher", "version", "description", "minCore", "kind"} {
		bad := entry
		switch field {
		case "name":
			bad.Name = "Misleading"
		case "publisher":
			bad.Publisher = "Other"
		case "version":
			bad.Version = "1.2.0"
		case "description":
			bad.Description = "Other"
		case "minCore":
			bad.MinCore = "0.0.1"
		case "kind":
			bad.Kind = "provider"
		}
		repo.Index.Packages = []repositoryEntry{bad}
		if _, err := c.repositoryPackage(context.Background(), repo, bad.ID, bad.Version); err == nil {
			t.Fatal("mismatched", field)
		}
	}
	repo.Index.Packages = []repositoryEntry{entry}
	served = bytes.Repeat([]byte("x"), packageformat.MaxArchive+1)
	if _, err := c.repositoryPackage(context.Background(), repo, entry.ID, entry.Version); err == nil {
		t.Fatal("unbounded download")
	}
	// Even a matching catalogue hash cannot grant services or reserve factory IDs.
	for _, reserved := range []bool{false, true} {
		m := p.Manifest
		if reserved {
			m.Module.ID = "mosaune.external"
		} else {
			m.Module.Services = []protocol.ServiceRequirement{{Name: "network.http.v1", Operations: []string{"get"}}}
		}
		var err error
		served, err = packageformat.CreateExecutable(m, p.Code)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := packageformat.ParseExecutable(served, version)
		if err != nil {
			t.Fatal(err)
		}
		bad := recognitionRepositoryEntry(parsed)
		repo.Index.Packages = []repositoryEntry{bad}
		if _, err = c.repositoryPackage(context.Background(), repo, bad.ID, bad.Version); err == nil {
			t.Fatal("host policy bypass")
		}
	}
}
