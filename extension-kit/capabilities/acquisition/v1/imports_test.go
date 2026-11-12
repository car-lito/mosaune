package acquisition

import "testing"

func TestImportEvidenceAndScope(t *testing.T) {
	q := PreviewQuery{Kind: "series", Season: 1}
	ep := func(file, season, n int) map[string]any {
		return map[string]any{"hasFile": true, "episodeFileId": file, "seasonNumber": season, "episodeNumber": n}
	}
	files := []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}}
	rows := []map[string]any{ep(1, 1, 1), ep(2, 2, 1), ep(3, 1, 2), ep(3, 1, 3)}
	candidates, err := ProjectImportCandidates(q, rows, files)
	if err != nil || len(candidates) != 2 {
		t.Fatal(candidates, err)
	}
	query := ImportQuery{Scope: q, Candidates: candidates}
	plan := ImportPlan{Version: 1, Mode: "verified", Limit: 5000, Reads: []string{"episodefiles"}}
	if err = (ImportResult{Slots: []int{0}}).Validate(query, plan); err != nil {
		t.Fatal(err)
	}
	for _, slots := range [][]int{{1}, {2}, {99}, {0, 0}} {
		if (ImportResult{Slots: slots}).Validate(query, plan) == nil {
			t.Fatal("unsafe selection", slots)
		}
	}
	rows = append(rows, ep(2, 1, 1))
	candidates, err = ProjectImportCandidates(q, rows, files)
	if err != nil || len(candidates) != 0 {
		t.Fatal("duplicate identity trusted", candidates, err)
	}
	candidates, err = ProjectImportCandidates(q, []map[string]any{ep(1, 1, 1)}, []map[string]any{{"id": 1}, {"id": 1}})
	if err != nil || len(candidates) != 0 {
		t.Fatal("duplicate file trusted")
	}
	bad := ep(1, 1, 1)
	bad["seasonNumber"] = 1.5
	if _, err = ProjectImportCandidates(q, []map[string]any{bad}, files); err == nil {
		t.Fatal("fractional season")
	}
	plan.Reads = []string{"create"}
	if plan.Validate(q) == nil {
		t.Fatal("write operation")
	}
	plan = ImportPlan{Version: 1, Mode: "deferred", Limit: 5000}
	if (ImportResult{Slots: []int{0}}).Validate(query, plan) == nil {
		t.Fatal("deferred assigned")
	}
	if _, err = ProjectImportCandidates(q, nil, make([]map[string]any, 5001)); err == nil {
		t.Fatal("unbounded files")
	}
}
