package main

import (
	"context"
	"io"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"mosaune/extensionkit/registry"
	"net/http"
	"strings"
	"testing"
)

func TestMusicRequestExecution(t *testing.T) {
	p := newMusicBrainzProvider(nil)
	calls := 0
	p.client.Transport = tmdbFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "musicbrainz.org" || r.URL.Path != "/ws/2/release/" || r.Method != "GET" || r.URL.Query().Get("query") != `release:"Album" AND artist:"Singer"` || r.URL.Query().Get("offset") != "20" || r.URL.Query().Get("fmt") != "json" || r.Header.Get("User-Agent") == "" {
			t.Fatal("plan not executed correctly", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"count":21,"releases":[{"id":"11111111-1111-1111-1111-111111111111","title":"Album"}]}`))}, nil
	})
	q := d.Request{Operation: "match", Title: "Album", Artist: "Singer", Page: 2}
	out, e := p.searchMusic(context.Background(), q)
	if e != nil || calls != 1 || out.Page != 2 || out.Pages != 2 || len(out.Items) != 1 {
		t.Fatal(out, e, calls)
	}
	q.Page = 51
	if _, e = p.searchMusic(context.Background(), q); e == nil || calls != 1 {
		t.Fatal("invalid request reached network", e, calls)
	}
}

func TestMusicRequestFactoryUpgrade(t *testing.T) {
	for _, mode := range []string{"enabled", "disabled", "removed", "modified", "imported"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			store := registry.SQLStore{DB: c.db}
			s, e := store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			entry, _ := registryEntryFor(s, "mosaune.music.results")
			entry.Descriptor.Version = "0.1.0"
			entry.Enabled = false
			plan, e := store.Commit(ctx, s.Revision, registry.Change{Operation: "restore", ID: entry.Descriptor.ID, Entry: &entry})
			if e != nil {
				t.Fatal(e)
			}
			if mode == "disabled" || mode == "removed" {
				op := "disable"
				if mode == "removed" {
					op = "remove"
				}
				if e = c.changeRecognition(ctx, "music-results", op, plan.After.Revision); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "modified" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET code=? WHERE id=?", []byte("modified bytes"), entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "imported" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET manifest='{}' WHERE id=?", entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = c.db.Exec("PRAGMA user_version=34"); e != nil {
				t.Fatal(e)
			}
			if e = migrateMusicRequests(c.db); e != nil {
				t.Fatal(e)
			}
			s, e = store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			next, present := registryEntryFor(s, entry.Descriptor.ID)
			if mode == "removed" {
				if present {
					t.Fatal("removed package restored")
				}
				return
			}
			want := "0.2.0"
			if mode == "modified" || mode == "imported" {
				want = "0.1.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(next, mode)
			}
			var selected string
			if e = c.db.QueryRow("SELECT id FROM recognition_selection WHERE kind='music-results'").Scan(&selected); e != nil || selected != entry.Descriptor.ID {
				t.Fatal("selection changed", selected, e)
			}
			if e = migrateMusicRequests(c.db); e != nil {
				t.Fatal("second migration", e)
			}
		})
	}
}
