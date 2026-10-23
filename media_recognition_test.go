package main

import (
	"context"
	"fmt"
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/protocol"
	"testing"
	"time"
)

// Raw factory execution is available only to contract tests; production scans
// always resolve installed, enabled packages through the persistent registry.
func newMediaRecognizer(ctx context.Context, kind string) (*mediaRecognizer, error) {
	d, code, err := modules.Lookup(kind)
	if err != nil {
		return nil, err
	}
	return loadMediaRecognizer(ctx, d, code)
}

func TestRecognitionModulesUseIndependentContracts(t *testing.T) {
	for _, kind := range []string{"films", "series", "music"} {
		t.Run(kind, func(t *testing.T) {
			reader, err := newMediaRecognizer(context.Background(), kind)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.close()
			items, err := reader.recognize(context.Background(), []string{"Album/Chanson.FLAC", "Show/Saison 01/S01E02.mkv", "notes.txt"})
			if err != nil {
				t.Fatal(err)
			}
			if items[0].Accepted != (kind == "music") || items[1].Accepted != (kind != "music") || items[2].Accepted {
				t.Fatal(items)
			}
			if kind == "series" && (items[1].Series != "Show" || items[1].Season != 1 || items[1].Episode != 2) {
				t.Fatal(items)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err = reader.recognize(ctx, []string{"Film.mkv"}); err == nil {
				t.Fatal("cancelled recognition succeeded")
			}
		})
	}
	d, code, _ := modules.Lookup("films")
	if _, err := loadMediaRecognizer(context.Background(), d, []byte("corrupt")); err == nil {
		t.Fatal("corrupt executable loaded")
	}
	d.Services = []protocol.ServiceRequirement{{Name: "files.read.v1", Operations: []string{"read"}}}
	if _, err := loadMediaRecognizer(context.Background(), d, code); err == nil {
		t.Fatal("recognizer requested filesystem service")
	}
	if _, err := newMediaRecognizer(context.Background(), "unknown"); err == nil {
		t.Fatal("implicit movie fallback")
	}
}

func TestRecognitionBatchesAndPreservesCatalogue(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := testContext(context.Background())
	dir := t.TempDir()
	for i := 0; i < recognition.BatchSize*2+1; i++ {
		putFile(t, dir, fmt.Sprintf("Film.%03d.2008.MKV", i), "fixture")
	}
	putFile(t, dir, "unrelated.mp3", "ignored")
	putFile(t, dir, "notes.txt", "ignored")
	l, err := c.scan(ctx, library{Path: dir, Kind: "films"}, true)
	if err != nil {
		t.Fatal(err)
	}
	before := getItems(t, c)
	if len(before) != 65 || l.Count != 65 {
		t.Fatal(l.Count, len(before))
	}
	for _, m := range before {
		if m.Metadata == nil || m.Metadata.Year != 2008 {
			t.Fatalf("missing module metadata: %+v", m)
		}
	}
	// An unavailable module cannot silently substitute film rules or empty a library.
	l.Kind = "unavailable"
	if _, err = c.scan(ctx, l, false); err == nil {
		t.Fatal("missing module accepted")
	}
	after := getItems(t, c)
	if len(after) != len(before) {
		t.Fatal("catalogue lost on failure")
	}
	for i := range before {
		if before[i].ID != after[i].ID || before[i].Revision != after[i].Revision {
			t.Fatal("catalogue mutated")
		}
	}
	if c.getScanProgress().State != "failed" {
		t.Fatal("failure not surfaced")
	}
}

func TestRecognitionThousandNames(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	r, err := newMediaRecognizer(ctx, "series")
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	compiled := time.Since(start)
	start = time.Now()
	for i := 0; i < 32; i++ {
		paths := make([]string, recognition.BatchSize)
		for j := range paths {
			paths[j] = fmt.Sprintf("Example/Saison 01/Example.S01E%02d.Title.1080p.mkv", j+1)
		}
		items, err := r.recognize(ctx, paths)
		if err != nil {
			t.Fatal(err)
		}
		for j, item := range items {
			if item.Episode != j+1 || item.Title != "Title" {
				t.Fatal(item)
			}
		}
	}
	t.Logf("1024 names: engine preparation %s, 32 batches %s (Windows/dev machine, not a Pi benchmark)", compiled, time.Since(start))
}
