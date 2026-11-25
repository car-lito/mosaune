package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func familyRepositoryFixture(t *testing.T, release string) ([]byte, repositoryEntry) {
	t.Helper()
	raw, e := exampleBundleArchive("films", release)
	if e != nil {
		t.Fatal(e)
	}
	p, e := parseBundleArchive(raw)
	if e != nil {
		t.Fatal(e)
	}
	m := p.Manifest
	return raw, repositoryEntry{ID: m.ID, Name: m.Name, Publisher: m.Publisher, Version: m.Version, Kind: "family", MinCore: m.MinCore, Path: "packages/films-" + release + ".zip", SHA256: p.Checksum}
}

func TestRepositoryFamilyLifecycle(t *testing.T) {
	c, h, a := accountFixture(t)
	a.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	raw, entry := familyRepositoryFixture(t, "1.0.0")
	next, newEntry := familyRepositoryFixture(t, "1.1.0")
	index := repositoryIndex{Schema: 1, Name: "Families", Packages: []repositoryEntry{entry}}
	files := map[string][]byte{entry.Path: raw, newEntry.Path: next}
	files[repositoryIndexFile], _ = json.Marshal(index)
	calls := 0
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Mosaune-Token") != "" {
			t.Fatal("credentials leaked")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(files[strings.TrimPrefix(r.URL.Path, "/repos/author/families/contents/")]))}, nil
	})
	root := "/api/v1/extension-repositories"
	w := a.request("POST", root, map[string]string{"url": "https://github.com/author/families"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var repo extensionRepository
	json.Unmarshal(w.Body.Bytes(), &repo)
	endpoint := root + "/" + repo.ID
	installed := func(want string) {
		t.Helper()
		w := a.request("GET", root, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var result struct {
			Installed []struct{ ID, Version, Kind string }
		}
		json.Unmarshal(w.Body.Bytes(), &result)
		got := ""
		for _, v := range result.Installed {
			if v.ID == entry.ID && v.Kind == "family" {
				got = v.Version
			}
		}
		if got != want {
			t.Fatalf("version %q != %q", got, want)
		}
	}
	upload := func(action string, b []byte, checksum string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/media-bundle-archives/"+action, bytes.NewReader(b))
		req.AddCookie(a.cookie)
		req.Header.Set("X-Mosaune-Token", a.token)
		req.Header.Set("X-Mosaune-Plan-Checksum", checksum)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	inspect := func(e repositoryEntry) ([]byte, bundleArchivePlan) {
		t.Helper()
		w := a.request("POST", endpoint+"/package", map[string]string{"id": e.ID, "version": e.Version, "sha256": e.SHA256})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		b := bytes.Clone(w.Body.Bytes())
		w = upload("preview", b, "")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var p bundleArchivePlan
		json.Unmarshal(w.Body.Bytes(), &p)
		return b, p
	}
	b, p := inspect(entry)
	installed("")
	if w = upload("import", b, p.Checksum); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	installed("1.0.0")
	if bundlePreview(t, c, "films", "prepare").ScreenState != "disabled" {
		t.Fatal("implicit activation")
	}
	if err := bundleApply(c, bundlePreview(t, c, "films", "prepare")); err != nil {
		t.Fatal(err)
	}
	index.Packages = append(index.Packages, newEntry)
	files[repositoryIndexFile], _ = json.Marshal(index)
	if w = a.request("POST", endpoint+"/refresh", nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	installed("1.0.0")
	b, p = inspect(newEntry)
	if err := bundleApply(c, bundlePreview(t, c, "films", "disable")); err != nil {
		t.Fatal(err)
	}
	if w = upload("import", b, p.Checksum); w.Code != 409 {
		t.Fatal("stale plan accepted", w.Code)
	}
	installed("1.0.0")
	b, p = inspect(newEntry)
	if w = upload("import", b, p.Checksum); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	installed("1.1.0")
	if bundlePreview(t, c, "films", "prepare").ScreenState != "disabled" {
		t.Fatal("pause lost")
	}
	// A removed source does not remove the installed family, or trigger traffic.
	if w = a.request("DELETE", endpoint, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	before := calls
	installed("1.1.0")
	if calls != before {
		t.Fatal("background fetch")
	}
	if err := bundleApply(c, bundlePreview(t, c, "films", "remove")); err != nil {
		t.Fatal(err)
	}
	installed("")
}

func TestRepositoryFamilyIntegrityAndPolicy(t *testing.T) {
	c, h, a := accountFixture(t)
	a.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	raw, entry := familyRepositoryFixture(t, "1.0.0")
	repo := extensionRepository{URL: "https://github.com/author/families", Index: repositoryIndex{Schema: 1, Name: "Families", Packages: []repositoryEntry{entry}}}
	data := raw
	size := int64(-1)
	c.repositoryTransport = tmdbFixtureTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: size, Body: io.NopCloser(bytes.NewReader(data))}, nil
	})
	valid := func(want bool) {
		t.Helper()
		_, e := c.repositoryPackage(context.Background(), repo, entry.ID, entry.Version)
		if (e == nil) != want {
			t.Fatalf("valid %v: %v", want, e)
		}
	}
	valid(true)
	repo.Index.Packages[0].Name = "Spoofed"
	valid(false)
	repo.Index.Packages[0] = entry
	data = append(bytes.Clone(raw), 'x')
	valid(false)
	repo.Index.Packages[0].SHA256 = packageformat.Hash(data)
	size = bundleArchiveLimit + 1
	valid(false)
	data = raw
	size = -1
	repo.Index.Packages[0] = entry
	repo.Index.Packages[0].Kind = "theme"
	valid(false)
	repo.Index.Packages[0] = entry
	repo.Index.Packages[0].MinCore = "99.0.0"
	valid(false)
	a.create("Reader", accountTestPassword, nil)
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	if w := reader.request("POST", "/api/v1/extension-repositories/anything/package", entry); w.Code != 403 {
		t.Fatal("reader download", w.Code)
	}
}

func TestFamilyRepositoryExport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repository")
	if e := exportFamilyRepository(dir); e != nil {
		t.Fatal(e)
	}
	if e := exportFamilyRepository(dir); e == nil {
		t.Fatal("overwrites existing directory")
	}
	raw, e := os.ReadFile(filepath.Join(dir, repositoryIndexFile))
	if e != nil {
		t.Fatal(e)
	}
	index, e := parseRepositoryIndex(raw)
	if e != nil || len(index.Packages) != 6 {
		t.Fatal(index, e)
	}
	c, _, _ := accountFixture(t)
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		b, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(r.URL.Path, "/repos/author/families/contents/"))))
		if e != nil {
			t.Fatal(e)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(b))}, nil
	})
	repo := extensionRepository{URL: "https://github.com/author/families", Index: index}
	for _, entry := range index.Packages {
		raw, e := c.repositoryPackage(context.Background(), repo, entry.ID, entry.Version)
		if e != nil {
			t.Fatal(entry.ID, e)
		}
		p, e := parseBundleArchive(raw)
		if e != nil {
			t.Fatal(e)
		}
		for _, program := range p.Modules {
			if bytes.Contains(program.Code, []byte("vcs.revision=")) {
				t.Fatal("Git metadata would change an immutable release")
			}
		}
	}
}
