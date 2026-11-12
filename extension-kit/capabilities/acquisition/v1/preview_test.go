package acquisition

import (
	"encoding/json"
	"strings"
	"testing"
)

func fixturePreviewPlan() PreviewPlan {
	return PreviewPlan{Version: 1, Connector: "sonarr", Reads: []string{"status", "find", "roots", "profiles"}, Missing: "lookup", MissingMessage: "Absent", SeasonMessage: "Saison absente", ProfilePrefix: "Profil #"}
}
func TestPreviewGuardsAndPrivateDestination(t *testing.T) {
	q := PreviewQuery{Kind: "series", Season: 1}
	p := fixturePreviewPlan()
	for _, mutate := range []func(*PreviewPlan){
		func(v *PreviewPlan) { v.Reads[0] = "find" }, func(v *PreviewPlan) { v.Reads[2] = "update" }, func(v *PreviewPlan) { v.Connector = "radarr" }, func(v *PreviewPlan) { v.Missing = "create" }, func(v *PreviewPlan) { v.ProfilePrefix = "bad\n" },
	} {
		bad := p
		bad.Reads = append([]string(nil), p.Reads...)
		mutate(&bad)
		if bad.Validate(q) == nil {
			t.Fatal("unsafe plan accepted", bad)
		}
	}
	obj := map[string]any{"id": json.Number("12"), "path": "/private/existing", "qualityProfileId": json.Number("7"), "seasons": []any{map[string]any{"seasonNumber": float64(2)}, map[string]any{"seasonNumber": float64(1)}}}
	got, err := p.Project(q, true, Bindings{Root: "/new", Profile: 1}, obj, []Choice{{ID: 1, Name: "New"}, {ID: 7, Name: "Existing"}})
	if err != nil || got.Root != "/private/existing" || got.Profile != "Existing" || len(got.Seasons) != 1 || got.Seasons[0] != 1 {
		t.Fatal(got, err)
	}
	payload, _ := json.Marshal(q)
	if strings.Contains(string(payload), "private") {
		t.Fatal("private destination in guest input")
	}
	q.Season = -1
	got, err = p.Project(q, true, Bindings{}, obj, nil)
	if err != nil || got.Profile != "Profil #7" || len(got.Seasons) != 2 || got.Seasons[0] != 1 {
		t.Fatal(got, err)
	}
	q.Season = 5
	if _, err = p.Project(q, true, Bindings{}, obj, nil); err == nil || err.Error() != p.SeasonMessage {
		t.Fatal("absent season accepted", err)
	}
	q.Season = 1
	obj["seasons"] = []any{map[string]any{"seasonNumber": 1.0}, map[string]any{"seasonNumber": 1.0}}
	if _, err = p.Project(q, true, Bindings{}, obj, nil); err == nil {
		t.Fatal("duplicate scope accepted")
	}
	obj["seasons"] = []any{map[string]any{"seasonNumber": 1.5}}
	if _, err = p.Project(q, true, Bindings{}, obj, nil); err == nil {
		t.Fatal("fractional scope accepted")
	}
	obj["seasons"] = []any{map[string]any{"seasonNumber": 1.0}}
	obj["qualityProfileId"] = "7"
	if _, err = p.Project(q, true, Bindings{}, obj, nil); err == nil {
		t.Fatal("malformed profile accepted")
	}
	obj["qualityProfileId"] = 7.0
	delete(obj, "path")
	if _, err = p.Project(q, true, Bindings{}, obj, nil); err == nil {
		t.Fatal("absent existing path fell back to new path")
	}
	if _, err = p.Project(q, false, Bindings{}, obj, nil); err == nil {
		t.Fatal("existing ID in lookup accepted")
	}
	delete(obj, "id")
	got, err = p.Project(q, false, Bindings{Root: "/new", Profile: 4}, obj, nil)
	if err != nil || got.Root != "/new" || got.Profile != "Profil #4" {
		t.Fatal(got, err)
	}
}
