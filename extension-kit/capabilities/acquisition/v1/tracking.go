package acquisition

import (
	"errors"
	"math"
	"slices"
	"unicode/utf8"
)

// Tracking carries only bounded observations, never credentials, paths or titles.
type Episode struct {
	ID      int  `json:"id"`
	Season  int  `json:"season"`
	Number  int  `json:"number"`
	HasFile bool `json:"hasFile"`
}
type Season struct {
	Number    int  `json:"number"`
	Monitored bool `json:"monitored"`
}
type QueueItem struct {
	RemoteID  int     `json:"remoteId"`
	EpisodeID int     `json:"episodeId"`
	Season    *int    `json:"season,omitempty"`
	Size      float64 `json:"size"`
	Remaining float64 `json:"remaining"`
	Status    string  `json:"status"`
}
type Tracking struct {
	Connector string      `json:"connector"`
	RemoteID  int         `json:"remoteId"`
	Season    int         `json:"season"`
	HasFile   bool        `json:"hasFile"`
	Monitored bool        `json:"monitored"`
	Uncertain bool        `json:"uncertain"`
	Seasons   []Season    `json:"seasons"`
	Episodes  []Episode   `json:"episodes"`
	Queue     []QueueItem `json:"queue"`
	Local     int         `json:"local"`
}

// Counts is evidence validation, not a download-state policy. The host retains
// authority over actual files and the confirmed season scope.
func (q Tracking) Counts() (total, imported int) {
	if q.RemoteID == 0 {
		return
	}
	if q.Connector == "radarr" {
		total = 1
		if q.HasFile {
			imported = 1
		}
		return
	}
	for _, ep := range q.Episodes {
		if q.Season >= 0 && ep.Season != q.Season {
			continue
		}
		total++
		if ep.HasFile {
			imported++
		}
	}
	return
}
func (q Tracking) Validate() error {
	bad := errors.New("invalid tracking observations")
	if !slices.Contains([]string{"radarr", "sonarr"}, q.Connector) || q.RemoteID < 0 || q.Season < -1 || q.Season > 1000 || len(q.Episodes) > 5000 || len(q.Queue) > 2000 || len(q.Seasons) > 1001 {
		return bad
	}
	if q.Connector == "radarr" && (q.Season != -1 || len(q.Episodes) != 0 || len(q.Seasons) != 0) {
		return bad
	}
	if q.RemoteID == 0 && (q.HasFile || len(q.Episodes) != 0 || len(q.Queue) != 0 || len(q.Seasons) != 0) {
		return bad
	}
	ids := map[int]bool{}
	keys := map[[2]int]bool{}
	for _, ep := range q.Episodes {
		key := [2]int{ep.Season, ep.Number}
		if ep.ID <= 0 || ep.Season < 0 || ep.Season > 1000 || ep.Number < 0 || ep.Number > 100000 || ids[ep.ID] || keys[key] {
			return bad
		}
		ids[ep.ID], keys[key] = true, true
	}
	seen := map[int]bool{}
	for _, s := range q.Seasons {
		if s.Number < 0 || s.Number > 1000 || seen[s.Number] {
			return bad
		}
		seen[s.Number] = true
	}
	for _, row := range q.Queue {
		if row.RemoteID < 0 || row.EpisodeID < 0 || row.Season != nil && (*row.Season < 0 || *row.Season > 1000) || !finite(row.Size) || !finite(row.Remaining) || row.Size < 0 || row.Remaining < 0 || row.Size > 1e18 || row.Remaining > 1e18 || len(row.Status) > 64 || !utf8.ValidString(row.Status) {
			return bad
		}
	}
	_, imported := q.Counts()
	if q.Local < 0 || q.Local > imported {
		return bad
	}
	return nil
}

type TrackingResult struct {
	Total    int     `json:"total"`
	Imported int     `json:"imported"`
	Progress float64 `json:"progress"`
	State    string  `json:"state"`
	Message  string  `json:"message"`
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func (r TrackingResult) Validate(q Tracking) error {
	bad := errors.New("invalid tracking result")
	total, imported := q.Counts()
	if q.Validate() != nil || r.Total != total || r.Imported != imported || !finite(r.Progress) || r.Progress < 0 || r.Progress > 100 || len(r.Message) == 0 || len(r.Message) > 1024 || !utf8.ValidString(r.Message) || !slices.Contains([]string{"missing", "waiting", "unmonitored", "uncertain", "downloading", "blocked", "available", "partial", "imported"}, r.State) {
		return bad
	}
	if (q.RemoteID == 0) != (r.State == "missing") {
		return bad
	}
	if r.State == "available" && (total == 0 || q.Local < total) {
		return bad
	}
	if r.State == "partial" && (q.Local == 0 || q.Local >= total) {
		return bad
	}
	if r.State == "imported" && (imported == 0 || q.Local != 0) {
		return bad
	}
	return nil
}
