package main

import (
	"math"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"testing"
)

func TestTrackingStatesAndScope(t *testing.T) {
	season := 2
	base := a.Tracking{Connector: "sonarr", RemoteID: 12, Season: 2, Monitored: true, Seasons: []a.Season{{Number: 2, Monitored: true}}, Episodes: []a.Episode{{ID: 1, Season: 2, Number: 1}, {ID: 2, Season: 2, Number: 2}, {ID: 3, Season: 1, Number: 1, HasFile: true}}}
	for _, name := range []string{"waiting", "unmonitored", "uncertain", "downloading", "blocked", "imported", "partial", "available", "other-season", "unknown-season", "season-pack", "zero-size", "over-left"} {
		t.Run(name, func(t *testing.T) {
			q := base
			q.Episodes = append([]a.Episode(nil), base.Episodes...)
			want, progress := name, float64(0)
			switch name {
			case "unmonitored":
				q.Monitored = false
			case "uncertain":
				q.Uncertain = true
			case "downloading", "blocked":
				q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 1, Size: 100, Remaining: 25}, {RemoteID: 12, EpisodeID: 3, Size: 100, Remaining: 100}, {RemoteID: 99, EpisodeID: 1, Size: 100, Remaining: 100}}
				progress = 75
				if name == "blocked" {
					q.Queue[0].Status = "warning"
				}
			case "imported", "partial", "available":
				q.Episodes[0].HasFile = true
				if name != "imported" {
					q.Local = 1
				}
				if name == "available" {
					q.Episodes[1].HasFile = true
					q.Local = 2
					q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 1, Status: "error"}}
				}
			case "other-season":
				q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 3, Status: "error"}}
				want = "waiting"
			case "unknown-season":
				q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 999}}
				want = "waiting"
			case "season-pack":
				q.Queue = []a.QueueItem{{RemoteID: 12, Season: &season, Size: 200, Remaining: 50}}
				want = "downloading"
				progress = 75
			case "zero-size":
				q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 1}}
				want = "downloading"
			case "over-left":
				q.Queue = []a.QueueItem{{RemoteID: 12, EpisodeID: 1, Size: 1, Remaining: 2}}
				want = "downloading"
			}
			if e := q.Validate(); e != nil {
				t.Fatal(e)
			}
			r, e := track(q)
			if e != nil || r.State != want || r.Total != 2 || r.Progress != progress {
				t.Fatal(r, e)
			}
		})
	}
	q := a.Tracking{Connector: "radarr", Season: -1}
	r, e := track(q)
	if e != nil || r.State != "missing" || r.Total != 0 {
		t.Fatal(r, e)
	}
	q.RemoteID = 9
	q.HasFile = true
	q.Local = 1
	r, e = track(q)
	if e != nil || r.State != "available" || r.Total != 1 || r.Imported != 1 {
		t.Fatal(r, e)
	}
}

func TestTrackingEvidenceGuards(t *testing.T) {
	q := a.Tracking{Connector: "sonarr", RemoteID: 12, Season: 2, Monitored: true, Episodes: []a.Episode{{ID: 1, Season: 2, Number: 1, HasFile: true}, {ID: 2, Season: 2, Number: 2}}}
	r, _ := track(q)
	for _, name := range []string{"available-without-file", "invented-count", "negative-progress", "nan", "infinite", "unknown-state", "fake-missing", "partial-without-file", "long-message"} {
		t.Run(name, func(t *testing.T) {
			bad := r
			switch name {
			case "available-without-file":
				bad.State = "available"
			case "invented-count":
				bad.Total = 1
			case "negative-progress":
				bad.Progress = -1
			case "nan":
				bad.Progress = math.NaN()
			case "infinite":
				bad.Progress = math.Inf(1)
			case "unknown-state":
				bad.State = "ready"
			case "fake-missing":
				bad.State = "missing"
			case "partial-without-file":
				bad.State = "partial"
			case "long-message":
				bad.Message = string(make([]byte, 1025))
			}
			if bad.Validate(q) == nil {
				t.Fatal("invalid result accepted", name)
			}
		})
	}
	for _, name := range []string{"foreign-local", "duplicate-episode", "too-many-episodes", "too-many-queue", "non-finite-size"} {
		t.Run(name, func(t *testing.T) {
			bad := q
			switch name {
			case "foreign-local":
				bad.Local = 2
			case "duplicate-episode":
				bad.Episodes = append(append([]a.Episode{}, q.Episodes...), q.Episodes[0])
			case "too-many-episodes":
				bad.Episodes = make([]a.Episode, 5001)
			case "too-many-queue":
				bad.Queue = make([]a.QueueItem, 2001)
			case "non-finite-size":
				bad.Queue = []a.QueueItem{{Size: math.Inf(1)}}
			}
			if bad.Validate() == nil {
				t.Fatal("invalid observations accepted")
			}
		})
	}
}
