package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"strings"
	"testing"
)

// Independent signed fixture format, not the TMDB parser. No downloaded code
// executes: the fixture verifies repository dispatch into the selected adapter.
type repositoryProviderFixture struct {
	*packageAdapterFixture
	claims []string
	key    ed25519.PublicKey
}

func (p *repositoryProviderFixture) repositoryPackageIDs() []string { return p.claims }

type repositorySignedFixture struct {
	Manifest  packageformat.Manifest
	Signature []byte
}

func (p *repositoryProviderFixture) parseProviderPackage(raw []byte) (packageformat.Package, error) {
	p.reads++
	var envelope repositorySignedFixture
	if json.Unmarshal(raw, &envelope) != nil {
		return packageformat.Package{}, errors.New("Invalid fixture archive")
	}
	signed, _ := json.Marshal(envelope.Manifest)
	if !ed25519.Verify(p.key, signed, envelope.Signature) {
		return packageformat.Package{}, errors.New("Untrusted fixture signature")
	}
	if len(p.claims) != 1 || envelope.Manifest.Descriptor.ID != p.claims[0] {
		return packageformat.Package{}, errors.New("Wrong fixture package")
	}
	return packageformat.Package{Manifest: envelope.Manifest, Checksum: packageformat.Hash(raw), Archive: raw}, nil
}

func TestProviderRepositoryDispatchAndTrust(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	p := &repositoryProviderFixture{packageAdapterFixture: &packageAdapterFixture{id: "other-source"}, claims: []string{"org.fixture.source"}, key: public}
	c.providers = append(c.providers, p)
	m := packageformat.Manifest{Name: "Other source", Publisher: "Fixture publisher", Description: "Independent provider", MinCore: "0.0.79"}
	m.Descriptor.ID = p.claims[0]
	m.Descriptor.Version = "1.0.0"
	sign := func(manifest packageformat.Manifest) []byte {
		signed, _ := json.Marshal(manifest)
		raw, _ := json.Marshal(repositorySignedFixture{Manifest: manifest, Signature: ed25519.Sign(private, signed)})
		return raw
	}
	raw := sign(m)
	entry := repositoryEntry{ID: m.Descriptor.ID, Version: m.Descriptor.Version, Name: m.Name, Publisher: m.Publisher, Description: m.Description, MinCore: m.MinCore, Kind: "provider", Path: "provider.zip", SHA256: packageformat.Hash(raw)}
	index := repositoryIndex{Schema: 1, Name: "Fixture repository", Packages: []repositoryEntry{entry}}
	indexRaw, _ := json.Marshal(index)
	downloads := 0
	c.repositoryTransport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		data := indexRaw
		if strings.HasSuffix(r.URL.Path, "/provider.zip") {
			downloads++
			data = raw
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data))}, nil
	})
	added := admin.request("POST", "/api/v1/extension-repositories", map[string]string{"url": "https://github.com/fixture/provider"})
	var repo extensionRepository
	if added.Code != 201 || json.Unmarshal(added.Body.Bytes(), &repo) != nil {
		t.Fatal(added.Code, added.Body.String())
	}
	route := "/api/v1/extension-repositories/" + repo.ID + "/package"
	input := map[string]string{"id": entry.ID, "version": entry.Version, "sha256": entry.SHA256}
	fetched := admin.request("POST", route, input)
	if fetched.Code != 200 || fetched.Header().Get("X-Mosaune-Package-Provider") != p.id || !bytes.Equal(fetched.Body.Bytes(), raw) || downloads != 1 {
		t.Fatal(fetched.Code, fetched.Body.String())
	}
	if p.mutations != 0 {
		t.Fatal("download installed an archive")
	}
	preview := providerUpload(admin, p.id+"/preview", fetched.Body.Bytes(), "", "")
	if preview.Code != 200 {
		t.Fatal(preview.Code, preview.Body.String())
	}
	if w := providerUpload(admin, p.id+"/install", raw, entry.SHA256, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	listing := admin.request("GET", "/api/v1/extension-repositories", nil)
	var state struct {
		Installed        []struct{ ID, Version, Kind string }
		ProviderBindings []providerRepositoryBinding
	}
	if listing.Code != 200 || json.Unmarshal(listing.Body.Bytes(), &state) != nil {
		t.Fatal(listing.Code)
	}
	found, bound := false, false
	for _, v := range state.Installed {
		if v.ID == entry.ID && v.Version == entry.Version && v.Kind == "provider" {
			found = true
		}
	}
	for _, v := range state.ProviderBindings {
		if v.PackageID == entry.ID && v.ProviderID == p.id {
			bound = true
		}
	}
	if !found || !bound {
		t.Fatal("alternate provider missing from catalogue", state)
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	if w := reader.request("POST", route, input); w.Code != 403 || downloads != 1 {
		t.Fatal("unprivileged download", w.Code)
	}
	// An absent or ambiguous handler is rejected before network access or parsing.
	base := append([]MetadataProvider{}, c.providers...)
	c.providers = c.providers[:len(c.providers)-1]
	if w := admin.request("POST", route, input); w.Code != 409 || downloads != 1 {
		t.Fatal("missing adapter downloaded", w.Code)
	}
	c.providers = append(base, &repositoryProviderFixture{packageAdapterFixture: &packageAdapterFixture{id: "conflict"}, claims: p.claims, key: public})
	if w := admin.request("POST", route, input); w.Code != 409 || downloads != 1 {
		t.Fatal("ambiguous adapter downloaded", w.Code)
	}
	c.providers = base
	// Correct hashes do not authorize a wrong signer or lying catalogue metadata.
	altered := m
	altered.Name = "Other name"
	raw = sign(altered)
	bad := repo
	bad.Index.Packages = append([]repositoryEntry{}, repo.Index.Packages...)
	bad.Index.Packages[0].SHA256 = packageformat.Hash(raw)
	if _, err := c.repositoryPackage(context.Background(), bad, entry.ID, entry.Version); err == nil {
		t.Fatal("lying catalogue accepted")
	}
	var forged repositorySignedFixture
	_ = json.Unmarshal(sign(m), &forged)
	forged.Signature[0] ^= 1
	raw, _ = json.Marshal(forged)
	bad.Index.Packages[0].SHA256 = packageformat.Hash(raw)
	if _, err := c.repositoryPackage(context.Background(), bad, entry.ID, entry.Version); err == nil {
		t.Fatal("catalogue hash bypassed signature")
	}
	if p.current.Checksum != entry.SHA256 || p.mutations != 1 {
		t.Fatal("failed download changed installed state")
	}
}

func TestProviderRepositoryBindingsValidateClaims(t *testing.T) {
	for _, claims := range [][]string{{"../bad"}, {"org.fixture.source", "org.fixture.source"}, make([]string, 129)} {
		c := &catalog{providers: []MetadataProvider{&repositoryProviderFixture{packageAdapterFixture: &packageAdapterFixture{id: "source"}, claims: claims}}}
		if _, err := c.providerRepositoryBindings(); err == nil {
			t.Fatal("invalid adapter claims accepted", claims)
		}
	}
	c := &catalog{providers: []MetadataProvider{&packageAdapterFixture{id: "local-only"}}}
	if bindings, err := c.providerRepositoryBindings(); err != nil || len(bindings) != 0 {
		t.Fatal("local installer granted repository dispatch")
	}
	if _, _, err := c.repositoryProvider("org.fixture.unknown"); err == nil {
		t.Fatal("unregistered package routed")
	}
}
