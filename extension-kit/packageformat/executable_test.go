package packageformat_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	f "mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"testing"
)

func executableFixture() f.ExecutableManifest {
	return f.ExecutableManifest{Schema: 2, Name: "Community", Publisher: "Declared author", MinCore: "0.0.31", Module: protocol.Descriptor{ID: "example.recognition", Version: "1.0.0", Capabilities: []string{"media.recognition.v1"}}, Contributions: []f.Contribution{{Role: "media.recognition.v1", Target: "films"}}}
}
func TestLocalExecutableFormatIntegrityAndCompatibility(t *testing.T) {
	code := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	m := executableFixture()
	raw, err := f.CreateExecutable(m, code)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.ParseExecutable(raw, "0.0.31")
	if err != nil || p.Checksum != f.Hash(raw) || !bytes.Equal(p.Code, code) {
		t.Fatal(p, err)
	}
	if _, err = f.ParseExecutable(raw, "0.0.30"); err == nil {
		t.Fatal("future core accepted")
	}
	for _, mutate := range []func(*f.ExecutableManifest){
		func(m *f.ExecutableManifest) { m.Schema = 1 }, func(m *f.ExecutableManifest) { m.Module.ID = "../../bad" }, func(m *f.ExecutableManifest) { m.Contributions = append(m.Contributions, m.Contributions[0]) }, func(m *f.ExecutableManifest) { m.Contributions[0].Role = "other.role.v1" }, func(m *f.ExecutableManifest) { m.MinCore = "garbage" },
	} {
		m := executableFixture()
		mutate(&m)
		if _, err = f.CreateExecutable(m, code); err == nil {
			t.Fatal("invalid manifest accepted", m)
		}
	}
}
func TestLocalExecutableRejectsArchiveAbuse(t *testing.T) {
	code := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	m := executableFixture()
	m.CodeSHA256 = f.Hash(code)
	manifest, _ := json.Marshal(m)
	for _, test := range []struct {
		name           string
		manifest, code []byte
		extra          string
	}{
		{"path", manifest, code, "../module.wasm"},
		{"duplicate", manifest, code, "extension.json"},
		{"digest", manifest, append(code, 0), ""},
		{"unknown manifest field", append(manifest[:len(manifest)-1], []byte(`,"execute":"host"}`)...), code, ""},
		{"oversized code", manifest, make([]byte, f.MaxCode+1), ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			z := zip.NewWriter(&out)
			w, _ := z.Create("extension.json")
			w.Write(test.manifest)
			w, _ = z.Create("module.wasm")
			w.Write(test.code)
			if test.extra != "" {
				z.Create(test.extra)
			}
			z.Close()
			if _, err := f.ParseExecutable(out.Bytes(), "0.0.31"); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
