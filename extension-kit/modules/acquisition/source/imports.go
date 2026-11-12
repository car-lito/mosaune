package main

import (
	_ "embed"
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed imports.json
var importsJSON []byte

func importPlan(q a.PreviewQuery) (a.ImportPlan, error) {
	var p a.ImportPlan
	if err := json.Unmarshal(importsJSON, &p); err != nil {
		return p, err
	}
	p.Reads = []string{}
	if p.Mode == "verified" && q.Kind == "series" {
		p.Reads = append(p.Reads, "episodefiles")
	}
	return p, p.Validate(q)
}
func matchImports(q a.ImportQuery) (a.ImportResult, error) {
	out := a.ImportResult{Slots: []int{}}
	if err := q.Validate(); err != nil {
		return out, err
	}
	p, err := importPlan(q.Scope)
	if err != nil {
		return out, err
	}
	if p.Mode == "verified" {
		for _, c := range q.Candidates {
			if q.Scope.Kind == "films" || q.Scope.Season < 0 || q.Scope.Season == c.Season {
				out.Slots = append(out.Slots, c.Slot)
				if len(out.Slots) == p.Limit {
					break
				}
			}
		}
	}
	return out, out.Validate(q, p)
}
