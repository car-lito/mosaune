package main

import (
	"context"
	"database/sql"
	"errors"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
	"net/http"
)

const audioTagsID = "mosaune.music.tags"

func installAudioTags(ctx context.Context, tx *sql.Tx, revision int64) (registry.Snapshot, error) {
	d, code, err := modules.Lookup("audiotags")
	if err != nil {
		return registry.Snapshot{}, err
	}
	entry := registry.Entry{Descriptor: d, CodeSHA256: recognitionHash(code)}
	plan, err := (registry.SQLStore{DB: tx}).Commit(ctx, revision, registry.Change{Operation: "install", ID: d.ID, Entry: &entry})
	if err != nil {
		return registry.Snapshot{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO local_metadata_packages(id,code) VALUES(?,?)", d.ID, code)
	return plan.After, err
}
func migrateAudioTags(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 20 {
		return nil
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "CREATE TABLE local_metadata_packages(id TEXT PRIMARY KEY,code BLOB NOT NULL)"); err != nil {
		return err
	}
	store := registry.SQLStore{DB: tx}
	s, err := store.Load(ctx)
	if err != nil {
		return err
	}
	s, err = installAudioTags(ctx, tx, s.Revision)
	if err != nil {
		return err
	}
	if err = registryWanted(ctx, store, s, audioTagsID, true); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=20"); err != nil {
		return err
	}
	return tx.Commit()
}

type audioTagsState struct {
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
	SafeMode  bool   `json:"safeMode"`
	Revision  int64  `json:"revision"`
	Version   string `json:"version"`
}

func (c *catalog) audioTagsState(ctx context.Context) (audioTagsState, error) {
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	e, ok := registryEntryFor(s, audioTagsID)
	d, _ := modules.Descriptor("audiotags")
	return audioTagsState{Installed: ok, Enabled: ok && e.Enabled, SafeMode: c.extensionSafeMode, Revision: s.Revision, Version: d.Version}, err
}

// Compatibility endpoint for 0.0.33 clients; all changes use the shared lifecycle.
func (c *catalog) changeAudioTags(ctx context.Context, op string, revision int64) error {
	return c.changeRecognition(ctx, audioTagsID, op, revision)
}
func (c *catalog) openAudioTags(ctx context.Context) (context.Context, *audioTagReader, func(), error) {
	call, done, descriptor, code, err := c.beginRecognition(ctx, "audiotags")
	if errors.Is(err, errRecognitionUnavailable) {
		return ctx, nil, func() {}, nil
	}
	if err != nil {
		return call, nil, done, err
	}
	reader, err := loadAudioTags(call, descriptor, code)
	if err != nil {
		done()
		return call, nil, func() {}, err
	}
	return call, reader, func() { reader.engine.Close(); done() }, nil
}

func (c *catalog) audioTagsRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/audio-tags", func(w http.ResponseWriter, r *http.Request) {
		s, err := c.audioTagsState(r.Context())
		if err != nil {
			problem(w, 500, "Extension indisponible.")
			return
		}
		sendJSON(w, 200, s)
	})
	mux.HandleFunc("POST /api/v1/audio-tags", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Operation string `json:"operation"`
			Revision  *int64 `json:"revision"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if in.Revision == nil {
			problem(w, 400, "Actualisez la liste.")
			return
		}
		if err := c.changeAudioTags(r.Context(), in.Operation, *in.Revision); err != nil {
			problem(w, 409, "Modification non appliquée. Actualisez la liste. "+err.Error())
			return
		}
		sendJSON(w, 200, map[string]bool{"ok": true})
	})
}
