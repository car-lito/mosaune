package main

// The first external package format is deliberately declarative. No scripts,
// native binaries, CSS, file paths, network calls or SQL supplied by a package
// are executed. Future executable capabilities need their own isolated runtime.
import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const extensionLimit = 256 * 1024

type extensionManifest struct {
	Schema      int            `json:"schema"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Publisher   string         `json:"publisher"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Family      string         `json:"family"`
	MinCore     string         `json:"minCore"`
	API         int            `json:"api"`
	Requires    []string       `json:"requires"`
	Permissions []string       `json:"permissions"`
	Theme       extensionTheme `json:"theme"`
}
type extensionTheme struct {
	Light    map[string]string            `json:"light,omitempty"`
	Dark     map[string]string            `json:"dark,omitempty"`
	Palette  map[string]themeModes        `json:"palette,omitempty"`
	Fonts    map[string][]string          `json:"fonts,omitempty"`
	Radii    map[string]string            `json:"radii,omitempty"`
	Tokens   map[string]string            `json:"tokens,omitempty"`
	Sections map[string]map[string]string `json:"sections,omitempty"`
}
type extensionPackage struct {
	Manifest extensionManifest            `json:"manifest"`
	Checksum string                       `json:"checksum"`
	Enabled  bool                         `json:"enabled"`
	Users    []string                     `json:"users"`
	Preview  map[string]map[string]string `json:"preview,omitempty"`
}
type extensionAppearance struct {
	Selected  string `json:"selected"`
	Effective string `json:"effective"`
	Stamp     string `json:"stamp"`
	SafeMode  bool   `json:"safeMode"`
}

func migrateExtensions(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 10 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS extensions(id TEXT PRIMARY KEY, manifest TEXT NOT NULL, checksum TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS extension_grants(user_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,PRIMARY KEY(user_id,extension_id));
 CREATE TABLE IF NOT EXISTS extension_preferences(user_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,theme_id TEXT REFERENCES extensions(id) ON DELETE SET NULL);
 PRAGMA user_version=10;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

var extensionID = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)+$`)
var extensionVersion = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})$`)
var extensionColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var themeKeys = []string{"canvas", "surface", "ink", "muted", "line", "accent", "accentInk", "sidebar", "sidebarInk"}

func compareExtensionVersions(a, b string) int {
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
func colorLuminance(color string) float64 {
	var c [3]float64
	for i := range c {
		n, _ := strconv.ParseUint(color[1+2*i:3+2*i], 16, 8)
		v := float64(n) / 255
		if v <= 0.04045 {
			c[i] = v / 12.92
		} else {
			c[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return .2126*c[0] + .7152*c[1] + .0722*c[2]
}
func colorContrast(a, b string) float64 {
	x, y := colorLuminance(a), colorLuminance(b)
	if x < y {
		x, y = y, x
	}
	return (x + .05) / (y + .05)
}
func validateExtension(m extensionManifest) error {
	if m.Schema != 1 || (m.API != 1 && m.API != 2) || m.Family != "interface" {
		return errors.New("Type ou API non pris en charge : thèmes déclaratifs v1 et v2 uniquement.")
	}
	if len(m.ID) > 80 || !extensionID.MatchString(m.ID) || strings.HasPrefix(m.ID, "mosaune.") {
		return errors.New("Identifiant invalide ou réservé à Mosaune.")
	}
	if len(strings.TrimSpace(m.Name)) == 0 || len(m.Name) > 80 || len(strings.TrimSpace(m.Publisher)) == 0 || len(m.Publisher) > 80 || len(m.Description) > 600 {
		return errors.New("Nom, éditeur ou description invalide.")
	}
	if !extensionVersion.MatchString(m.Version) || !extensionVersion.MatchString(m.MinCore) || compareExtensionVersions(m.MinCore, version) > 0 {
		return errors.New("Version invalide ou version de Mosaune trop ancienne pour ce paquet.")
	}
	if len(m.Requires) != 1 || m.Requires[0] != fmt.Sprintf("ui.theme.v%d", m.API) {
		return errors.New("Le paquet doit demander la capacité ui.theme correspondant à son API.")
	}
	if len(m.Permissions) != 0 {
		return errors.New("Un thème déclaratif ne peut demander aucun accès au réseau, aux fichiers ou aux comptes.")
	}
	if m.API == 2 {
		return validateThemeV2(m.Theme)
	}
	if len(m.Theme.Palette) > 0 || len(m.Theme.Fonts) > 0 || len(m.Theme.Radii) > 0 || len(m.Theme.Tokens) > 0 || len(m.Theme.Sections) > 0 {
		return errors.New("Utilisez l’API 2 pour les réglages détaillés du thème.")
	}
	for _, palette := range []map[string]string{m.Theme.Light, m.Theme.Dark} {
		if len(palette) != len(themeKeys) {
			return errors.New("Les palettes claire et sombre doivent définir les neuf couleurs du contrat.")
		}
		for _, k := range themeKeys {
			if !extensionColor.MatchString(palette[k]) {
				return fmt.Errorf("Couleur %s invalide : utilisez #RRGGBB, sans CSS ni URL.", k)
			}
		}
		for _, pair := range [][2]string{{"ink", "canvas"}, {"ink", "surface"}, {"muted", "canvas"}, {"muted", "surface"}, {"accentInk", "accent"}, {"sidebarInk", "sidebar"}} {
			if colorContrast(palette[pair[0]], palette[pair[1]]) < 4.5 {
				return fmt.Errorf("Contraste insuffisant entre %s et %s. Le texte doit rester lisible.", pair[0], pair[1])
			}
		}
	}
	return nil
}

func parseExtensionPackage(data []byte) (extensionPackage, error) {
	var p extensionPackage
	if len(data) > extensionLimit {
		return p, errors.New("Paquet trop volumineux (256 Ko maximum).")
	}
	archive, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return p, errors.New("Sélectionnez un paquet ZIP Mosaune valide.")
	}
	if len(archive.File) != 1 || archive.File[0].Name != "extension.json" || !archive.File[0].Mode().IsRegular() || archive.File[0].UncompressedSize64 > extensionLimit {
		return p, errors.New("Le paquet doit contenir uniquement extension.json à sa racine (256 Ko maximum), sans dossier ni fichier exécutable.")
	}
	stream, e := archive.File[0].Open()
	if e != nil {
		return p, errors.New("Paquet illisible.")
	}
	defer stream.Close()
	raw, e := io.ReadAll(io.LimitReader(stream, extensionLimit+1))
	if e != nil || len(raw) > extensionLimit {
		return p, errors.New("Manifeste illisible ou trop volumineux.")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&p.Manifest); e != nil {
		return p, errors.New("Manifeste invalide ou contenant des champs non pris en charge.")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return p, errors.New("Le manifeste doit contenir un seul objet JSON.")
	}
	if e = validateExtension(p.Manifest); e != nil {
		return p, e
	}
	canonical, _ := json.Marshal(p.Manifest)
	p.Checksum = digest(string(canonical))
	p.Users = []string{}
	p.Preview = themePreviewFor(p.Manifest)
	return p, nil
}
func (c *catalog) extension(ctx context.Context, id string) (extensionPackage, error) {
	var p extensionPackage
	var raw string
	e := c.db.QueryRowContext(ctx, "SELECT manifest,checksum,enabled FROM extensions WHERE id=?", id).Scan(&raw, &p.Checksum, &p.Enabled)
	if e == nil {
		e = json.Unmarshal([]byte(raw), &p.Manifest)
	}
	p.Users = []string{}
	return p, e
}
func (c *catalog) canUseExtension(ctx context.Context, id string) bool {
	u := viewer(ctx)
	if u == nil {
		return false
	}
	if u.Admin {
		return true
	}
	var n int
	return c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM extension_grants WHERE user_id=? AND extension_id=?", u.ID, id).Scan(&n) == nil && n == 1
}
func (c *catalog) extensionAppearance(ctx context.Context) (extensionAppearance, extensionPackage, error) {
	a := extensionAppearance{SafeMode: c.extensionSafeMode}
	var p extensionPackage
	var selected sql.NullString
	e := c.db.QueryRowContext(ctx, "SELECT theme_id FROM extension_preferences WHERE user_id=?", userID(ctx)).Scan(&selected)
	if e != nil && e != sql.ErrNoRows {
		return a, p, e
	}
	a.Selected = selected.String
	if a.Selected == "" || c.extensionSafeMode || !c.canUseExtension(ctx, a.Selected) {
		return a, p, nil
	}
	p, e = c.extension(ctx, a.Selected)
	if e != nil {
		return a, p, e
	}
	if !p.Enabled {
		return a, p, nil
	}
	if e = validateExtension(p.Manifest); e != nil {
		return a, extensionPackage{}, nil
	}
	a.Effective = a.Selected
	a.Stamp = digest(userID(ctx) + ":" + a.Selected + ":" + p.Checksum)
	return a, p, nil
}
func extensionCSS(p extensionPackage, stamp string) string {
	return renderTheme(themeForManifest(p.Manifest), fmt.Sprintf(`.app-shell[data-extension-theme="%s"]`, stamp))
}
