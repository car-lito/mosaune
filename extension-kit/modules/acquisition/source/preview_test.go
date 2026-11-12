package main

import (
	"bytes"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"testing"
)

func TestPreviewPolicy(t *testing.T) {
	original := previewJSON
	defer func() { previewJSON = original }()
	previewJSON = bytes.ReplaceAll(previewJSON, []byte(`"missing":"lookup"`), []byte(`"missing":"reject"`))
	for _, q := range []a.PreviewQuery{{Kind: "films", Season: -1}, {Kind: "series", Season: 1}} {
		p, err := previewPlan(q)
		if err != nil || p.Missing != "reject" {
			t.Fatal(p, err)
		}
	}
	for _, q := range []a.PreviewQuery{{Kind: "films", Season: 1}, {Kind: "books", Season: -1}, {Kind: "series", Season: 1001}} {
		if _, err := previewPlan(q); err == nil {
			t.Fatal("invalid scope accepted", q)
		}
	}
}
