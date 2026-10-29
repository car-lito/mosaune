package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Screen manifests use the same installer and permissions as community pages.
//
//go:embed extensions/ui-*/extension.json
var suppliedScreens embed.FS

//go:embed examples/cinema-compact/extension.json
var compactScreen []byte

func screenExamplePackage() extensionPackage {
	var m extensionManifest
	_ = json.Unmarshal(compactScreen, &m)
	return extensionPackage{Manifest: m}
}

func screenPackages() []extensionPackage {
	out := []extensionPackage{}
	dirs, _ := suppliedScreens.ReadDir("extensions")
	for _, d := range dirs {
		raw, e := suppliedScreens.ReadFile("extensions/" + d.Name() + "/extension.json")
		if e != nil {
			panic(e)
		}
		var m extensionManifest
		if e = json.Unmarshal(raw, &m); e != nil {
			panic(e)
		}
		if e = validateExtension(m); e != nil {
			panic(e)
		}
		canonical, _ := json.Marshal(m)
		out = append(out, extensionPackage{Manifest: m, Checksum: digest(string(canonical)), Enabled: true, Users: []string{}})
	}
	return out
}
func screenArchive(p extensionPackage) []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, _ := z.Create("extension.json")
	raw, _ := json.Marshal(p.Manifest)
	_, _ = f.Write(raw)
	_ = z.Close()
	return b.Bytes()
}
func screenArchivePath(p extensionPackage) string {
	return "packages/" + p.Manifest.ID + ".mosaune.zip"
}
func migrateUIScreens(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 27 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, p := range screenPackages() {
		raw, _ := json.Marshal(p.Manifest)
		if _, e = tx.Exec("INSERT OR IGNORE INTO extensions(id,manifest,checksum,enabled) VALUES(?,?,?,1)", p.Manifest.ID, string(raw), p.Checksum); e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT OR IGNORE INTO extension_grants(user_id,extension_id) SELECT accounts.id,extensions.id FROM accounts,extensions WHERE extensions.id=? AND extensions.checksum=?", p.Manifest.ID, p.Checksum); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=27"); e != nil {
		return e
	}
	return tx.Commit()
}

// New accounts get only exact supplied packages, never an arbitrary package using their ID.
func grantSuppliedScreens(ctx context.Context, tx *sql.Tx, user string) error {
	for _, p := range screenPackages() {
		if _, e := tx.ExecContext(ctx, "INSERT OR IGNORE INTO extension_grants(user_id,extension_id) SELECT ?,id FROM extensions WHERE id=? AND checksum=?", user, p.Manifest.ID, p.Checksum); e != nil {
			return e
		}
	}
	return nil
}
func validUIFeed(b uiBlock) bool {
	if !slices.Contains([]string{"home", "films", "series", "music"}, b.View) || b.Target != "" || len(b.Facts) > 0 || !slices.Contains([]string{"", "shelf", "grid", "spotlight"}, b.Display) {
		return false
	}
	if !slices.Contains([]string{"recent", "resume", "favorites", "watchlist", "remote"}, b.Source) {
		return false
	}
	if b.Source != "remote" {
		return b.Provider == "" && b.Shelf == ""
	}
	// These are versioned host service names, not URLs or executable instructions.
	if b.Provider == "tmdb" {
		return b.View == "films" && (b.Shelf == "now_playing" || b.Shelf == "upcoming") || b.View == "series" && b.Shelf == "popular"
	}
	return b.Provider == "musicbrainz" && b.View == "music" && b.Shelf == "upcoming"
}
func uiNavigationOrder(p uiPage) int {
	for i, b := range uiBuiltins() {
		if p.Screen == b.ID || p.Owner == "" && p.ID == b.ID {
			return i
		}
	}
	return 100
}

// Caller holds extensionMu. A screen has one active owner, not an implicit priority.
func (c *catalog) checkUIScreenConflicts(ctx context.Context, m extensionManifest) error {
	claims := map[string]bool{}
	for _, p := range m.Pages {
		if p.Screen != "" {
			claims[p.Screen] = true
		}
	}
	if len(claims) == 0 {
		return nil
	}
	rows, e := c.db.QueryContext(ctx, "SELECT manifest FROM extensions WHERE enabled=1 AND id<>?", m.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return e
		}
		var other extensionManifest
		if e = json.Unmarshal([]byte(raw), &other); e != nil {
			return errors.New("Extension existante illisible.")
		}
		for _, p := range other.Pages {
			if claims[p.Screen] {
				return fmt.Errorf("La page %s est déjà fournie par %s. Désactivez ce paquet avant de choisir son remplacement.", p.Screen, other.Name)
			}
		}
	}
	return rows.Err()
}
