package packageformat_test

import (
	"crypto/ed25519"
	"crypto/rand"
	discovery "mosaune/extensionkit/discovery/v1"
	f "mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/providers/tmdb"
	"testing"
)

func TestSignedDiscoverySurface(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	code := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	makeManifest := func() f.Manifest {
		m := fixture()
		s := tmdb.DiscoverySurface()
		m.Discovery = &s
		m.MinCore = "0.0.52"
		return m
	}
	m := makeManifest()
	raw, err := f.Create(m, code, key)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := f.Parse(raw, pub, "0.0.52")
	if err != nil || pack.Manifest.Discovery == nil || len(pack.Manifest.Discovery.Shelves) != 3 {
		t.Fatal("declaration lost", err)
	}
	if _, err = f.Parse(raw, pub, "0.0.51"); err == nil {
		t.Fatal("old host accepted new declaration")
	}
	m.Discovery = &discovery.Surface{Version: 1}
	raw, err = f.Create(m, code, key)
	if err != nil {
		t.Fatal(err)
	}
	pack, err = f.Parse(raw, pub, "0.0.52")
	if err != nil || pack.Manifest.Discovery == nil || len(pack.Manifest.Discovery.Search) != 0 {
		t.Fatal("explicit empty lost", err)
	}
	for name, mutate := range map[string]func(*f.Manifest){
		"old minimum":      func(m *f.Manifest) { m.MinCore = "0.0.51" },
		"unsupported kind": func(m *f.Manifest) { m.Discovery.Search[0].Kind = "music" },
		"duplicate":        func(m *f.Manifest) { m.Discovery.Search = append(m.Discovery.Search, m.Discovery.Search[0]) },
		"unknown shelf":    func(m *f.Manifest) { m.Discovery.Shelves[0].ID = "arbitrary" },
		"wrong selector":   func(m *f.Manifest) { m.Discovery.Shelves[0].Selector = "contributors" },
		"wrong shelf kind": func(m *f.Manifest) { m.Discovery.Shelves[0].Kind = "series" },
		"missing capability": func(m *f.Manifest) {
			m.Descriptor.Capabilities = []string{"metadata.search.v1", "metadata.video-details.v1", "system.health.v1"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := makeManifest()
			mutate(&m)
			if _, err := f.Parse(archive(t, m, code, key, ""), pub, "0.0.52"); err == nil {
				t.Fatal("invalid signed declaration accepted")
			}
		})
	}
}
