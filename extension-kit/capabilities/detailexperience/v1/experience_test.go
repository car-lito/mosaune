package detailexperience

import (
	"encoding/json"
	"strings"
	"testing"
)

func fixture() (Query, Result) {
	q := Query{Kind: "films", Files: []File{{ID: "visible", Name: "Film"}}, People: []string{"Person"}}
	r := Result{Cover: "portrait", Actions: []Action{}, People: []int{}, Sections: []Section{{Key: "files", Rows: []Row{{Key: "one", FileID: "visible", Title: "Film", Actions: []Action{{Operation: "play", Label: "Lire", FileID: "visible"}}}}, Actions: []Action{}}}}
	return q, r
}
func TestRejectUnsafeOrIncompleteProjection(t *testing.T) {
	cases := map[string]func(*Result){"foreign file": func(r *Result) { r.Sections[0].Rows[0].FileID = "private" }, "foreign action": func(r *Result) {
		r.Actions = append(r.Actions, Action{Operation: "play", Label: "Lire", FileID: "private"})
	}, "arbitrary operation": func(r *Result) { r.Actions = append(r.Actions, Action{Operation: "fetch", Label: "Go"}) }, "missing file": func(r *Result) { r.Sections[0].Rows = []Row{} }, "foreign person": func(r *Result) { r.People = []int{1} }, "repeated person": func(r *Result) { r.People = []int{0, 0} }, "request without work": func(r *Result) { r.Actions = append(r.Actions, Action{Operation: "request", Label: "Go"}) }, "nil rows": func(r *Result) { r.Sections[0].Rows = nil }, "nil actions": func(r *Result) { r.Sections[0].Rows[0].Actions = nil }, "oversized": func(r *Result) { r.Sections[0].Help = strings.Repeat("x", 5000) }, "bypass file coverage": func(r *Result) { r.EpisodeList = true; r.Sections = []Section{} }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			q, r := fixture()
			if r.Validate(q) != nil {
				t.Fatal("invalid fixture")
			}
			mutate(&r)
			if r.Validate(q) == nil {
				t.Fatal("unsafe result accepted")
			}
		})
	}
}
func TestOnlyKnownJSONFieldsAndReferences(t *testing.T) {
	q, r := fixture()
	q.Focus = "hidden"
	if r.Validate(q) == nil {
		t.Fatal("foreign focus")
	}
	q, r = fixture()
	q.Files = append(q.Files, q.Files[0])
	if r.Validate(q) == nil {
		t.Fatal("duplicate identities")
	}
	q, r = fixture()
	b, _ := json.Marshal(r)
	if strings.Contains(string(b), "url") {
		t.Fatal("unexpected address")
	}
}
