package main

import (
	_ "embed"
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed preview.json
var previewJSON []byte

func previewPlan(q a.PreviewQuery) (a.PreviewPlan, error) {
	var plans map[string]a.PreviewPlan
	if err := json.Unmarshal(previewJSON, &plans); err != nil {
		return a.PreviewPlan{}, err
	}
	p := plans[q.Kind]
	return p, p.Validate(q)
}
