package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type discoveredMedia struct {
	rel           string
	info          fs.FileInfo
	metadata      Metadata
	pictureOffset int64
	pictureLength int
}
type artworkSpool struct {
	dir  string
	file *os.File
	size int64
}

func (s *artworkSpool) close() {
	if s.file != nil {
		name := s.file.Name()
		s.file.Close()
		os.Remove(name)
	}
}
func (s *artworkSpool) add(data []byte) (int64, int, error) {
	var err error
	data, err = normalizeArtwork(data, 640)
	if err != nil {
		return 0, 0, nil
	}
	if len(data) == 0 || s.size+int64(len(data)) > 128<<20 {
		return 0, 0, nil
	}
	if s.file == nil {
		f, err := os.CreateTemp(s.dir, "mosaune-artwork-*.tmp")
		if err != nil {
			return 0, 0, err
		}
		s.file = f
	}
	offset := s.size
	n, err := s.file.Write(data)
	s.size += int64(n)
	return offset, n, err
}
func (s *artworkSpool) read(offset int64, n int) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	b := make([]byte, n)
	_, err := s.file.ReadAt(b, offset)
	return b, err
}
func (c *catalog) stageAudioTags(ctx context.Context, reader *audioTagReader, root *os.Root, l library, files []discoveredMedia, spool *artworkSpool) error {
	if reader == nil {
		return nil
	}
	for i := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		file := &files[i]
		var unchanged bool
		err := c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media m JOIN metadata d ON d.media_id=m.id AND d.revision=m.revision WHERE m.library_id=? AND m.relative_path=? AND m.size=? AND m.modified=?)`, l.ID, file.rel, file.info.Size(), file.info.ModTime().UnixNano()).Scan(&unchanged)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if unchanged {
			c.setScanProgress(scanProgress{LibraryID: l.ID, Name: l.Name, Kind: l.Kind, State: "tags", Found: len(files), Total: len(files), Processed: i + 1})
			continue
		}
		f, err := root.Open(filepath.FromSlash(file.rel))
		if err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		if !info.Mode().IsRegular() || info.Size() != file.info.Size() || !info.ModTime().Equal(file.info.ModTime()) {
			f.Close()
			return errors.New("Fichier modifié pendant l’analyse. Relancez le scan.")
		}
		md, picture, err := reader.extract(ctx, f, info.Size(), file.metadata)
		f.Close()
		if err != nil {
			return err
		}
		file.metadata = md
		file.pictureOffset, file.pictureLength, err = spool.add(picture)
		if err != nil {
			return err
		}
		c.setScanProgress(scanProgress{LibraryID: l.ID, Name: l.Name, Kind: l.Kind, State: "tags", Found: len(files), Total: len(files), Processed: i + 1})
	}
	return nil
}
