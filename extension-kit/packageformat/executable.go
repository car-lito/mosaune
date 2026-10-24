package packageformat

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mosaune/extensionkit/protocol"
	"strings"
)

// Contribution names the host contract and its target. The package reader does
// not decide which contributions the host supports or grant any permissions.
type Contribution struct {
	Role   string `json:"role"`
	Target string `json:"target"`
}

type ExecutableManifest struct {
	Schema        int                 `json:"schema"`
	Name          string              `json:"name"`
	Publisher     string              `json:"publisher"`
	Description   string              `json:"description"`
	MinCore       string              `json:"minCore"`
	Module        protocol.Descriptor `json:"module"`
	Contributions []Contribution      `json:"contributions"`
	CodeSHA256    string              `json:"codeSha256"`
}

type Executable struct {
	Manifest ExecutableManifest `json:"manifest"`
	Checksum string             `json:"checksum"`
	Code     []byte             `json:"-"`
}

func (m ExecutableManifest) Validate(core string) error {
	invalid := errors.New("Manifeste du module invalide ou incompatible avec cette version de Mosaune.")
	if m.Schema != 2 || !semver.MatchString(core) || !semver.MatchString(m.MinCore) || Compare(m.MinCore, core) > 0 || strings.TrimSpace(m.Name) == "" || len(m.Name) > 80 || strings.TrimSpace(m.Publisher) == "" || len(m.Publisher) > 80 || len(m.Description) > 1000 || m.Module.Validate() != nil || len(m.Contributions) == 0 || len(m.Contributions) > 16 || len(m.CodeSHA256) != 64 {
		return invalid
	}
	if _, err := hex.DecodeString(m.CodeSHA256); err != nil {
		return invalid
	}
	seen := map[Contribution]bool{}
	for _, c := range m.Contributions {
		if seen[c] || strings.TrimSpace(c.Target) == "" || len(c.Target) > 80 {
			return invalid
		}
		found := false
		for _, cap := range m.Module.Capabilities {
			if cap == c.Role {
				found = true
			}
		}
		if !found {
			return invalid
		}
		seen[c] = true
	}
	return nil
}

// ParseExecutable is for administrator-imported, UNVERIFIED local packages.
// Hashes bind preview to installation and detect damage, not publisher identity.
// No archive entry is ever extracted onto the filesystem or executed here.
func ParseExecutable(raw []byte, core string) (Executable, error) {
	var p Executable
	fail := errors.New("Archive de module invalide ou endommagée.")
	if len(raw) > MaxArchive {
		return p, fail
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(z.File) != 2 {
		return p, fail
	}
	files := map[string][]byte{}
	for _, f := range z.File {
		limit := 0
		switch f.Name {
		case "extension.json":
			limit = 64 << 10
		case "module.wasm":
			limit = MaxCode
		default:
			return p, fail
		}
		if !f.Mode().IsRegular() || f.UncompressedSize64 > uint64(limit) || files[f.Name] != nil {
			return p, fail
		}
		r, e := f.Open()
		if e != nil {
			return p, fail
		}
		b, e := io.ReadAll(io.LimitReader(r, int64(limit+1)))
		r.Close()
		if e != nil || len(b) > limit {
			return p, fail
		}
		files[f.Name] = b
	}
	if protocol.DecodeOne(files["extension.json"], &p.Manifest) != nil {
		return p, fail
	}
	if err = p.Manifest.Validate(core); err != nil {
		return p, err
	}
	p.Code = files["module.wasm"]
	if len(p.Code) < 8 || !bytes.Equal(p.Code[:8], []byte{0, 97, 115, 109, 1, 0, 0, 0}) || Hash(p.Code) != p.Manifest.CodeSHA256 {
		return Executable{}, fail
	}
	p.Checksum = Hash(raw)
	return p, nil
}

func CreateExecutable(m ExecutableManifest, code []byte) ([]byte, error) {
	m.CodeSHA256 = Hash(code)
	if err := m.Validate(m.MinCore); err != nil {
		return nil, err
	}
	if len(code) > MaxCode {
		return nil, errors.New("module too large")
	}
	manifest, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, f := range []struct {
		name string
		data []byte
	}{{"extension.json", manifest}, {"module.wasm", code}} {
		w, e := z.Create(f.name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(f.data); e != nil {
			return nil, e
		}
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	if _, err = ParseExecutable(out.Bytes(), m.MinCore); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
