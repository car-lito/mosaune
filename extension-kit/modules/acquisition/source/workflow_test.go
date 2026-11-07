package main

import (
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"reflect"
	"testing"
)

func TestWorkflowScope(t *testing.T) {
	queries := []a.Query{
		{Connector: "radarr", Season: -1}, {Connector: "radarr", RemoteID: 12, Season: -1},
		{Connector: "radarr", RemoteID: 12, HasFile: true, Season: -1},
		{Connector: "sonarr", Season: 2, Seasons: []int{0, 1, 2, 3}},
		{Connector: "sonarr", RemoteID: 12, Season: 2, Seasons: []int{0, 1, 2, 3}},
		{Connector: "sonarr", RemoteID: 12, Season: -1, Seasons: []int{0, 1, 2, 3}},
	}
	for _, q := range queries {
		w, err := workflow(q)
		if err != nil || w.Validate(q) != nil {
			t.Fatal(q, err)
		}
	}
	q := queries[4]
	for _, mode := range []string{"path", "profile", "other-season", "other-id", "drop", "reorder", "duplicate", "skip-monitor", "extra", "binding-and-value"} {
		w, _ := workflow(q)
		switch mode {
		case "path":
			w.Actions[0].Fields["path"] = a.Constant("/elsewhere")
		case "profile":
			w.Actions[0].Fields["qualityProfileId"] = a.Reference("destination.profile")
		case "other-season":
			w.Plan.Seasons = []int{1}
		case "other-id":
			w.Actions[2].Fields["seriesId"] = a.Constant(123)
		case "drop":
			w.Actions[0].Drop = []string{"path"}
		case "reorder":
			w.Actions[1], w.Actions[2] = w.Actions[2], w.Actions[1]
		case "duplicate":
			w.Actions = append(w.Actions, w.Actions[2])
		case "skip-monitor":
			w.Actions = append(w.Actions[:1], w.Actions[2])
		case "extra":
			w.Actions[2].Fields["unapproved"] = a.Constant(true)
		case "binding-and-value":
			w.Actions[2].Fields["seriesId"] = a.Value{Ref: "resource.id", Literal: json.RawMessage(`123`)}
		}
		if w.Validate(q) == nil {
			t.Fatal("scope escape", mode)
		}
	}
}

func TestWorkflowPrivateBindingsAndPreservation(t *testing.T) {
	q := a.Query{Connector: "sonarr", RemoteID: 12, Season: 2, Seasons: []int{1, 2, 3}}
	w, err := workflow(q)
	if err != nil {
		t.Fatal(err)
	}
	original := map[string]any{"id": json.Number("12"), "path": "/existing", "qualityProfileId": json.Number("87"), "unknown": json.Number("9007199254740993"), "seasons": []any{map[string]any{"seasonNumber": float64(1), "monitored": true}, map[string]any{"seasonNumber": float64(2), "monitored": false}, map[string]any{"seasonNumber": float64(3), "monitored": false}}}
	before, _ := json.Marshal(original)
	bindings := a.Bindings{ID: 12, Season: 2, Root: "/new", Profile: 1}
	out, err := w.Actions[0].Resource(original, w.Plan, bindings)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("input mutated")
	}
	if out["path"] != "/existing" || out["qualityProfileId"] != json.Number("87") || out["unknown"] != json.Number("9007199254740993") {
		t.Fatal("existing data lost")
	}
	for i, raw := range out["seasons"].([]any) {
		if raw.(map[string]any)["monitored"] != (i < 2) {
			t.Fatal("unrelated season changed")
		}
	}
	ids, err := a.SelectedEpisodes([]map[string]any{{"id": json.Number("20"), "seasonNumber": float64(2)}, {"id": float64(30), "seasonNumber": float64(3)}}, w.Plan.Seasons)
	if err != nil || !reflect.DeepEqual(ids, []int{20}) {
		t.Fatal(ids, err)
	}
	bindings.Episodes = ids
	body, err := w.Actions[1].Resolve(bindings)
	if err != nil || !reflect.DeepEqual(body["episodeIds"], ids) {
		t.Fatal(body, err)
	}
	q.RemoteID = 0
	w, err = workflow(q)
	if err != nil {
		t.Fatal(err)
	}
	out, err = w.Actions[0].Resource(original, w.Plan, bindings)
	if err != nil {
		t.Fatal(err)
	}
	if out["rootFolderPath"] != "/new" || out["qualityProfileId"] != 1 {
		t.Fatal("destination not bound")
	}
	if _, ok := out["path"]; ok {
		t.Fatal("lookup path retained")
	}
	for i, raw := range out["seasons"].([]any) {
		if raw.(map[string]any)["monitored"] != (i == 1) {
			t.Fatal("lookup flags retained")
		}
	}
	duplicate := []map[string]any{{"id": float64(20), "seasonNumber": float64(2)}, {"id": float64(20), "seasonNumber": float64(2)}}
	if _, err = a.SelectedEpisodes(duplicate, []int{2}); err == nil {
		t.Fatal("duplicate episode accepted")
	}
}
