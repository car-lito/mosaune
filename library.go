package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type library struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Path            string `json:"path"`
	Kind            string `json:"kind"`
	Count           int    `json:"count"`
	MetadataMessage string `json:"metadataMessage,omitempty"`
}
type media struct {
	PosterURL  string    `json:"posterUrl,omitempty"`
	ID         string    `json:"id"`
	AddedAt    int64     `json:"addedAt"`
	LastPlayed int64     `json:"lastPlayed"`
	WorkID     string    `json:"workId,omitempty"`
	LibraryID  string    `json:"libraryId"`
	Name       string    `json:"name"`
	Filename   string    `json:"filename"`
	Kind       string    `json:"kind"`
	URL        string    `json:"url"`
	Size       int64     `json:"size"`
	Position   float64   `json:"position"`
	Duration   float64   `json:"duration"`
	Completed  bool      `json:"completed"`
	Revision   string    `json:"revision"`
	Metadata   *Metadata `json:"metadata,omitempty"`
}
type catalog struct {
	repositoryMu        sync.Mutex
	repositoryTransport http.RoundTripper // nil in production; deterministic HTTP fixtures in tests
	extensionMu         sync.Mutex
	extensionSafeMode   bool
	db                  *sql.DB
	auth                authentication
	authRead            *sql.DB
	scanMu              sync.Mutex
	scanStatusMu        sync.Mutex
	scanStatus          scanProgress
	metadataMu          sync.Mutex
	metadataActivityMu  sync.Mutex
	metadataCurrent     string
	discoveryMu         sync.Mutex
	posterMu            sync.Mutex
	customPosterMu      sync.Mutex
	requestMu           sync.Mutex
	requestScans        map[string]time.Time
	providers           []MetadataProvider
}

func openCatalog(dir string) (*catalog, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	// An ordinary filename avoids interpreting user-selected directory characters as URI options.
	db, err := sql.Open("sqlite", filepath.Join(dir, "mosaune.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;
 PRAGMA cache_size=-4096;
 CREATE TABLE IF NOT EXISTS libraries (id TEXT PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL UNIQUE, kind TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS media (
 id TEXT PRIMARY KEY, library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 relative_path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL, modified INTEGER NOT NULL,
 available INTEGER NOT NULL DEFAULT 1, position REAL NOT NULL DEFAULT 0, duration REAL NOT NULL DEFAULT 0,
 completed INTEGER NOT NULL DEFAULT 0, revision TEXT NOT NULL,
 UNIQUE(library_id, relative_path));
 `)
	if err != nil {
		db.Close()
		return nil, err
	}
	var oldVersion int
	if e := db.QueryRow("PRAGMA user_version").Scan(&oldVersion); e != nil {
		db.Close()
		return nil, e
	}
	if oldVersion > catalogSchemaVersion {
		db.Close()
		return nil, errors.New("Catalogue créé par une version plus récente de Mosaune.")
	}
	if oldVersion > 0 && oldVersion < catalogSchemaVersion {
		backup := filepath.Join(dir, "mosaune-before-extensions-"+time.Now().Format("20060102-150405.000000000")+".db")
		if _, e := db.Exec("VACUUM INTO ?", backup); e != nil {
			db.Close()
			return nil, fmt.Errorf("sauvegarde avant migration : %w", e)
		}
	}
	if err = migrateMetadata(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateMusic(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateDiscovery(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateExperience(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateRequests(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateMetadataQueue(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migratePosters(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateAccounts(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateExtensions(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateProviderRuntime(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateProviderPackages(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateExtensionRepositories(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateProviderStates(db); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateExtensionRegistry(db); err != nil {
		db.Close()
		return nil, err
	}
	reader, e := sql.Open("sqlite", filepath.Join(dir, "mosaune.db"))
	if e != nil {
		db.Close()
		return nil, e
	}
	reader.SetMaxOpenConns(1)
	if _, e = reader.Exec("PRAGMA query_only=ON; PRAGMA busy_timeout=5000; PRAGMA cache_size=-1024;"); e != nil {
		reader.Close()
		db.Close()
		return nil, e
	}
	c := &catalog{db: db, authRead: reader}
	if c.accountCount() == 0 {
		c.auth.code = rand.Text()
	}
	c.providers = []MetadataProvider{newTMDBProvider(db), newMusicBrainzProvider(db)}
	p := c.providers[0].(*tmdbProvider)
	var active = true
	if e := db.QueryRow("SELECT enabled FROM provider_states WHERE id='tmdb'").Scan(&active); e != nil && e != sql.ErrNoRows {
		reader.Close()
		db.Close()
		return nil, e
	}
	p.state.disabled = !active
	p.extension.blocked = func() bool { return c.extensionSafeMode }
	_ = db.QueryRow(`SELECT enabled FROM provider_runtime WHERE id='tmdb'`).Scan(&p.extension.enabled)
	return c, nil
}

func supported(kind, name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if kind == "music" {
		return strings.Contains("|.mp3|.flac|.m4a|.aac|.ogg|.opus|.wav|.aiff|.wma|", "|"+ext+"|") && ext != ""
	}
	return strings.Contains("|.mp4|.mkv|.webm|.mov|.avi|.m4v|.ts|.mpeg|.mpg|", "|"+ext+"|") && ext != ""
}

func (c *catalog) libraries(ctx context.Context) ([]library, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT l.id,l.name,l.path,l.kind,COUNT(m.id) FROM libraries l LEFT JOIN media m ON m.library_id=l.id AND m.available=1 GROUP BY l.id ORDER BY l.rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []library{}
	for rows.Next() {
		var l library
		if err = rows.Scan(&l.ID, &l.Name, &l.Path, &l.Kind, &l.Count); err != nil {
			return nil, err
		}
		if canLibrary(ctx, l.ID, l.Kind, "view") {
			if u := viewer(ctx); u != nil && !u.Admin {
				l.Path = ""
			}
			result = append(result, l)
		}
	}
	return result, rows.Err()
}

func (c *catalog) items(ctx context.Context) ([]media, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT m.id,m.library_id,m.name,m.relative_path,l.kind,m.size,m.position,m.duration,m.completed,m.revision,COALESCE(d.data,'{}'),m.added_at,m.last_played,COALESCE(a.work_id,'') FROM media m LEFT JOIN album_links a ON a.media_id=m.id AND a.revision=m.revision JOIN libraries l ON l.id=m.library_id LEFT JOIN metadata d ON d.media_id=m.id AND d.revision=m.revision WHERE m.available=1 ORDER BY m.name COLLATE NOCASE,m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []media{}
	for rows.Next() {
		var m media
		var metadataJSON string
		if err = rows.Scan(&m.ID, &m.LibraryID, &m.Name, &m.Filename, &m.Kind, &m.Size, &m.Position, &m.Duration, &m.Completed, &m.Revision, &metadataJSON, &m.AddedAt, &m.LastPlayed, &m.WorkID); err != nil {
			return nil, err
		}
		m.URL = "/api/v1/media/" + m.ID + "/stream"
		_ = json.Unmarshal([]byte(metadataJSON), &m.Metadata)
		if m.Metadata != nil && m.Metadata.Title != "" {
			m.Name = m.Metadata.Title
			if m.Metadata.Series != "" {
				m.Name = fmt.Sprintf("%s · S%02dE%02d · %s", m.Metadata.Series, m.Metadata.Season, m.Metadata.Episode, m.Metadata.Title)
			}
		}
		if canLibrary(ctx, m.LibraryID, m.Kind, "view") {
			result = append(result, m)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if userID(ctx) != "" {
		if err = c.personalProgress(ctx, result); err != nil {
			return nil, err
		}
	}
	choices := map[string]string{}
	images, e := c.db.QueryContext(ctx, "SELECT target,image_id FROM poster_choices")
	if e != nil {
		return nil, e
	}
	for images.Next() {
		var key, id string
		if e = images.Scan(&key, &id); e != nil {
			break
		}
		choices[key] = id
	}
	rowErr := images.Err()
	images.Close()
	if e != nil {
		return nil, e
	}
	if rowErr != nil {
		return nil, rowErr
	}
	for i := range result {
		m := &result[i]
		key := "media:" + m.ID
		md := m.Metadata
		if md == nil {
			continue
		}
		if choices[key] == "" {
			key = "work:" + workID(m.Kind, md.Provider, md.ProviderID)
		}
		if image := choices[key]; image != "" {
			m.PosterURL = "/api/v1/posters/media/" + m.ID + "?v=" + image
		}
	}
	return result, nil
}

// Scans are transactional: cancellation, an unreadable directory, or a disconnected disk
// leaves the previous catalog intact. Only metadata is written; media are never opened for writing.
func (c *catalog) scan(ctx context.Context, l library, add bool) (result library, scanErr error) {
	if !c.scanMu.TryLock() {
		return l, errors.New("Une analyse est déjà en cours. Réessayez après sa fin.")
	}
	defer c.scanMu.Unlock()
	c.setScanProgress(scanProgress{Name: l.Name, Kind: l.Kind, State: "discovering"})
	defer func() {
		c.scanStatusMu.Lock()
		defer c.scanStatusMu.Unlock()
		if scanErr != nil {
			c.scanStatus.State = "failed"
			c.scanStatus.Message = scanErr.Error()
		} else {
			c.scanStatus.State = "complete"
		}
		c.scanStatus.Updated = time.Now().Unix()
	}()
	if add {
		p := strings.Trim(strings.TrimSpace(l.Path), "\"")
		if !filepath.IsAbs(p) {
			return l, errors.New("Indiquez le chemin complet du dossier.")
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return l, errors.New("Dossier introuvable ou inaccessible.")
		}
		l.Path = filepath.Clean(resolved)
		if l.Name == "" {
			l.Name = filepath.Base(l.Path)
		}
		if len(l.Name) > 150 {
			return l, errors.New("Le nom du dossier est trop long.")
		}
		libs, err := c.libraries(ctx)
		if err != nil {
			return l, err
		}
		for _, other := range libs {
			if pathsOverlap(l.Path, other.Path) {
				return l, errors.New("Ce dossier ou un de ses parents est déjà dans la bibliothèque.")
			}
		}
		l.ID = rand.Text()
	}
	root, err := os.OpenRoot(l.Path)
	if err != nil {
		return l, errors.New("Dossier inaccessible. Le catalogue précédent est conservé.")
	}
	defer root.Close()
	// Discover before opening the transaction: the total is honest, and remote
	// directory traversal does not keep the catalog locked. No media bytes are kept.
	type discovered struct {
		rel  string
		info fs.FileInfo
	}
	files := []discovered{}
	err = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 || d.IsDir() || !supported(l.Kind, d.Name()) {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, discovered{rel, info})
		c.setScanProgress(scanProgress{LibraryID: l.ID, Name: l.Name, Kind: l.Kind, State: "discovering", Found: len(files)})
		return nil
	})
	if err != nil {
		return l, fmt.Errorf("Analyse interrompue ; catalogue précédent conservé : %w", err)
	}
	c.setScanProgress(scanProgress{LibraryID: l.ID, Name: l.Name, Kind: l.Kind, State: "indexing", Found: len(files), Total: len(files)})
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return l, err
	}
	defer tx.Rollback()
	if add {
		_, err = tx.ExecContext(ctx, `INSERT INTO libraries(id,name,path,kind) VALUES(?,?,?,?)`, l.ID, l.Name, l.Path, l.Kind)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE media SET available=0 WHERE library_id=?`, l.ID)
	}
	if err != nil {
		return l, err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO media(id,library_id,relative_path,name,size,modified,revision,added_at) VALUES(?,?,?,?,?,?,?,?)
 ON CONFLICT(library_id,relative_path) DO UPDATE SET available=1,name=excluded.name,
 position=CASE WHEN media.size=excluded.size AND media.modified=excluded.modified THEN media.position ELSE 0 END,
 duration=CASE WHEN media.size=excluded.size AND media.modified=excluded.modified THEN media.duration ELSE 0 END,
 completed=CASE WHEN media.size=excluded.size AND media.modified=excluded.modified THEN media.completed ELSE 0 END,
 revision=CASE WHEN media.size=excluded.size AND media.modified=excluded.modified THEN media.revision ELSE excluded.revision END,
 size=excluded.size,modified=excluded.modified`)
	if err != nil {
		return l, err
	}
	defer stmt.Close()
	l.Count = 0
	for _, file := range files {
		if err = ctx.Err(); err != nil {
			break
		}
		rel, info := file.rel, file.info
		_, err = stmt.ExecContext(ctx, rand.Text(), l.ID, rel, strings.TrimSuffix(info.Name(), path.Ext(info.Name())), info.Size(), info.ModTime().UnixNano(), rand.Text(), time.Now().Unix())
		if err == nil {
			err = indexLocalMetadata(ctx, tx, root, l, rel)
		}
		if err == nil {
			l.Count++
		}
		if err != nil {
			break
		}
		c.setScanProgress(scanProgress{LibraryID: l.ID, Name: l.Name, Kind: l.Kind, State: "indexing", Found: len(files), Total: len(files), Processed: l.Count})
	}
	if err != nil {
		return l, fmt.Errorf("Analyse interrompue ; catalogue précédent conservé : %w", err)
	}
	return l, tx.Commit()
}

func pathsOverlap(a, b string) bool {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && filepath.IsLocal(rel) {
			return true
		}
	}
	return false
}

// Root.Open prevents a file replaced by an escaping symlink from exposing other folders.
func (c *catalog) openMedia(ctx context.Context, id string) (*os.File, string, error) {
	var dir, rel, kind string
	err := c.db.QueryRowContext(ctx, `SELECT l.path,m.relative_path,l.kind FROM media m JOIN libraries l ON l.id=m.library_id WHERE m.id=? AND m.available=1`, id).Scan(&dir, &rel, &kind)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", err
	}
	defer root.Close()
	f, err := root.Open(filepath.FromSlash(rel))
	if err != nil {
		return nil, "", err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, "", errors.New("fichier indisponible")
	}
	return f, kind, nil
}

func (c *catalog) close() error {
	for _, provider := range c.providers {
		if p, ok := provider.(*tmdbProvider); ok {
			p.closeExtension()
		}
	}
	if c.authRead != nil {
		_ = c.authRead.Close()
	}
	return c.db.Close()
}
