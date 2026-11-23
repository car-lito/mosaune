package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/packageformat"
	"testing"
)

// This in-memory adapter tests HTTP dispatch, not archive trust. Real signed
// TMDB archives and failed signatures are covered by provider_packages_test.
type packageAdapterFixture struct {
	MetadataProvider
	id                string
	current, previous packageformat.Package
	reads, mutations  int
	fail              bool
}

func (p *packageAdapterFixture) Info() ProviderInfo    { return ProviderInfo{ID: p.id} }
func (p *packageAdapterFixture) packageSigner() string { return "Fixture signer: " + p.id }
func (p *packageAdapterFixture) parseProviderPackage(raw []byte) (packageformat.Package, error) {
	p.reads++
	if string(raw) != "1.0.0" && string(raw) != "2.0.0" {
		return packageformat.Package{}, errors.New("Invalid fixture archive")
	}
	var pack packageformat.Package
	pack.Manifest.Descriptor.ID = "org.fixture." + p.id
	pack.Manifest.Descriptor.Version = string(raw)
	pack.Checksum = packageformat.Hash(append([]byte(p.id), raw...))
	pack.Archive = raw
	return pack, nil
}
func (p *packageAdapterFixture) installedPackage(context.Context) (packageformat.Package, error) {
	if p.current.Checksum == "" {
		return packageformat.Package{}, sql.ErrNoRows
	}
	return p.current, nil
}
func (p *packageAdapterFixture) packageState(context.Context) (providerPackageState, error) {
	s := providerPackageState{PreviousChecksum: p.previous.Checksum}
	if p.current.Checksum != "" {
		pack := p.current
		s.Package = &pack
	}
	return s, nil
}
func (p *packageAdapterFixture) installPackage(_ context.Context, pack packageformat.Package, expected string, rollback bool) error {
	if p.fail || expected != p.current.Checksum || (rollback && pack.Checksum != p.previous.Checksum) {
		return errors.New("Stale or rejected mutation")
	}
	if p.current.Checksum != "" && !rollback && packageformat.Compare(pack.Manifest.Descriptor.Version, p.current.Manifest.Descriptor.Version) <= 0 {
		return errors.New("Older candidate")
	}
	p.previous, p.current = p.current, pack
	p.mutations++
	return nil
}
func (p *packageAdapterFixture) rollbackPackage(ctx context.Context, current, previous string) error {
	if previous == "" || previous != p.previous.Checksum {
		return errors.New("Stale previous package")
	}
	return p.installPackage(ctx, p.previous, current, true)
}
func (p *packageAdapterFixture) removePackage(_ context.Context, checksum string) error {
	if p.fail || checksum == "" || checksum != p.current.Checksum {
		return errors.New("Stale removal")
	}
	p.current, p.previous = packageformat.Package{}, packageformat.Package{}
	p.mutations++
	return nil
}

func TestProviderPackageContractRoutesAndIsolation(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	first, second := &packageAdapterFixture{id: "external"}, &packageAdapterFixture{id: "other"}
	c.providers = append(c.providers, first, second)
	list := admin.request("GET", "/api/v1/provider-packages", nil)
	var managers []map[string]string
	if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &managers) != nil || len(managers) != 3 || managers[0]["id"] != "tmdb" || managers[1]["id"] != "external" || managers[2]["id"] != "other" {
		t.Fatal("package contracts not discoverable", list.Body.String())
	}
	tmdbBefore, err := c.provider("tmdb").(*tmdbProvider).installedPackage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	preview := providerUpload(admin, "external/preview", []byte("1.0.0"), "", "")
	if preview.Code != 200 {
		t.Fatal(preview.Code, preview.Body.String())
	}
	var result struct {
		Package packageformat.Package `json:"package"`
		Signer  string                `json:"signer"`
	}
	if json.Unmarshal(preview.Body.Bytes(), &result) != nil || result.Signer != "Fixture signer: external" {
		t.Fatal(preview.Body.String())
	}
	a := result.Package
	if first.mutations != 0 || second.reads != 0 {
		t.Fatal("preview mutated or targeted another adapter")
	}
	for _, tc := range []struct {
		raw, checksum string
		status        int
	}{{"bad", "", 400}, {"1.0.0", "wrong", 409}, {"1.0.0", a.Checksum, 201}} {
		if w := providerUpload(admin, "external/install", []byte(tc.raw), tc.checksum, ""); w.Code != tc.status {
			t.Fatal(tc, w.Code)
		}
	}
	if w := providerUpload(admin, "external/preview", []byte("1.0.0"), "", ""); w.Code != 409 {
		t.Fatal("same version accepted")
	}
	b, _ := first.parseProviderPackage([]byte("2.0.0"))
	if w := providerUpload(admin, "external/install", b.Archive, b.Checksum, "stale"); w.Code != 409 {
		t.Fatal("stale install accepted")
	}
	first.fail = true
	if w := providerUpload(admin, "external/install", b.Archive, b.Checksum, a.Checksum); w.Code != 409 || first.current.Checksum != a.Checksum {
		t.Fatal("failed install changed state")
	}
	first.fail = false
	if w := providerUpload(admin, "external/install", b.Archive, b.Checksum, a.Checksum); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("POST", "/api/v1/provider-packages/external/rollback", map[string]string{"checksum": b.Checksum, "previousChecksum": "stale"}); w.Code != 409 {
		t.Fatal("stale rollback accepted")
	}
	if w := admin.request("POST", "/api/v1/provider-packages/external/rollback", map[string]string{"checksum": b.Checksum, "previousChecksum": a.Checksum}); w.Code != 204 {
		t.Fatal(w.Code)
	}
	state := admin.request("GET", "/api/v1/provider-packages/external", nil)
	var got providerPackageState
	if state.Code != 200 || json.Unmarshal(state.Body.Bytes(), &got) != nil || got.Package == nil || got.Package.Checksum != a.Checksum {
		t.Fatal(state.Body.String())
	}
	admin.create("Reader", accountTestPassword, map[string]libraryGrant{})
	reader := &accountClient{t: t, h: h}
	reader.enter("login", map[string]string{"name": "Reader", "password": accountTestPassword})
	if w := reader.request("GET", "/api/v1/provider-packages", nil); w.Code != 403 {
		t.Fatal("package managers exposed to ordinary account", w.Code)
	}
	calls, changes := first.reads, first.mutations
	for _, method := range []string{"GET", "DELETE"} {
		if w := reader.request(method, "/api/v1/provider-packages/external", nil); w.Code != 403 {
			t.Fatal("permission", method, w.Code)
		}
	}
	for _, action := range []string{"preview", "install", "rollback"} {
		if w := providerUpload(reader, "external/"+action, b.Archive, b.Checksum, a.Checksum); w.Code != 403 {
			t.Fatal("permission", action, w.Code)
		}
	}
	if calls != first.reads || changes != first.mutations {
		t.Fatal("unauthorized request reached adapter")
	}
	for _, id := range []string{"missing", "musicbrainz"} {
		if w := admin.request("GET", "/api/v1/provider-packages/"+id, nil); w.Code != 404 {
			t.Fatal(id, w.Code)
		}
		if w := providerUpload(admin, id+"/install", b.Archive, b.Checksum, ""); w.Code != 404 {
			t.Fatal(id, w.Code)
		}
	}
	if w := admin.request("DELETE", "/api/v1/provider-packages/external", map[string]string{"checksum": b.Checksum}); w.Code != 409 {
		t.Fatal("stale removal accepted")
	}
	if w := admin.request("DELETE", "/api/v1/provider-packages/external", map[string]string{"checksum": a.Checksum}); w.Code != 204 {
		t.Fatal(w.Code)
	}
	tmdbAfter, err := c.provider("tmdb").(*tmdbProvider).installedPackage(context.Background())
	if err != nil || tmdbAfter.Checksum != tmdbBefore.Checksum || second.reads != 0 || second.mutations != 0 || first.current.Checksum != "" {
		t.Fatal("cross-provider mutation", err)
	}
}

func TestProviderPackageContractTMDBNamedUpload(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	p := c.provider("tmdb").(*tmdbProvider)
	removeTMDBPackageFixture(t, p)
	preview := providerUpload(admin, "tmdb/preview", providerIdentifiersExample, "", "")
	if preview.Code != 200 {
		t.Fatal(preview.Code, preview.Body.String())
	}
	var result struct {
		Package packageformat.Package `json:"package"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w := providerUpload(admin, "tmdb/install", providerIdentifiersExample, result.Package.Checksum, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	pack, err := p.installedPackage(context.Background())
	if err != nil || pack.Checksum != result.Package.Checksum {
		t.Fatal("named route did not install real signed package", err)
	}
}
