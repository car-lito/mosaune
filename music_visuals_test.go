package main

import (
	"context"
	"mosaune/extensionkit/registry"
	"net/http"
	"testing"
	"time"
)

func TestMusicVisualDisableCancelsFetch(t *testing.T) {
	c, _ := testCatalog(t)
	ctx := context.Background()
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	started := make(chan struct{})
	finished := make(chan error, 1)
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	go func() { _, e := c.visualImage(ctx, p, visualRelease); finished <- e }()
	select {
	case <-started:
	case <-time.After(15 * time.Second):
		t.Fatal("image request did not start")
	}
	s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.changeRecognition(ctx, "music-results", "disable", s.Revision); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-finished:
		if e == nil {
			t.Fatal("disabled request succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("image request not cancelled")
	}
	var count int
	if e = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE key=?", "visual-image:v2:"+visualRelease).Scan(&count); e != nil || count != 0 {
		t.Fatal("cancelled request stored a negative result", e, count)
	}
}

func TestMusicVisualDisabledKeepsExpiredImages(t *testing.T) {
	c, dir := testCatalog(t)
	ctx := context.Background()
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	calls := 0
	p.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return visualResponse(200, string(testPicture())), nil
	})
	if data, e := c.visualImage(ctx, p, visualRelease); e != nil || len(data) == 0 {
		t.Fatal(e)
	}
	if _, e := c.db.Exec("UPDATE provider_cache SET expires=0 WHERE key=?", "visual-image:v2:"+visualRelease); e != nil {
		t.Fatal(e)
	}
	s, e := (registry.SQLStore{DB: c.db}).Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.changeRecognition(ctx, "music-results", "disable", s.Revision); e != nil {
		t.Fatal(e)
	}
	before := calls
	if data, e := c.visualImage(ctx, p, visualRelease); e != nil || len(data) == 0 || calls != before {
		t.Fatal("stored image unavailable", e, calls)
	}
	if _, e = c.visualImage(ctx, p, visualGroup); e == nil || calls != before {
		t.Fatal("disabled component contacted source", e, calls)
	}
	var negative int
	if e = c.db.QueryRow("SELECT count(*) FROM provider_cache WHERE key=?", "visual-image:v2:"+visualGroup).Scan(&negative); e != nil || negative != 0 {
		t.Fatal("disabled request poisoned image cache", e, negative)
	}
	c.close()
	c, e = openCatalog(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer c.close()
	p = c.provider("musicbrainz").(*musicBrainzProvider)
	if data, e := c.visualImage(ctx, p, visualRelease); e != nil || len(data) == 0 {
		t.Fatal("stored image lost after restart", e)
	}
}

func TestMusicVisualFactoryUpgrade(t *testing.T) {
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
			entry.Descriptor.Version = "0.2.0"
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
				if _, e = c.db.Exec("UPDATE recognition_packages SET code=? WHERE id=?", []byte("altered"), entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "imported" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET manifest='{}' WHERE id=?", entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = c.db.Exec("PRAGMA user_version=35"); e != nil {
				t.Fatal(e)
			}
			if e = migrateMusicVisuals(c.db); e != nil {
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
			want := "0.4.0"
			if mode == "modified" || mode == "imported" {
				want = "0.2.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
