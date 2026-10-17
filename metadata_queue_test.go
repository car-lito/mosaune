package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type queueFixture struct {
	sampleProvider
	fail        bool
	searchCount int
	started     chan struct{}
	release     chan struct{}
}

func (p *queueFixture) Search(ctx context.Context, _ string, q MetadataQuery) ([]MetadataCandidate, error) {
	p.searchCount++
	if p.started != nil {
		close(p.started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.release:
		}
	}
	if p.fail {
		return nil, errors.New("Fournisseur temporairement indisponible.")
	}
	if q.Title == "Ambiguous" {
		return []MetadataCandidate{}, nil
	}
	return []MetadataCandidate{{ID: q.Title, Title: q.Title, Year: q.Year}}, nil
}
func allKinds(string) bool { return true }
func queueState(t *testing.T, c *catalog) metadataQueueStatus {
	t.Helper()
	s, e := c.metadataStatus(testContext(context.Background()), allKinds)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestMetadataQueueLargeLibraryAndRestart(t *testing.T) {
	c, dbDir := testCatalog(t)
	dir := t.TempDir()
	ctx := testContext(context.Background())
	for i := 0; i < 1001; i++ {
		putFile(t, dir, fmt.Sprintf("Example%04d.2020.mp4", i), "fixture")
	}
	if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	p := &queueFixture{}
	c.providers = []MetadataProvider{p}
	for i := 0; i < 25; i++ {
		c.metadataStep(ctx, allKinds)
	}
	s := queueState(t, c)
	if s.Libraries[0].Matched != 25 || s.Libraries[0].Pending != 976 {
		t.Fatal(s)
	}
	h, token := handlerFor(t, c, "video")
	if w := call(t, h, "POST", "/api/v1/metadata/queue/pause", token, nil); w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	c.metadataStep(ctx, allKinds)
	if p.searchCount != 25 {
		t.Fatal("paused worker advanced")
	}
	c.close()
	next, e := openCatalog(dbDir)
	if e != nil {
		t.Fatal(e)
	}
	defer next.close()
	next.providers = []MetadataProvider{p}
	if !queueState(t, next).Paused {
		t.Fatal("pause was not persisted")
	}
	h, token = handlerFor(t, next, "video")
	if w := call(t, h, "POST", "/api/v1/metadata/queue/resume", token, nil); w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	for i := 0; i < 976; i++ {
		next.metadataStep(ctx, allKinds)
	}
	s = queueState(t, next)
	if s.State != "complete" || s.Libraries[0].Matched != 1001 || p.searchCount != 1001 {
		t.Fatal(s, p.searchCount)
	}
	next.metadataStep(ctx, allKinds)
	if p.searchCount != 1001 {
		t.Fatal("reprocessed completed files")
	}
}
func TestMetadataQueueFailureBackoffAndRetry(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	ctx := testContext(context.Background())
	putFile(t, dir, "Example.2020.mp4", "x")
	if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	p := &queueFixture{fail: true}
	c.providers = []MetadataProvider{p}
	for n := 1; n <= 3; n++ {
		c.metadataStep(ctx, allKinds)
		s := queueState(t, c)
		if s.Libraries[0].Failed != 1 || s.Errors[0].Attempts != n {
			t.Fatal(s)
		}
		c.metadataStep(ctx, allKinds)
		if p.searchCount != n {
			t.Fatal("provider hammered during backoff")
		}
		if _, e := c.db.Exec(`UPDATE metadata_queue_state SET retry_at=0;UPDATE metadata_failures SET retry_at=0`); e != nil {
			t.Fatal(e)
		}
	}
	c.metadataStep(ctx, allKinds)
	if p.searchCount != 3 || queueState(t, c).State != "errors" {
		t.Fatal("retry limit ignored")
	}
	h, token := handlerFor(t, c, "video")
	if w := call(t, h, "POST", "/api/v1/metadata/queue/retry", "", nil); w.Code != 403 {
		t.Fatal("missing token accepted")
	}
	if w := call(t, h, "POST", "/api/v1/metadata/queue/retry", token, nil); w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	p.fail = false
	c.metadataStep(ctx, allKinds)
	s := queueState(t, c)
	if s.Libraries[0].Matched != 1 || s.Libraries[0].Failed != 0 || len(s.Errors) != 0 {
		t.Fatal(s)
	}
}
func TestMetadataQueueReviewAndModuleGuard(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	ctx := testContext(context.Background())
	putFile(t, dir, "Ambiguous.mp4", "x")
	if _, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	p := &queueFixture{}
	c.providers = []MetadataProvider{p}
	c.metadataStep(ctx, func(string) bool { return false })
	if p.searchCount != 0 {
		t.Fatal("disabled module processed")
	}
	c.metadataStep(ctx, allKinds)
	c.metadataStep(ctx, allKinds)
	if queueState(t, c).Libraries[0].Review != 1 || p.searchCount != 1 {
		t.Fatal("ambiguous identification repeated")
	}
	h, token := handlerFor(t, c, "music")
	if w := call(t, h, "POST", "/api/v1/metadata/queue/resume", token, nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
}
func TestMetadataQueueCancelKeepsPending(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "Example.mp4", "x")
	if _, e := c.scan(testContext(context.Background()), library{Path: dir, Kind: "films"}, true); e != nil {
		t.Fatal(e)
	}
	p := &queueFixture{started: make(chan struct{}), release: make(chan struct{})}
	c.providers = []MetadataProvider{p}
	ctx, cancel := context.WithCancel(testContext(context.Background()))
	defer cancel()
	done := make(chan struct{})
	go func() { c.metadataStep(ctx, allKinds); close(done) }()
	select {
	case <-p.started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not start")
	}
	if queueState(t, c).Current == "" {
		t.Fatal("no live title")
	}
	h, token := handlerFor(t, c, "video")
	if w := call(t, h, "POST", "/api/v1/metadata/queue/pause", token, nil); w.Code != 202 {
		t.Fatal(w.Code)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not cancel")
	}
	s := queueState(t, c)
	if s.Libraries[0].Pending != 1 || s.Libraries[0].Failed != 0 || s.Current != "" {
		t.Fatal(s)
	}
}
func TestAcquisitionPriorityUsesExactPath(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	for i := 0; i < 30; i++ {
		putFile(t, dir, fmt.Sprintf("Old%d.mp4", i), "x")
	}
	if e := os.Mkdir(filepath.Join(dir, "Arrival"), 0700); e != nil {
		t.Fatal(e)
	}
	putFile(t, dir, "Arrival/opaque.mp4", "x")
	l, e := c.scan(ctx, library{Path: dir, Kind: "films"}, true)
	if e != nil {
		t.Fatal(e)
	}
	work, e := c.rememberWork(ctx, "films", Metadata{Provider: "sample", ProviderID: "known-42", Title: "Arrival"}, true)
	if e != nil {
		t.Fatal(e)
	}
	p := &queueFixture{}
	c.providers = []MetadataProvider{p}
	cfg := arrConfig{Library: l.ID, Root: "/movies"}
	obj := arrObject{"path": "/movies/Arrival", "movieFile": map[string]any{"relativePath": "opaque.mp4"}}
	if e = c.prioritizeImport(ctx, arrClient{cfg}, acquisitionRequest{Library: l.ID, WorkID: work, Kind: "films", Root: "/movies"}, obj, nil); e != nil {
		t.Fatal(e)
	}
	c.metadataStep(ctx, allKinds)
	if p.searchCount != 0 {
		t.Fatal("priority used filename search")
	}
	found := false
	for _, m := range getItems(t, c) {
		if m.Filename == "Arrival/opaque.mp4" {
			found = m.Metadata.ProviderID == "known-42"
		}
	}
	if !found {
		t.Fatal("confirmed import not identified first")
	}
	for _, bad := range []struct{ root, dir, file string }{{"/movies", "/movies-other/A", "a.mp4"}, {"/movies", "/movies/A", "../../a.mp4"}, {"/movies", "/movies/A", "/a.mp4"}, {"/movies", "/movies/A", "C:\\a.mp4"}} {
		if _, ok := mappedImport(bad.root, bad.dir, bad.file); ok {
			t.Fatal("unsafe mapping", bad)
		}
	}
	if p, ok := mappedImport(`D:\Films`, `D:\Films\Arrival`, `Arrival.mkv`); !ok || p != "Arrival/Arrival.mkv" {
		t.Fatal(p, ok)
	}
}
func TestScanProgressAndRollback(t *testing.T) {
	c, _ := testCatalog(t)
	dir := t.TempDir()
	putFile(t, dir, "Movie.mp4", "x")
	ctx := testContext(context.Background())
	l, e := c.scan(ctx, library{Path: dir, Name: "Films NAS", Kind: "films"}, true)
	if e != nil {
		t.Fatal(e)
	}
	s := c.getScanProgress()
	if s.State != "complete" || s.Total != 1 || s.Processed != 1 || s.Found != 1 {
		t.Fatal(s)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = c.scan(cancelled, l, false); e == nil {
		t.Fatal("cancel ignored")
	}
	if c.getScanProgress().State != "failed" || len(getItems(t, c)) != 1 {
		t.Fatal("failed scan erased catalog")
	}
	h, _ := handlerFor(t, c, "video")
	// Polling does not need the only SQLite connection held by an indexing transaction.
	tx, e := c.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	w := call(t, h, "GET", "/api/v1/library/scan", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
