package main

// Repositories advertise packages; they never grant execution rights. Downloaded
// bytes still pass their package validator and host policy. A repository hash
// verifies integrity, not the identity of an executable module's publisher.
import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mosaune/extensionkit/packageformat"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const repositoryIndexLimit = 256 * 1024
const repositoryIndexFile = "mosaune-repository.json"
const bundledRepositoryID = "mosaune-bundled"
const officialRepositoryURL = "https://github.com/car-lito/mosaune-extensions"

// Use the same identity as a source added manually in 0.0.20: no duplicate
// source or discarded cache on upgrade. The official flag is never supplied
// by a remote catalogue.
func officialRepository() extensionRepository {
	return extensionRepository{ID: packageformat.Hash([]byte(officialRepositoryURL + "\n")), URL: officialRepositoryURL, Official: true, Index: repositoryIndex{Schema: 1, Name: "Mosaune — extensions officielles", Packages: []repositoryEntry{}}}
}

type repositoryEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	MinCore     string `json:"minCore"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
}
type repositoryIndex struct {
	Schema   int               `json:"schema"`
	Name     string            `json:"name"`
	Packages []repositoryEntry `json:"packages"`
}
type extensionRepository struct {
	ID        string          `json:"id"`
	URL       string          `json:"url"`
	Ref       string          `json:"ref"`
	UpdatedAt int64           `json:"updatedAt"`
	Index     repositoryIndex `json:"index"`
	Bundled   bool            `json:"bundled"`
	Official  bool            `json:"official"`
}

func migrateExtensionRepositories(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 13 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS extension_repositories(id TEXT PRIMARY KEY,url TEXT NOT NULL,ref TEXT NOT NULL,index_json TEXT NOT NULL,updated_at INTEGER NOT NULL,UNIQUE(url,ref)); PRAGMA user_version=13;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

var repositorySegment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var repositoryHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func repositoryPath(s string) bool {
	if len(s) == 0 || len(s) > 512 {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "." || p == ".." || !repositorySegment.MatchString(p) {
			return false
		}
	}
	return true
}
func normalizeRepository(raw, ref string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return "", errors.New("Utilisez l’adresse HTTPS d’un dépôt public : https://github.com/auteur/depot.")
	}
	p := strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), ".git")
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	if len(parts) != 2 || !repositoryPath(strings.Join(parts, "/")) {
		return "", errors.New("Indiquez le dépôt GitHub, sans chemin de fichier.")
	}
	if len(ref) > 128 || (ref != "" && !repositoryPath(ref)) {
		return "", errors.New("Branche ou référence GitHub non valide.")
	}
	return "https://github.com/" + strings.ToLower(strings.Join(parts, "/")), nil
}

func parseRepositoryIndex(raw []byte) (repositoryIndex, error) {
	var index repositoryIndex
	if len(raw) > repositoryIndexLimit {
		return index, errors.New("Catalogue trop volumineux.")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&index); err != nil {
		return index, errors.New("Catalogue JSON non reconnu.")
	}
	var extra any
	if d.Decode(&extra) != io.EOF || index.Schema != 1 || strings.TrimSpace(index.Name) == "" || len(index.Name) > 100 || len(index.Packages) > 128 {
		return index, errors.New("Catalogue incompatible ou trop volumineux (128 paquets maximum).")
	}
	seen := map[string]bool{}
	for _, p := range index.Packages {
		if !extensionID.MatchString(p.ID) || len(p.ID) > 100 || !extensionVersion.MatchString(p.Version) || !extensionVersion.MatchString(p.MinCore) || (p.Kind != "pages" && p.Kind != "theme" && p.Kind != "provider" && p.Kind != "recognition" && p.Kind != "audio-tags" && p.Kind != "catalogue" && p.Kind != "detail-list" && p.Kind != "detail-sections" && p.Kind != "detail-header" && p.Kind != "detail-experience") || strings.TrimSpace(p.Name) == "" || len(p.Name) > 100 || strings.TrimSpace(p.Publisher) == "" || len(p.Publisher) > 100 || len(p.Description) > 1000 || !repositoryPath(p.Path) || !strings.HasSuffix(p.Path, ".zip") || !repositoryHash.MatchString(p.SHA256) {
			return index, errors.New("Une fiche du catalogue est incomplète ou non valide.")
		}
		key := p.ID + "@" + p.Version
		if seen[key] {
			return index, errors.New("Le catalogue contient une version en double.")
		}
		seen[key] = true
	}
	if index.Packages == nil {
		index.Packages = []repositoryEntry{}
	}
	return index, nil
}

// No arbitrary URL, credentials, cookies or redirects from repository input.
// GitHub's raw Contents media type supports files up to 100 MB; our bounds are
// considerably smaller. Omitting ref uses the repository's default branch.
func (c *catalog) fetchRepositoryFile(ctx context.Context, repo extensionRepository, path string, limit int64) ([]byte, error) {
	canonical, err := normalizeRepository(repo.URL, repo.Ref)
	if err != nil || !repositoryPath(path) {
		return nil, errors.New("Source du paquet non valide.")
	}
	u := "https://api.github.com/repos/" + strings.TrimPrefix(canonical, "https://github.com/") + "/contents/" + path
	if repo.Ref != "" {
		u += "?ref=" + url.QueryEscape(repo.Ref)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Mosaune/"+version)
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection interdite") }}
	if c.repositoryTransport != nil {
		client.Transport = c.repositoryTransport
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("GitHub ne répond pas. Le catalogue et les paquets installés restent conservés.")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 404:
			return nil, errors.New("Fichier absent ou dépôt privé. Vérifiez le dépôt, sa branche et mosaune-repository.json.")
		case 403, 429:
			return nil, errors.New("GitHub limite les requêtes. Réessayez plus tard ; le catalogue précédent est conservé.")
		}
		return nil, fmt.Errorf("Téléchargement GitHub indisponible (HTTP %d).", resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, errors.New("Fichier trop volumineux.")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("Fichier incomplet ou trop volumineux.")
	}
	return raw, nil
}

func bundledThemeArchive() []byte {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	f, _ := w.Create("extension.json")
	_, _ = f.Write(exampleThemeV2)
	_ = w.Close()
	return b.Bytes()
}

// A complete, self-contained public-repository example, with no local settings.
func bundledRepositoryArchive() ([]byte, error) {
	repo := bundledRepository()
	index, err := json.MarshalIndent(repo.Index, "", "  ")
	if err != nil {
		return nil, err
	}
	files := []struct {
		name string
		raw  []byte
	}{
		{repositoryIndexFile, index},
		{"README.md", []byte("# Exemple de dépôt Mosaune\n\nPaquets de démonstration : Ambre et TMDB. Aucun réglage, compte ou secret.\n\nDans un dépôt GitHub public, placez mosaune-repository.json et le dossier packages à la racine. Dans Mosaune 0.0.20 ou plus : Réglages → Extensions → À télécharger → Gérer les dépôts GitHub. Ajoutez l'adresse https://github.com/VOTRE-COMPTE/VOTRE-DEPOT. Laissez la référence vide pour la branche par défaut.\n\nL'empreinte sha256 de chaque fiche est celle du fichier ZIP complet. Ne modifiez pas un paquet sans mettre à jour sa fiche. TMDB reste signé par la clé de développement du projet, sans autoriser d'autres signatures. Les deux versions TMDB ont les mêmes fonctions : elles servent à tester les mises à jour et le retour arrière.\n")},
		{"packages/ambre.mosaune.zip", bundledThemeArchive()},
		{"packages/salon.mosaune.zip", uiExampleArchive()},
		{"packages/tmdb-0.4.0.mosaune.zip", providerExample},
		{"packages/tmdb-0.4.1.mosaune.zip", providerUpdateExample},
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, f := range files {
		w, e := z.Create(f.name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(f.raw); e != nil {
			return nil, e
		}
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func bundledRepository() extensionRepository {
	var theme extensionManifest
	_ = json.Unmarshal(exampleThemeV2, &theme)
	entries := []repositoryEntry{{ID: theme.ID, Name: theme.Name, Publisher: theme.Publisher, Description: theme.Description, Version: theme.Version, Kind: "theme", MinCore: theme.MinCore, Path: "packages/ambre.mosaune.zip", SHA256: packageformat.Hash(bundledThemeArchive())}}
	var pages extensionManifest
	_ = json.Unmarshal(exampleUIPages, &pages)
	entries = append(entries, repositoryEntry{ID: pages.ID, Name: pages.Name, Publisher: pages.Publisher, Description: pages.Description, Version: pages.Version, Kind: "pages", MinCore: pages.MinCore, Path: "packages/salon.mosaune.zip", SHA256: packageformat.Hash(uiExampleArchive())})
	for _, raw := range [][]byte{providerExample, providerUpdateExample} {
		p, err := packageformat.Parse(raw, trustedProviderKey(), version)
		if err != nil {
			continue
		}
		m := p.Manifest
		entries = append(entries, repositoryEntry{ID: m.Descriptor.ID, Name: m.Name, Publisher: m.Publisher, Description: m.Description, Version: m.Descriptor.Version, Kind: "provider", MinCore: m.MinCore, Path: "packages/tmdb-" + m.Descriptor.Version + ".mosaune.zip", SHA256: packageformat.Hash(raw)})
	}
	return extensionRepository{ID: bundledRepositoryID, Bundled: true, Index: repositoryIndex{Schema: 1, Name: "Inclus avec Mosaune", Packages: entries}}
}
func (c *catalog) repository(ctx context.Context, id string) (extensionRepository, error) {
	if id == bundledRepositoryID {
		return bundledRepository(), nil
	}
	var r extensionRepository
	var raw string
	err := c.db.QueryRowContext(ctx, `SELECT id,url,ref,index_json,updated_at FROM extension_repositories WHERE id=?`, id).Scan(&r.ID, &r.URL, &r.Ref, &raw, &r.UpdatedAt)
	if err == sql.ErrNoRows && id == officialRepository().ID {
		return officialRepository(), nil
	}
	if err != nil {
		return r, err
	}
	r.Index, err = parseRepositoryIndex([]byte(raw))
	r.Official = r.ID == officialRepository().ID && r.URL == officialRepositoryURL && r.Ref == ""
	return r, err
}

func (c *catalog) repositoryPackage(ctx context.Context, repo extensionRepository, id, release string) ([]byte, error) {
	var entry *repositoryEntry
	for i := range repo.Index.Packages {
		p := &repo.Index.Packages[i]
		if p.ID == id && p.Version == release {
			entry = p
			break
		}
	}
	if entry == nil {
		return nil, errors.New("Paquet absent du catalogue. Actualisez le dépôt.")
	}
	if compareExtensionVersions(entry.MinCore, version) > 0 {
		return nil, errors.New("Mettez Mosaune à jour avant d’installer ce paquet.")
	}
	var raw []byte
	var err error
	if repo.Bundled {
		switch entry.Path {
		case "packages/salon.mosaune.zip":
			raw = uiExampleArchive()
		case "packages/ambre.mosaune.zip":
			raw = bundledThemeArchive()
		case "packages/tmdb-0.4.0.mosaune.zip":
			raw = providerExample
		case "packages/tmdb-0.4.1.mosaune.zip":
			raw = providerUpdateExample
		}
	} else {
		limit := int64(extensionLimit)
		if entry.Kind == "provider" || (entry.Kind == "recognition" || entry.Kind == "audio-tags" || (entry.Kind == "catalogue" || (entry.Kind == "detail-list" || entry.Kind == "detail-sections" || entry.Kind == "detail-header" || entry.Kind == "detail-experience"))) {
			limit = packageformat.MaxArchive
		}
		raw, err = c.fetchRepositoryFile(ctx, repo, entry.Path, limit)
	}
	if err != nil {
		return nil, err
	}
	if packageformat.Hash(raw) != entry.SHA256 {
		return nil, errors.New("Le fichier ne correspond plus à l’empreinte du catalogue. Actualisez le dépôt et vérifiez de nouveau le paquet.")
	}
	var actual repositoryEntry
	if entry.Kind == "recognition" || entry.Kind == "audio-tags" || (entry.Kind == "catalogue" || (entry.Kind == "detail-list" || entry.Kind == "detail-sections" || entry.Kind == "detail-header" || entry.Kind == "detail-experience")) {
		p, err := packageformat.ParseExecutable(raw, version)
		if err != nil {
			return nil, err
		}
		kind, err := validateRecognitionImport(p)
		if err != nil {
			return nil, err
		}
		if (kind == "audiotags") != (entry.Kind == "audio-tags") || strings.HasPrefix(kind, "catalogue-") != (entry.Kind == "catalogue") || (kind == "detail-series") != (entry.Kind == "detail-list") || (kind == "detail-information") != (entry.Kind == "detail-sections") || (kind == "detail-header") != (entry.Kind == "detail-header") || (kind == "detail-experience") != (entry.Kind == "detail-experience") {
			return nil, errors.New("Le rôle du paquet ne correspond pas à sa fiche du dépôt.")
		}
		m := p.Manifest
		actual = repositoryEntry{ID: m.Module.ID, Version: m.Module.Version, Name: m.Name, Publisher: m.Publisher, MinCore: m.MinCore, Description: m.Description}
	} else if entry.Kind == "provider" {
		p, err := packageformat.Parse(raw, trustedProviderKey(), version)
		if err != nil {
			return nil, err
		}
		m := p.Manifest
		actual = repositoryEntry{ID: m.Descriptor.ID, Version: m.Descriptor.Version, Name: m.Name, Publisher: m.Publisher, MinCore: m.MinCore, Description: m.Description}
	} else {
		p, err := parseExtensionPackage(raw)
		if err != nil {
			return nil, err
		}
		m := p.Manifest
		if (m.API == 3) != (entry.Kind == "pages") {
			return nil, errors.New("Le rôle des pages ne correspond pas au catalogue.")
		}
		actual = repositoryEntry{ID: m.ID, Version: m.Version, Name: m.Name, Publisher: m.Publisher, MinCore: m.MinCore, Description: m.Description}
	}
	if actual.ID != entry.ID || actual.Version != entry.Version || actual.Name != entry.Name || actual.Publisher != entry.Publisher || actual.MinCore != entry.MinCore || actual.Description != entry.Description {
		return nil, errors.New("La fiche du catalogue ne correspond pas au manifeste du paquet.")
	}
	return raw, nil
}
