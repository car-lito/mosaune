package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
)

type databaseAccess interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func migrateMetadata(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > catalogSchemaVersion {
		return errors.New("Ce catalogue appartient à une version plus récente de Mosaune.")
	}
	if version >= 2 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS metadata (media_id TEXT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE, revision TEXT NOT NULL, data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS artwork (id TEXT PRIMARY KEY, data BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS provider_settings (id TEXT PRIMARY KEY, secret TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS provider_cache (provider TEXT NOT NULL, key TEXT NOT NULL, body BLOB NOT NULL, expires INTEGER NOT NULL, PRIMARY KEY(provider,key));
 PRAGMA user_version=2;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeArtwork(data []byte, maxSide int) ([]byte, error) {
	if len(data) == 0 || len(data) > 8<<20 {
		return nil, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return nil, nil
	}
	original, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil
	}
	w, h := cfg.Width, cfg.Height
	if w > maxSide || h > maxSide {
		if w >= h {
			h = h * maxSide / w
			w = maxSide
		} else {
			w = w * maxSide / h
			h = maxSide
		}
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	resized := image.NewRGBA(image.Rect(0, 0, w, h))
	bounds := original.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			resized.Set(x, y, original.At(bounds.Min.X+x*cfg.Width/w, bounds.Min.Y+y*cfg.Height/h))
		}
	}
	var output bytes.Buffer
	if err = jpeg.Encode(&output, resized, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func storeArtwork(ctx context.Context, db databaseAccess, data []byte) (string, error) {
	normalized, err := normalizeArtwork(data, 640)
	if err != nil || len(normalized) == 0 {
		return "", err
	}
	output := bytes.NewBuffer(normalized)
	sum := sha256.Sum256(output.Bytes())
	id := hex.EncodeToString(sum[:])
	// Cap the shared image cache, not just each individual image.
	_, err = db.ExecContext(ctx, `INSERT OR IGNORE INTO artwork(id,data) SELECT ?,? WHERE (SELECT COALESCE(SUM(length(data)),0) FROM artwork)+? <= 134217728`, id, output.Bytes(), output.Len())
	if err != nil {
		return "", err
	}
	var exists int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artwork WHERE id=?`, id).Scan(&exists); err != nil {
		return "", err
	}
	if exists == 0 {
		return "", nil
	}
	return id, nil
}

func indexLocalMetadata(ctx context.Context, tx *sql.Tx, root *os.Root, l library, rel string) error {
	var id, revision, oldRevision string
	err := tx.QueryRowContext(ctx, `SELECT m.id,m.revision,COALESCE(d.revision,'') FROM media m LEFT JOIN metadata d ON d.media_id=m.id WHERE m.library_id=? AND m.relative_path=?`, l.ID, rel).Scan(&id, &revision, &oldRevision)
	if err != nil {
		return err
	}
	if revision == oldRevision {
		return nil
	}
	var md Metadata
	var picture []byte
	if l.Kind == "music" {
		f, err := root.Open(filepath.FromSlash(rel))
		if err != nil {
			return err
		}
		md, picture = localMusic(f, rel)
		f.Close()
	} else {
		md = localVideo(l.Kind, rel)
	}
	md.ImageID, err = storeArtwork(ctx, tx, picture)
	if err != nil {
		return err
	}
	if md.ImageID != "" {
		md.Origins["image"] = "tags"
	}
	data, err := json.Marshal(md)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO metadata(media_id,revision,data) VALUES(?,?,?) ON CONFLICT(media_id) DO UPDATE SET revision=excluded.revision,data=excluded.data`, id, revision, string(data))
	return err
}

func (c *catalog) credential(ctx context.Context, id string) string {
	var secret string
	_ = c.db.QueryRowContext(ctx, `SELECT secret FROM provider_settings WHERE id=?`, id).Scan(&secret)
	return secret
}
func (c *catalog) providerInfo(ctx context.Context) []ProviderInfo {
	infos := []ProviderInfo{}
	for _, p := range c.providers {
		info := p.Info()
		info.Enabled = providerEnabled(p)
		_, info.Discovery = p.(DiscoveryProvider)
		info.Configured = !info.RequiresCredential || c.credential(ctx, info.ID) != ""
		infos = append(infos, info)
	}
	return infos
}
func (c *catalog) mediaMetadata(ctx context.Context, id string) (media, error) {
	var m media
	var data string
	err := c.db.QueryRowContext(ctx, `SELECT m.id,m.library_id,m.relative_path,m.revision,l.kind,COALESCE(d.data,'{}') FROM media m JOIN libraries l ON l.id=m.library_id LEFT JOIN metadata d ON d.media_id=m.id AND d.revision=m.revision WHERE m.id=? AND m.available=1`, id).Scan(&m.ID, &m.LibraryID, &m.Filename, &m.Revision, &m.Kind, &data)
	if err != nil {
		return m, err
	}
	_ = json.Unmarshal([]byte(data), &m.Metadata)
	if m.Metadata == nil || m.Metadata.Title == "" {
		md := localVideo(m.Kind, m.Filename)
		m.Metadata = &md
	}
	return m, nil
}
func (c *catalog) saveMetadata(ctx context.Context, m media, md Metadata) error {
	if err := md.Presentation.Validate(); err != nil {
		return err
	}
	unlock, e := providerCommit(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	data, err := json.Marshal(md)
	if err != nil {
		return err
	}
	result, err := c.db.ExecContext(ctx, `INSERT INTO metadata(media_id,revision,data) SELECT id,revision,? FROM media WHERE id=? AND revision=? AND available=1 ON CONFLICT(media_id) DO UPDATE SET revision=excluded.revision,data=excluded.data`, string(data), m.ID, m.Revision)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.New("Le fichier a changé. Actualisez la bibliothèque.")
	}
	return nil
}
func readBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("Réponse trop volumineuse.")
	}
	return data, nil
}
