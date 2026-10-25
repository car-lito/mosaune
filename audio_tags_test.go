package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	tags "mosaune/extensionkit/capabilities/audiotags/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
	"os"
	"path/filepath"
	"testing"
)

func testFLAC() []byte {
	var b bytes.Buffer
	b.WriteString("fLaC")
	block := func(kind byte, data []byte) {
		b.Write([]byte{kind, byte(len(data) >> 16), byte(len(data) >> 8), byte(len(data))})
		b.Write(data)
	}
	block(0, make([]byte, 34))
	var comments bytes.Buffer
	binary.Write(&comments, binary.LittleEndian, uint32(0))
	binary.Write(&comments, binary.LittleEndian, uint32(4))
	for _, s := range []string{"TITLE=Titre FLAC", "ARTIST=Artiste FLAC", "ALBUM=Album FLAC", "TRACKNUMBER=7"} {
		binary.Write(&comments, binary.LittleEndian, uint32(len(s)))
		comments.WriteString(s)
	}
	block(4, comments.Bytes())
	var pic bytes.Buffer
	binary.Write(&pic, binary.BigEndian, uint32(3))
	for _, s := range []string{"image/png", ""} {
		binary.Write(&pic, binary.BigEndian, uint32(len(s)))
		pic.WriteString(s)
	}
	for _, n := range []uint32{4, 4, 24, 0, uint32(len(testPicture()))} {
		binary.Write(&pic, binary.BigEndian, n)
	}
	pic.Write(testPicture())
	block(0x86, pic.Bytes())
	return b.Bytes()
}
func TestAudioTagsModuleMP3FLACAndInvalid(t *testing.T) {
	ctx := context.Background()
	d, code, err := modules.Lookup("audiotags")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := loadAudioTags(ctx, d, code)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.engine.Close()
	for _, tc := range []struct {
		name  string
		data  []byte
		title string
		track int
		art   bool
	}{{"MP3", testMP3(), "Titre de test", 2, true}, {"FLAC", testFLAC(), "Titre FLAC", 7, true}, {"broken", []byte("bad"), "filename", 0, false}, {"empty", nil, "filename", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			md, pic, e := reader.extract(ctx, bytes.NewReader(tc.data), int64(len(tc.data)), Metadata{Title: "filename"})
			if e != nil || md.Title != tc.title || md.Track != tc.track || (len(pic) > 0) != tc.art {
				t.Fatalf("%+v picture=%d error=%v", md, len(pic), e)
			}
		})
	}
	// A picture larger than four blocks exercises the bounded service budget.
	var frames bytes.Buffer
	picture := append([]byte{0}, []byte("image/png\x00\x03\x00")...)
	picture = append(picture, bytes.Repeat([]byte{42}, 400<<10)...)
	frames.WriteString("APIC")
	binary.Write(&frames, binary.BigEndian, uint32(len(picture)))
	frames.Write([]byte{0, 0})
	frames.Write(picture)
	size := frames.Len()
	large := append([]byte{'I', 'D', '3', 3, 0, 0, byte(size>>21) & 127, byte(size>>14) & 127, byte(size>>7) & 127, byte(size) & 127}, frames.Bytes()...)
	if _, pic, e := reader.extract(ctx, bytes.NewReader(large), int64(len(large)), Metadata{}); e != nil || len(pic) != (400<<10) {
		t.Fatal("multi-range picture", len(pic), e)
	}
	// The audio payload is not read.
	data := testMP3()
	data = append(data, make([]byte, 8<<20)...)
	counter := &countingAudio{Reader: bytes.NewReader(data)}
	if _, _, err = reader.extract(ctx, counter, int64(len(data)), Metadata{}); err != nil {
		t.Fatal(err)
	}
	if counter.read > tags.BlockSize {
		t.Fatal("audio content read unnecessarily", counter.read)
	}
}

type countingAudio struct {
	*bytes.Reader
	read int
}

func (r *countingAudio) ReadAt(b []byte, offset int64) (int, error) {
	n, e := r.Reader.ReadAt(b, offset)
	r.read += n
	return n, e
}
func TestAudioTagsScopedRangesAndLimits(t *testing.T) {
	ctx := context.Background()
	data := bytes.Repeat([]byte{42}, tags.BlockSize)
	reader := &countingAudio{Reader: bytes.NewReader(data)}
	service := audioFileService(reader, int64(len(data)))
	for _, raw := range []string{`{"offset":-1,"length":1}`, `{"offset":0,"length":65537}`, `{"offset":65536,"length":1}`, `{"offset":0,"length":1,"path":"other.mp3"}`, `{"offset":9223372036854775807,"length":1}`} {
		if _, e := service(ctx, "range", json.RawMessage(raw)); e == nil {
			t.Fatal("invalid range accepted", raw)
		}
	}
	if reader.read != 0 {
		t.Fatal("denied range read file")
	}
	q := json.RawMessage(`{"offset":0,"length":65536}`)
	for i := 0; i < tags.MaxRead/tags.BlockSize; i++ {
		if _, e := service(ctx, "range", q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := service(ctx, "range", q); e == nil || reader.read != tags.MaxRead {
		t.Fatal("byte budget not enforced", e, reader.read)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := audioFileService(reader, 65536)(cancelled, "range", q); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e := audioFileService(bytes.NewReader(nil), 10)(ctx, "range", json.RawMessage(`{"offset":0,"length":10}`)); !errors.Is(e, io.ErrUnexpectedEOF) {
		t.Fatal("truncated file accepted", e)
	}
}
func audioTagsChange(t *testing.T, c *catalog, op string) {
	t.Helper()
	s, e := c.audioTagsState(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if e = c.changeAudioTags(context.Background(), op, s.Revision); e != nil {
		t.Fatal(e)
	}
}
func TestAudioTagsLifecycleScanAndCancellation(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := testContext(context.Background())
	music := t.TempDir()
	os.WriteFile(filepath.Join(music, "one.mp3"), testMP3(), 0600)
	l, err := c.scan(ctx, library{Path: music, Kind: "music"}, true)
	if err != nil {
		t.Fatal(err)
	}
	original := getItems(t, c)[0]
	if original.Metadata.Title != "Titre de test" || original.Metadata.ImageID == "" {
		t.Fatal(original.Metadata)
	}
	call, reader, finish, err := c.openAudioTags(ctx)
	if err != nil || reader == nil {
		t.Fatal(err)
	}
	state, _ := c.audioTagsState(ctx)
	if err = c.changeAudioTags(ctx, "disable", state.Revision-1); !errors.Is(err, registry.ErrConflict) || call.Err() != nil {
		t.Fatal("stale update cancelled session", err)
	}
	audioTagsChange(t, c, "disable")
	if call.Err() == nil {
		t.Fatal("in-flight extraction not cancelled")
	}
	if unlock, e := c.recognitionCommit(call); e == nil {
		unlock()
		t.Fatal("cancelled extraction can commit")
	}
	finish()
	os.WriteFile(filepath.Join(music, "two.mp3"), testMP3(), 0600)
	if _, err = c.scan(ctx, l, false); err != nil {
		t.Fatal(err)
	}
	for _, item := range getItems(t, c) {
		if item.ID == original.ID {
			if item.Metadata.Title != "Titre de test" || item.Metadata.ImageID != original.Metadata.ImageID {
				t.Fatal("cached tags lost")
			}
		} else if item.Metadata.Title != "two" || item.Metadata.Provider != "filename" {
			t.Fatal("native fallback still running", item.Metadata)
		}
	}
	audioTagsChange(t, c, "remove")
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	s, err := next.audioTagsState(ctx)
	if err != nil || s.Installed {
		t.Fatal("removed module resurrected", s, err)
	}
	audioTagsChange(t, next, "install")
	s, _ = next.audioTagsState(ctx)
	if s.Enabled {
		t.Fatal("reinstall enabled unexpectedly")
	}
	audioTagsChange(t, next, "enable")
	// Unchanged items are never sent to the tag reader.
	root, err := os.OpenRoot(music)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	info, _ := os.Stat(filepath.Join(music, "one.mp3"))
	spool := &artworkSpool{}
	defer spool.close()
	if err = next.stageAudioTags(ctx, &audioTagReader{}, root, l, []discoveredMedia{{rel: "one.mp3", info: info}}, spool); err != nil || spool.file != nil {
		t.Fatal("unchanged file parsed", err)
	}
	// A corrupt active package must fail before changing the existing catalogue.
	next.db.Exec("UPDATE recognition_packages SET code=? WHERE kind='audiotags'", []byte("invalid"))
	if _, err = next.scan(ctx, l, false); err == nil {
		t.Fatal("corrupt package accepted")
	}
	if len(getItems(t, next)) != 2 {
		t.Fatal("catalogue damaged")
	}
	next.extensionSafeMode = true
	if _, r, done, e := next.openAudioTags(ctx); e != nil || r != nil {
		done()
		t.Fatal("safe mode loaded guest", e)
	} else {
		done()
	}
}
func TestAudioTagsHTTPPermissions(t *testing.T) {
	c, h, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"name": "Admin", "password": accountTestPassword, "code": c.auth.code})
	endpoint := "/api/v1/audio-tags"
	if w := admin.request("GET", endpoint, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := admin.request("POST", endpoint, map[string]any{"operation": "disable"}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	s, _ := c.audioTagsState(context.Background())
	if w := admin.request("POST", endpoint, map[string]any{"operation": "disable", "revision": s.Revision}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	admin.create("Viewer", accountTestPassword, map[string]libraryGrant{})
	user := &accountClient{t: t, h: h}
	user.enter("login", map[string]string{"name": "Viewer", "password": accountTestPassword})
	for _, method := range []string{"GET", "POST"} {
		if w := user.request(method, endpoint, map[string]any{"operation": "remove", "revision": s.Revision}); w.Code != 403 {
			t.Fatal("viewer accessed module", w.Code)
		}
	}
}
func TestAudioTagsMigration19SeedsOnlyOnce(t *testing.T) {
	c, dir := testCatalog(t)
	removeAudioTagsForLegacyFixture(t, c)
	c.db.Exec("PRAGMA user_version=19")
	c.close()
	next, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, e := next.audioTagsState(context.Background())
	if e != nil || !s.Installed || !s.Enabled {
		t.Fatal(s, e)
	}
	audioTagsChange(t, next, "disable")
	next.close()
	last, err := openCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer last.close()
	s, e = last.audioTagsState(context.Background())
	if e != nil || s.Enabled {
		t.Fatal("disabled state lost", s, e)
	}
}
