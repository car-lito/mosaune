package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	"os"
	"path/filepath"
	"testing"
)

func audioTagsExample(t *testing.T, v string) ([]byte, packageformat.Executable) {
	t.Helper()
	raw, err := os.ReadFile("extension-kit/examples/audiotags/extension.json")
	if err != nil {
		t.Fatal(err)
	}
	var m packageformat.ExecutableManifest
	if err = protocol.DecodeOne(raw, &m); err != nil {
		t.Fatal(err)
	}
	m.Module.Version = v
	code, err := os.ReadFile("extension-kit/examples/audiotags/module.wasm")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = packageformat.CreateExecutable(m, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	return raw, pack
}
func testID3v1() []byte {
	b := make([]byte, 128)
	copy(b, "TAG")
	copy(b[3:33], []byte{'C', 'a', 'f', 0xe9})
	copy(b[33:63], "Artiste externe")
	copy(b[63:93], "Album externe")
	return append(make([]byte, 200), b...)
}
func TestAudioTagsExternalScanChoiceAndRestore(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	raw, p := audioTagsExample(t, "1.0.0")
	id := p.Manifest.Module.ID
	old, _, done, err := c.openAudioTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	installRecognitionExample(t, c, raw, p)
	recognitionChange(t, c, id, "enable")
	if old.Err() != nil {
		t.Fatal("unselected import cancelled official reader")
	}
	call, reader, finish, err := c.openAudioTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	data := testMP3()
	md, _, err := reader.extract(call, bytes.NewReader(data), int64(len(data)), Metadata{})
	finish()
	if err != nil || md.Title != "Titre de test" {
		t.Fatal("implicit selection", md, err)
	}
	recognitionChange(t, c, id, "select")
	if old.Err() == nil {
		t.Fatal("selection did not revoke previous reader")
	}
	files := t.TempDir()
	if err = os.WriteFile(filepath.Join(files, "external.mp3"), testID3v1(), 0600); err != nil {
		t.Fatal(err)
	}
	l, err := c.scan(ctx, library{Path: files, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	items := getItems(t, c)
	if len(items) != 1 || items[0].Metadata.Title != "Café" || items[0].Metadata.Artist != "Artiste externe" {
		t.Fatal("external reader not used", items)
	}
	active, _, finish, err := c.openAudioTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	newer, p2 := audioTagsExample(t, "1.1.0")
	_, rev, _ := c.recognitionPackages(ctx)
	if err = c.importRecognition(ctx, newer, p2.Checksum, rev-1); !errors.Is(err, registry.ErrConflict) || active.Err() != nil {
		t.Fatal("stale update affected session", err)
	}
	installRecognitionExample(t, c, newer, p2)
	if active.Err() == nil {
		t.Fatal("updated reader still running")
	}
	finish()
	recognitionChange(t, c, id, "restore")
	list, _, _ := c.recognitionPackages(ctx)
	for _, v := range list {
		if v.ID == id && (v.Version != "1.0.0" || !v.Enabled || !v.Selected) {
			t.Fatal(v)
		}
	}
	done()
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	list, _, _ = next.recognitionPackages(ctx)
	for _, v := range list {
		if v.ID == id && !v.Selected {
			t.Fatal("choice lost")
		}
	}
	recognitionChange(t, next, id, "disable")
	_, r, closeReader, err := next.openAudioTags(ctx)
	closeReader()
	if err != nil || r != nil {
		t.Fatal("disabled reader fell back", err)
	}
	recognitionChange(t, next, id, "remove")
	if err = os.WriteFile(filepath.Join(files, "new.mp3"), testMP3(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = next.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	items = getItems(t, next)
	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Metadata.Title] = true
	}
	if len(items) != 2 || !titles["Café"] || !titles["new"] {
		t.Fatal("removed choice substituted or cached metadata lost", items)
	}
	// Selecting the official package is the only way back; its enabled state alone is insufficient.
	recognitionChange(t, next, audioTagsID, "select")
	call, reader, finish, err = next.openAudioTags(ctx)
	if err != nil || reader == nil {
		t.Fatal(err)
	}
	defer finish()
	md, _, err = reader.extract(call, bytes.NewReader(data), int64(len(data)), Metadata{})
	if err != nil || md.Title != "Titre de test" {
		t.Fatal(md, err)
	}
}
func TestAudioTagsImportLeastPrivilege(t *testing.T) {
	_, p := audioTagsExample(t, "1.0.0")
	cases := []func(*packageformat.ExecutableManifest){
		func(m *packageformat.ExecutableManifest) { m.Module.ID = "mosaune.music.tags" },
		func(m *packageformat.ExecutableManifest) { m.Module.Services = nil },
		func(m *packageformat.ExecutableManifest) {
			m.Module.Services[0].Operations = append(m.Module.Services[0].Operations, "write")
		},
		func(m *packageformat.ExecutableManifest) {
			m.Module.Services = append(m.Module.Services, protocol.ServiceRequirement{Name: "network.http.v1", Operations: []string{"get"}})
		},
		func(m *packageformat.ExecutableManifest) { m.Contributions[0].Target = "films" },
	}
	for i, change := range cases {
		raw, _ := json.Marshal(p.Manifest)
		var m packageformat.ExecutableManifest
		json.Unmarshal(raw, &m)
		change(&m)
		q := p
		q.Manifest = m
		if _, err := validateRecognitionImport(q); err == nil {
			t.Fatalf("policy accepted case %d", i)
		}
	}
	c, _ := testCatalog(t)
	installRecognitionExample(t, c, mustAudioArchive(t, p), p)
	// A recognizer update cannot become a file-reading role under the same identity.
	raw, q := recognitionExample(t, "1.1.0")
	_ = raw
	q.Manifest.Module.ID = p.Manifest.Module.ID
	if _, err := c.previewRecognition(context.Background(), q); err == nil {
		t.Fatal("role-changing update accepted")
	}
}
func mustAudioArchive(t *testing.T, p packageformat.Executable) []byte {
	t.Helper()
	b, e := packageformat.CreateExecutable(p.Manifest, p.Code)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestAudioTagsMigration20PreservesState(t *testing.T) {
	for _, op := range []string{"enabled", "disable", "remove"} {
		t.Run(op, func(t *testing.T) {
			c, dir := testCatalog(t)
			if op != "enabled" {
				audioTagsChange(t, c, op)
			}
			if _, err := c.db.Exec(`CREATE TABLE local_metadata_packages(id TEXT PRIMARY KEY,code BLOB NOT NULL);
 INSERT INTO local_metadata_packages SELECT id,code FROM recognition_packages WHERE kind='audiotags';
 DELETE FROM recognition_packages WHERE kind='audiotags'; DELETE FROM recognition_selection WHERE kind='audiotags'; PRAGMA user_version=20;`); err != nil {
				t.Fatal(err)
			}
			c.close()
			next, err := openCatalog(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer next.close()
			state, err := next.audioTagsState(context.Background())
			if err != nil || state.Installed != (op != "remove") || state.Enabled != (op == "enabled") {
				t.Fatal(state, err)
			}
			var id string
			if err = next.db.QueryRow("SELECT id FROM recognition_selection WHERE kind='audiotags'").Scan(&id); err != nil || id != audioTagsID {
				t.Fatal(id, err)
			}
			backups, _ := filepath.Glob(filepath.Join(dir, "mosaune-before-extensions-*.db"))
			if len(backups) != 1 {
				t.Fatal("missing migration backup")
			}
		})
	}
}
