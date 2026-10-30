package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/providers/tmdb"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	keyPath := flag.String("key", "", "private signing key (never committed)")
	public := flag.String("public", "", "write or verify pinned public key")
	create := flag.Bool("create-key", false, "create a new key only if absent")
	out := flag.String("out", "", "package path")
	codePath := flag.String("code", "", "Wasm program path")
	v := flag.String("version", "0.5.0", "provider version")
	flag.Parse()
	if *keyPath == "" || *public == "" {
		return fmt.Errorf("key and public paths required")
	}
	raw, err := os.ReadFile(*keyPath)
	if os.IsNotExist(err) && *create {
		_, key, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			return e
		}
		raw = []byte(base64.StdEncoding.EncodeToString(key))
		f, e := os.OpenFile(*keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		_, e = f.Write(raw)
		f.Close()
		if e != nil {
			return e
		}
		err = nil
	}
	if err != nil {
		return err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid signing key")
	}
	pub := []byte(base64.StdEncoding.EncodeToString(ed25519.PrivateKey(key).Public().(ed25519.PublicKey)))
	existing, e := os.ReadFile(*public)
	if e == nil && strings.TrimSpace(string(existing)) != string(pub) {
		return fmt.Errorf("public key differs; rotation must be explicit")
	}
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	if e = os.WriteFile(*public, pub, 0644); e != nil {
		return e
	}
	if *out == "" {
		return nil
	}
	code, err := os.ReadFile(*codePath)
	if err != nil {
		return err
	}
	d := tmdb.Descriptor()
	d.Version = *v
	m := packageformat.Manifest{Schema: 1, Name: "TMDB", Publisher: "Mosaune", Description: "Recherche, fiches, découverte, profils et affiches alternatives de films et séries. Programme indépendant du serveur Mosaune.", MinCore: "0.0.43", Adapter: "tmdb.v1", Descriptor: d, Settings: tmdb.Settings()}
	data, err := packageformat.Create(m, code, key)
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, data, 0644); err != nil {
		return err
	}
	fmt.Println("Paquet signé créé :", *out)
	return nil
}
