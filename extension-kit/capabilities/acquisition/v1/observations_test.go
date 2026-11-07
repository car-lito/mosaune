package acquisition

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestObservationPlanGuards(t *testing.T) {
	query := ObservationQuery{Connector: "sonarr", RemoteID: 7}
	valid := ObservationPlan{Version: 1, Seasons: true, Reads: []ObservationRead{{"queue", 2}, {"episodes", 30}}}
	if valid.Validate(query) != nil {
		t.Fatal("custom order and limits rejected")
	}
	for _, p := range []ObservationPlan{
		{Version: 1, Seasons: true, Reads: []ObservationRead{{"queue", 21}, {"episodes", 30}}},
		{Version: 1, Seasons: true, Reads: []ObservationRead{{"queue", 2}, {"episodes", 5001}}},
		{Version: 1, Seasons: true, Reads: []ObservationRead{{"queue", 2}}},
		{Version: 1, Seasons: true, Reads: []ObservationRead{{"queue", 2}, {"queue", 2}}},
		{Version: 1, Seasons: true, Reads: []ObservationRead{{"update", 2}, {"episodes", 30}}},
		{Version: 1, Reads: valid.Reads},
	} {
		if p.Validate(query) == nil {
			t.Fatal("unsafe plan accepted", p)
		}
	}
	if valid.Validate(ObservationQuery{Connector: "sonarr"}) == nil {
		t.Fatal("reads without resource")
	}
	if valid.Validate(ObservationQuery{Connector: "radarr", RemoteID: 7}) == nil {
		t.Fatal("series reads for movie")
	}
}

func TestObservationPrivateProjectionAndBounds(t *testing.T) {
	p := ObservationPlan{Version: 1, Seasons: true, Reads: []ObservationRead{{"episodes", 3}, {"queue", 2}}}
	query := ObservationQuery{Connector: "sonarr", RemoteID: 7}
	obj := map[string]any{"id": json.Number("7"), "path": "secret-path", "seasons": []any{map[string]any{"seasonNumber": float64(1), "monitored": true}}}
	q, err := p.Begin(query, 1, true, obj)
	if err != nil {
		t.Fatal(err)
	}
	episodes := []map[string]any{{"id": json.Number("11"), "seasonNumber": 1, "episodeNumber": 1, "hasFile": true, "path": "secret-file"}}
	if err = q.ObserveEpisodes(episodes, 3); err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{{"resourceId": 7, "episodeId": 11, "size": json.Number("100"), "sizeleft": float64(25), "downloadId": "secret-id"}}
	if more, err := q.ObserveQueue(rows, 1, 1, 2); more || err != nil {
		t.Fatal(more, err)
	}
	raw, _ := json.Marshal(q)
	if strings.Contains(string(raw), "secret-") {
		t.Fatal("private data escaped")
	}
	if q.Queue[0].Remaining != 25 || len(q.Episodes) != 1 || !q.Uncertain {
		t.Fatal(q)
	}
	for _, row := range []map[string]any{
		{"resourceId": 1.5}, {"size": math.NaN()}, {"sizeleft": -1.0}, {"trackedDownloadStatus": true},
	} {
		copy := q
		if _, err = copy.ObserveQueue([]map[string]any{row}, 1, 1, 2); err == nil {
			t.Fatal("malformed evidence accepted", row)
		}
	}
	for _, args := range [][3]int{{201, 1, 2}, {101, 1, 2}, {1, 3, 2}, {-1, 1, 2}} {
		copy := q
		if _, err = copy.ObserveQueue(rows, args[0], args[1], args[2]); err == nil {
			t.Fatal("incomplete or oversized page accepted", args)
		}
	}
	if err = q.ObserveEpisodes(episodes, 3); err == nil {
		t.Fatal("duplicate episode accepted")
	}
	obj["id"] = 7.5
	if _, err = p.Begin(query, 1, false, obj); err == nil {
		t.Fatal("fractional ID accepted")
	}
	obj["id"] = 7
	obj["hasFile"] = "yes"
	if _, err = p.Begin(query, 1, false, obj); err == nil {
		t.Fatal("malformed file flag accepted")
	}
}
