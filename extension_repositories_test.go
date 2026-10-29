package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryExportCanBePublishedUnchanged(t *testing.T) {
	raw, err := bundledRepositoryArchive()
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		files[f.Name] = data
	}
	if len(files) != 7+len(screenPackages()) || len(files["README.md"]) == 0 {
		t.Fatal("incomplete export")
	}
	index, err := parseRepositoryIndex(files[repositoryIndexFile])
	if err != nil {
		t.Fatal(err)
	}
	c, _, _ := accountFixture(t)
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		data := files[strings.TrimPrefix(r.URL.Path, "/repos/author/repo/contents/")]
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data))}, nil
	})
	repo := extensionRepository{URL: "https://github.com/author/repo", Index: index}
	for _, p := range index.Packages {
		download, e := c.repositoryPackage(context.Background(), repo, p.ID, p.Version)
		if e != nil || !bytes.Equal(download, files[p.Path]) {
			t.Fatal("export cannot be consumed", p.ID, e)
		}
	}
}

func TestRepositoryInputAndBounds(t *testing.T) {
	for _, url := range []string{"http://github.com/a/b", "https://github.com.evil.test/a/b", "https://github.com:443/a/b", "https://u:p@github.com/a/b", "https://github.com/a/b?x=1", "https://github.com/a/b/blob/main/x", "https://github.com/a/%2e%2e", "https://127.0.0.1/a/b", "https://github.com/../b"} {
		if _, err := normalizeRepository(url, ""); err == nil {
			t.Error("accepted", url)
		}
	}
	if normalized, err := normalizeRepository("https://github.com/Author/Repo.git/", "feature/test"); err != nil || normalized != "https://github.com/author/repo" {
		t.Fatal(normalized, err)
	}
	for _, path := range []string{"../package.zip", "/package.zip", "https://example.org/file.zip", "x/./y.zip", "x%2Fy.zip", "x\\y.zip"} {
		if repositoryPath(path) {
			t.Error("accepted path", path)
		}
	}
	base := bundledRepository().Index
	check := func(index repositoryIndex, want bool) {
		t.Helper()
		raw, _ := json.Marshal(index)
		_, err := parseRepositoryIndex(raw)
		if (err == nil) != want {
			t.Fatalf("validation err %v expected valid %v", err, want)
		}
	}
	check(base, true)
	bad := base
	bad.Schema = 2
	check(bad, false)
	bad = base
	bad.Packages = append(append([]repositoryEntry{}, base.Packages...), base.Packages[0])
	check(bad, false)
	bad = base
	bad.Packages = append([]repositoryEntry{}, base.Packages...)
	bad.Packages[0].SHA256 = "changed"
	check(bad, false)
	bad.Packages[0] = base.Packages[0]
	bad.Packages[0].Path = "../../outside.zip"
	check(bad, false)
	bad.Packages[0] = base.Packages[0]
	bad.Packages[0].Kind = "native-binary"
	check(bad, false)
	for _, raw := range [][]byte{bytes.Repeat([]byte(" "), repositoryIndexLimit+1), []byte(`{"schema":1,"name":"Test","packages":[],"unknown":true}`), []byte(`{"schema":1,"name":"Test","packages":[]} {}`)} {
		if _, err := parseRepositoryIndex(raw); err == nil {
			t.Error("bad index accepted")
		}
	}
}

func TestRepositoryCatalogDownloadAndPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	index := bundledRepository().Index
	index.Name = "Public fixture"
	indexRaw, _ := json.Marshal(index)
	files := map[string][]byte{repositoryIndexFile: indexRaw, "packages/ambre.mosaune.zip": bundledThemeArchive(), "packages/salon.mosaune.zip": uiExampleArchive(), "packages/tmdb-0.4.0.mosaune.zip": providerExample, "packages/tmdb-0.4.1.mosaune.zip": providerUpdateExample}
	for _, p := range append(screenPackages(), screenExamplePackage()) {
		files[screenArchivePath(p)] = screenArchive(p)
	}
	calls := 0
	status := 200
	networkError := false
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Scheme != "https" || r.URL.Host != "api.github.com" || !strings.HasPrefix(r.URL.Path, "/repos/author/repo/contents/") || r.URL.Query().Get("ref") != "branch/test" {
			t.Fatal("unexpected target", r.URL)
		}
		if r.Header.Get("Accept") != "application/vnd.github.raw+json" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Mosaune-Token") != "" {
			t.Fatal("headers", r.Header)
		}
		if networkError {
			return nil, errors.New("offline")
		}
		body := files[strings.TrimPrefix(r.URL.Path, "/repos/author/repo/contents/")]
		return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"https://localhost/private"}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}, nil
	})
	root := "/api/v1/extension-repositories"
	if w := admin.request("GET", root, nil); w.Code != 200 || calls != 0 {
		t.Fatal("local list reached network", w.Code, calls)
	}
	w := admin.request("POST", root, map[string]string{"url": "https://github.com/Author/Repo", "ref": "branch/test"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var repo extensionRepository
	_ = json.Unmarshal(w.Body.Bytes(), &repo)
	endpoint := root + "/" + repo.ID
	if w = admin.request("POST", root, map[string]string{"url": repo.URL, "ref": repo.Ref}); w.Code != 409 {
		t.Fatal("duplicate", w.Code)
	}
	if calls != 1 {
		t.Fatal("duplicate fetched", calls)
	}
	for _, entry := range index.Packages {
		w = admin.request("POST", endpoint+"/package", map[string]string{"id": entry.ID, "version": entry.Version, "sha256": entry.SHA256})
		if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), files[entry.Path]) {
			t.Fatal("valid download", w.Code, w.Body.String())
		}
	}
	theme := index.Packages[0]
	request := map[string]string{"id": theme.ID, "version": theme.Version, "sha256": theme.SHA256}
	// Install using the same existing validator and mutation routes as the UI.
	pack, err := parseExtensionPackage(files[theme.Path])
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/extensions/install", bytes.NewReader(files[theme.Path]))
	r.Header.Set("X-Mosaune-Token", admin.token)
	r.Header.Set("X-Mosaune-Package-Checksum", pack.Checksum)
	r.AddCookie(admin.cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	// Same catalogue entry cannot silently switch bytes or identity.
	files[theme.Path] = []byte("changed")
	if w = admin.request("POST", endpoint+"/package", request); w.Code != 400 {
		t.Fatal("changed bytes", w.Code)
	}
	files[theme.Path] = bundledThemeArchive()
	files[repositoryIndexFile] = []byte(`{"schema":2}`)
	if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 400 {
		t.Fatal("bad refresh", w.Code)
	}
	old, err := c.repository(context.Background(), repo.ID)
	if err != nil || old.Index.Name != index.Name || len(old.Index.Packages) != len(index.Packages) {
		t.Fatal("previous lost", old, err)
	}
	for _, failure := range []int{403, 404, 429, 500, 302} {
		status = failure
		if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 400 {
			t.Fatal("HTTP failure", failure, w.Code)
		}
	}
	status = 200
	networkError = true
	if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 400 {
		t.Fatal("offline", w.Code)
	}
	networkError = false
	index.Packages[0].Name = "Misleading title"
	files[repositoryIndexFile], _ = json.Marshal(index)
	if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = admin.request("POST", endpoint+"/package", request); w.Code != 400 {
		t.Fatal("misleading title accepted", w.Code)
	}
	index.Packages[0].SHA256 = packageformat.Hash([]byte("new"))
	files[repositoryIndexFile], _ = json.Marshal(index)
	if w = admin.request("POST", endpoint+"/refresh", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = admin.request("POST", endpoint+"/package", request); w.Code != 409 {
		t.Fatal("stale card", w.Code)
	}
	// Access control must precede every network action, not just hide the UI.
	w = admin.request("POST", "/api/v1/accounts", map[string]any{"account": map[string]string{"name": "Reader"}, "password": accountTestPassword})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	before := calls
	for _, op := range []struct{ method, path string }{{"GET", root}, {"POST", root}, {"POST", endpoint + "/refresh"}, {"POST", endpoint + "/package"}, {"DELETE", endpoint}} {
		if w = reader.request(op.method, op.path, request); w.Code != 403 {
			t.Fatal("reader", op, w.Code)
		}
	}
	if calls != before {
		t.Fatal("reader reached network")
	}
	if w = admin.request("DELETE", root+"/"+bundledRepositoryID, nil); w.Code != 400 {
		t.Fatal("removed bundled", w.Code)
	}
	if w = admin.request("DELETE", endpoint, nil); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, err = c.extension(context.Background(), theme.ID); err != nil {
		t.Fatal("removing source removed installed theme", err)
	}
}

func TestRepositoryDownloadLimitsAndProviderTrust(t *testing.T) {
	c, _, _ := accountFixture(t)
	repo := bundledRepository()
	repo.Bundled = false
	repo.URL = "https://github.com/author/repo"
	oversized := bytes.Repeat([]byte("x"), extensionLimit+1)
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: -1, Body: io.NopCloser(bytes.NewReader(oversized))}, nil
	})
	p := repo.Index.Packages[0]
	if _, err := c.repositoryPackage(context.Background(), repo, p.ID, p.Version); err == nil {
		t.Fatal("unbounded chunked body")
	}
	// Correct catalogue hash cannot bypass the provider's signature verification.
	corrupt := append([]byte{}, providerExample...)
	corrupt[len(corrupt)/2] ^= 0xff
	repo.Index.Packages[1].SHA256 = packageformat.Hash(corrupt)
	p = repo.Index.Packages[1]
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(corrupt))}, nil
	})
	if _, err := c.repositoryPackage(context.Background(), repo, p.ID, p.Version); err == nil {
		t.Fatal("corrupted executable accepted")
	}
}

func TestRepositoryMigrationAndReopen(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	removeRecognitionForLegacyFixture(t, c)
	_, err = c.db.Exec(`DROP TABLE extension_repositories;PRAGMA user_version=12`)
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	index := bundledRepository().Index
	raw, _ := json.Marshal(index)
	_, err = c.db.Exec(`INSERT INTO extension_repositories VALUES('fixture','https://github.com/author/repo','',?,1)`, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal("reopen", err)
	}
	defer c.close()
	if _, err = c.repository(context.Background(), "fixture"); err != nil {
		t.Fatal("repository lost", err)
	}
	var v int
	_ = c.db.QueryRow("PRAGMA user_version").Scan(&v)
	backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
	if v != catalogSchemaVersion || len(backups) != 1 {
		t.Fatal(v, backups)
	}
}
