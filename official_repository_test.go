package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestOfficialRepositoryOfflineRefreshAndPersistence(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	official := officialRepository()
	root := "/api/v1/extension-repositories"
	endpoint := root + "/" + official.ID
	calls := 0
	offline := false
	index := bundledRepository().Index
	index.Name = "Official fixture"
	raw, _ := json.Marshal(index)
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.github.com/repos/car-lito/mosaune-extensions/contents/mosaune-repository.json" {
			t.Fatal("wrong official source", r.URL)
		}
		if offline {
			return nil, errors.New("offline")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(raw))}, nil
	})
	read := func() extensionRepository {
		t.Helper()
		w := admin.request("GET", root, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var data struct {
			Repositories []extensionRepository `json:"repositories"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		count := 0
		var found extensionRepository
		for _, r := range data.Repositories {
			if r.Official {
				count++
				found = r
			}
		}
		if count != 1 || len(data.Repositories) != 2 || found.ID != official.ID {
			t.Fatal("missing or duplicate official source", data)
		}
		return found
	}
	if r := read(); r.UpdatedAt != 0 || len(r.Index.Packages) != 0 || calls != 0 {
		t.Fatal("initial list must stay local", r, calls)
	}
	if w := admin.request("POST", root, map[string]string{"url": official.URL}); w.Code != 409 || calls != 0 {
		t.Fatal("duplicate official add", w.Code, calls)
	}
	if w := admin.request("POST", endpoint+"/refresh", nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	loaded := read()
	if loaded.UpdatedAt == 0 || len(loaded.Index.Packages) != len(index.Packages) || calls != 1 {
		t.Fatal("first refresh not saved", loaded, calls)
	}
	offline = true
	if w := admin.request("POST", endpoint+"/refresh", nil); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if r := read(); r.UpdatedAt != loaded.UpdatedAt || r.Index.Name != index.Name || len(r.Index.Packages) != len(index.Packages) {
		t.Fatal("cache lost offline", r)
	}
	if w := admin.request("DELETE", endpoint, nil); w.Code != 400 {
		t.Fatal("official removed", w.Code)
	}
	if calls != 2 {
		t.Fatal("unexpected background calls", calls)
	}
}

func TestOfficialRepositoryAdoptsExistingSource(t *testing.T) {
	dir := t.TempDir()
	c, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	official := officialRepository()
	index := bundledRepository().Index
	index.Name = "Existing 0.0.20 cache"
	raw, _ := json.Marshal(index)
	_, err = c.db.Exec(`INSERT INTO extension_repositories VALUES(?,?,?,?,?)`, official.ID, official.URL, "", string(raw), 123)
	if err != nil {
		t.Fatal(err)
	}
	c.close()
	c, err = openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	r, err := c.repository(context.Background(), official.ID)
	if err != nil || !r.Official || r.UpdatedAt != 123 || r.Index.Name != index.Name {
		t.Fatal("existing source lost", r, err)
	}
}
