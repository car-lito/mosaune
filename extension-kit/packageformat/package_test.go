package packageformat_test

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	f "mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/providers/tmdb"
	"testing"
)

func fixture() f.Manifest {
	return f.Manifest{Schema: 1, Name: "TMDB", Publisher: "Mosaune", Description: "Fixture", MinCore: "0.0.43", Adapter: "tmdb.v1", Descriptor: tmdb.Descriptor(), Settings: tmdb.Settings()}
}
func archive(t *testing.T, m f.Manifest, code []byte, key ed25519.PrivateKey, extra string) []byte {
	t.Helper()
	m.CodeSHA256 = f.Hash(code)
	raw, _ := json.Marshal(m)
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, file := range []struct {
		name string
		data []byte
	}{{"extension.json", raw}, {"provider.wasm", code}, {"signature.ed25519", ed25519.Sign(key, raw)}} {
		w, e := z.Create(file.name)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = w.Write(file.data)
	}
	if extra != "" {
		w, _ := z.Create(extra)
		_, _ = w.Write([]byte("extra"))
	}
	_ = z.Close()
	return b.Bytes()
}
func TestSignedPackagesValidateBeforeExecution(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	code := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	raw, e := f.Create(fixture(), code, key)
	if e != nil {
		t.Fatal(e)
	}
	pack, e := f.Parse(raw, pub, "0.0.43")
	if e != nil || pack.Checksum == "" || !bytes.Equal(pack.Code, code) {
		t.Fatal(pack, e)
	}
	if _, e = f.Parse(raw, other, "0.0.43"); e == nil {
		t.Fatal("unknown signer")
	}
	if _, e = f.Parse(raw, pub, "0.0.18"); e == nil {
		t.Fatal("old core")
	}
	for _, extra := range []string{"../escape", "provider.wasm", "directory/"} {
		if _, e = f.Parse(archive(t, fixture(), code, key, extra), pub, "0.0.43"); e == nil {
			t.Fatal("extra path", extra)
		}
	}
	for _, mutate := range []func(*f.Manifest){func(m *f.Manifest) { m.MinCore = "0.0.42" }, func(m *f.Manifest) { m.Descriptor.Capabilities[2] = "unapproved.service.v1" }, func(m *f.Manifest) { m.Adapter = "arbitrary.v1" }, func(m *f.Manifest) { m.Descriptor.Services[0].Operations = []string{"delete"} }, func(m *f.Manifest) { m.Settings.Fields[0].ID = "anotherSecret" }, func(m *f.Manifest) { m.Descriptor.Capabilities = []string{"system.health.v1"} }, func(m *f.Manifest) { m.MinCore = "999.0.0" }} {
		m := fixture()
		mutate(&m)
		if _, e = f.Parse(archive(t, m, code, key, ""), pub, "0.0.43"); e == nil {
			t.Fatal("unsupported signed declaration")
		}
	}
	if _, e = f.Parse(archive(t, fixture(), make([]byte, f.MaxCode+1), key, ""), pub, "0.0.43"); e == nil {
		t.Fatal("compressed bomb")
	}
	corrupted := append([]byte{}, raw...)
	corrupted[len(corrupted)/2] ^= 1
	if _, e = f.Parse(corrupted, pub, "0.0.43"); e == nil {
		t.Fatal("tampering accepted")
	}
}
