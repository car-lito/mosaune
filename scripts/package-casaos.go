//go:build ignore

// Build the small legacy CasaOS catalogue deterministically from reviewed files.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var data bytes.Buffer
	archive := zip.NewWriter(&data)
	for _, name := range []string{"README.md", "Apps/Mosaune/docker-compose.yml", "Apps/Mosaune/icon.svg"} {
		content, err := os.ReadFile(filepath.Join("deploy", "casaos", filepath.FromSlash(name)))
		if err != nil {
			panic(err)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0644)
		file, err := archive.CreateHeader(header)
		if err != nil {
			panic(err)
		}
		if _, err := file.Write(content); err != nil {
			panic(err)
		}
	}
	if err := archive.Close(); err != nil {
		panic(err)
	}
	if err := os.WriteFile("deploy/mosaune-casaos.zip", data.Bytes(), 0644); err != nil {
		panic(err)
	}
	fmt.Println("Catalogue CasaOS préparé : deploy/mosaune-casaos.zip")
}
