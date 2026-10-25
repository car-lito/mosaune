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
func (c *catalog) changeAudioTags(ctx context.Context, op string, revision int64) error {
	if op != "install" && op != "enable" && op != "disable" && op != "remove" {
		return errors.New("Action inconnue.")
	}
	if c.extensionSafeMode && (op == "install" || op == "enable") {
		return errors.New("Quittez le mode de secours pour activer cette extension.")
	}
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if op == "install" {
		_, err = installAudioTags(ctx, tx, revision)
	} else {
		_, err = (registry.SQLStore{DB: tx}).Commit(ctx, revision, registry.Change{Operation: op, ID: audioTagsID})
	}
	if err != nil {
		return err
	}
	if op == "remove" {
		if _, err = tx.ExecContext(ctx, "DELETE FROM local_metadata_packages WHERE id=?", audioTagsID); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if op == "disable" || op == "remove" {
		c.cancelRecognitionID(audioTagsID)
	}
	return nil
}
func (c *catalog) openAudioTags(ctx context.Context) (context.Context, *audioTagReader, func(), error) {
	c.recognition.mu.Lock()
	finish := func() { c.recognition.mu.Unlock() }
	s, err := (registry.SQLStore{DB: c.db}).Load(ctx)
	if err != nil {
		finish()
		return ctx, nil, func() {}, err
	}
	entry, ok := registryEntryFor(s, audioTagsID)
	if c.extensionSafeMode || !ok || !entry.Enabled {
		finish()
		return ctx, nil, func() {}, nil
	}
	var code []byte
	err = c.db.QueryRowContext(ctx, "SELECT code FROM local_metadata_packages WHERE id=?", audioTagsID).Scan(&code)
	if err != nil || recognitionHash(code) != entry.CodeSHA256 {
		finish()
		return ctx, nil, func() {}, errors.New("Extension musicale absente ou endommagée. Réinstallez-la ou désactivez-la ; votre catalogue est conservé.")
	}
	call, cancel := context.WithCancelCause(ctx)
	c.recognition.next++
	key := c.recognition.next
	if c.recognition.calls == nil {
		c.recognition.calls = map[uint64]recognitionCall{}
	}
	c.recognition.calls[key] = recognitionCall{kind: "music", id: audioTagsID, cancel: cancel}
	finish()
	done := func() {
		c.recognition.mu.Lock()
		delete(c.recognition.calls, key)
		c.recognition.mu.Unlock()
		cancel(context.Canceled)
	}
	reader, err := loadAudioTags(call, entry.Descriptor, code)
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
