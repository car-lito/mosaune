// Package packageformat verifies bounded signed archives before compiling code.
package packageformat

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/protocol"
	"regexp"
	"strconv"
	"strings"
)

const MaxArchive = 9 << 20
const MaxCode = 8 << 20
const HealthCapability = "system.health.v1"

type Manifest struct {
	Schema      int                  `json:"schema"`
	Name        string               `json:"name"`
	Publisher   string               `json:"publisher"`
	Description string               `json:"description"`
	MinCore     string               `json:"minCore"`
	Adapter     string               `json:"adapter"`
	Descriptor  protocol.Descriptor  `json:"module"`
	Settings    configuration.Schema `json:"settings"`
	CodeSHA256  string               `json:"codeSha256"`
}
type Package struct {
	Manifest Manifest `json:"manifest"`
	Checksum string   `json:"checksum"`
	Code     []byte   `json:"-"`
	Archive  []byte   `json:"-"`
}

var semver = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})$`)

func Compare(a, b string) int {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(aa[i])
		y, _ := strconv.Atoi(bb[i])
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}
func Hash(raw []byte) string { s := sha256.Sum256(raw); return hex.EncodeToString(s[:]) }
func (m Manifest) Validate(core string) error {
	invalid := errors.New("Paquet fournisseur incompatible avec cette version de Mosaune.")
	if m.Schema != 1 || !semver.MatchString(core) || !semver.MatchString(m.MinCore) || Compare(m.MinCore, core) > 0 || m.Name == "" || len(m.Name) > 80 || m.Publisher != "Mosaune" || len(m.Description) > 1000 || m.Adapter != "tmdb.v1" || m.Descriptor.Validate() != nil || m.Settings.Validate() != nil {
		return invalid
	}
	// The first installed adapter is TMDB. Arbitrary domains, storage schemas and
	// executable community packages are not enabled by this pilot.
	d := m.Descriptor
	if d.ID != "mosaune.tmdb-search" || len(d.Dependencies) != 0 || len(d.Capabilities) != 3 || len(d.Services) != 1 || d.Services[0].Name != "tmdb.api.v1" || len(d.Services[0].Operations) != 1 || d.Services[0].Operations[0] != "get" {
		return invalid
	}
	caps := map[string]bool{}
	for _, c := range d.Capabilities {
		caps[c] = true
	}
	if !caps["metadata.search.v1"] || !caps["metadata.video-details.v1"] || !caps[HealthCapability] {
		return invalid
	}
	if len(m.Settings.Fields) != 1 {
		return invalid
	}
	f := m.Settings.Fields[0]
	if f.ID != "apiToken" || f.Type != "secret" || f.Format != "token" || f.MinLength != 20 || f.MaxLength != 2048 {
		return invalid
	}
	if len(m.CodeSHA256) != 64 {
		return invalid
	}
	if _, e := hex.DecodeString(m.CodeSHA256); e != nil {
		return invalid
	}
	return nil
}
func Parse(raw []byte, key ed25519.PublicKey, core string) (Package, error) {
	var p Package
	fail := errors.New("Archive fournisseur invalide, altérée ou non signée par la clé Mosaune reconnue.")
	if len(raw) > MaxArchive || len(key) != ed25519.PublicKeySize {
		return p, fail
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil || len(z.File) != 3 {
		return p, fail
	}
	files := map[string][]byte{}
	for _, f := range z.File {
		limit := 0
		switch f.Name {
		case "extension.json":
			limit = 64 << 10
		case "provider.wasm":
			limit = MaxCode
		case "signature.ed25519":
			limit = ed25519.SignatureSize
		default:
			return p, fail
		}
		if !f.Mode().IsRegular() || f.UncompressedSize64 > uint64(limit) || files[f.Name] != nil {
			return p, fail
		}
		reader, err := f.Open()
		if err != nil {
			return p, fail
		}
		data, err := io.ReadAll(io.LimitReader(reader, int64(limit+1)))
		reader.Close()
		if err != nil || len(data) > limit {
			return p, fail
		}
		files[f.Name] = data
	}
	manifest := files["extension.json"]
	// Authenticate bytes before parsing the manifest or compiling any Wasm.
	if !ed25519.Verify(key, manifest, files["signature.ed25519"]) {
		return p, fail
	}
	if protocol.DecodeOne(manifest, &p.Manifest) != nil {
		return p, fail
	}
	if err := p.Manifest.Validate(core); err != nil {
		return p, err
	}
	p.Code = files["provider.wasm"]
	if len(p.Code) < 8 || !bytes.Equal(p.Code[:8], []byte{0, 97, 115, 109, 1, 0, 0, 0}) || Hash(p.Code) != p.Manifest.CodeSHA256 {
		return Package{}, fail
	}
	p.Checksum = Hash(append(append([]byte{}, manifest...), p.Code...))
	p.Archive = raw
	return p, nil
}
func Create(m Manifest, code []byte, key ed25519.PrivateKey) ([]byte, error) {
	if len(key) != ed25519.PrivateKeySize || len(code) > MaxCode {
		return nil, errors.New("invalid key or code")
	}
	m.CodeSHA256 = Hash(code)
	if err := m.Validate(m.MinCore); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, file := range []struct {
		name string
		data []byte
	}{{"extension.json", raw}, {"provider.wasm", code}, {"signature.ed25519", ed25519.Sign(key, raw)}} {
		w, e := z.Create(file.name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(file.data); e != nil {
			return nil, e
		}
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
