//go:build ignore

// Package reviewed component archives into a family. No code is executed here.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: go run scripts/package-family.go family.json output.zip")
	}
	raw, e := os.ReadFile(os.Args[1])
	if e != nil {
		return e
	}
	if len(raw) > 64<<10 {
		return fmt.Errorf("manifest too large")
	}
	var manifest map[string]json.RawMessage
	if e = json.Unmarshal(raw, &manifest); e != nil {
		return e
	}
	type member struct {
		Slot   string `json:"slot"`
		File   string `json:"file"`
		SHA256 string `json:"sha256"`
	}
	var components []member
	var screen member
	if e = json.Unmarshal(manifest["components"], &components); e != nil {
		return e
	}
	if e = json.Unmarshal(manifest["screen"], &screen); e != nil {
		return e
	}
	if len(components) < 1 || len(components) > 7 {
		return fmt.Errorf("invalid component count")
	}
	files := map[string][]byte{}
	size := 0
	read := func(ref *member) error {
		if ref.File == "family.json" || ref.File == "" || strings.ContainsAny(ref.File, "/\\:") || strings.Contains(ref.File, "..") || files[ref.File] != nil {
			return fmt.Errorf("invalid or duplicate member name")
		}
		f, e := os.Open(filepath.Join(filepath.Dir(os.Args[1]), ref.File))
		if e != nil {
			return e
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, (9<<20)+1))
		if e != nil {
			return e
		}
		size += len(b)
		if len(b) > 9<<20 || size > 64<<20 {
			return fmt.Errorf("archive limits exceeded")
		}
		sum := sha256.Sum256(b)
		ref.SHA256 = hex.EncodeToString(sum[:])
		files[ref.File] = b
		return nil
	}
	for i := range components {
		if e = read(&components[i]); e != nil {
			return e
		}
	}
	if e = read(&screen); e != nil {
		return e
	}
	manifest["components"], _ = json.Marshal(components)
	manifest["screen"], _ = json.Marshal(screen)
	files["family.json"], e = json.MarshalIndent(manifest, "", "  ")
	if e != nil {
		return e
	}
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, name := range names {
		f, e := z.Create(name)
		if e != nil {
			return e
		}
		if _, e = f.Write(files[name]); e != nil {
			return e
		}
	}
	if e = z.Close(); e != nil {
		return e
	}
	if out.Len() > 32<<20 {
		return fmt.Errorf("archive exceeds 32 MiB")
	}
	if e = os.WriteFile(os.Args[2], out.Bytes(), 0600); e != nil {
		return e
	}
	fmt.Println("Archive prepared; Mosaune will verify contracts and compatibility before import.")
	return nil
}
