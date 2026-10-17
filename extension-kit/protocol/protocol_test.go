package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeAndVersionBoundaries(t *testing.T) {
	valid := `{"protocol":1,"id":"x","capability":"metadata.search.v1","operation":"search","payload":{"newOptionalField":true}}`
	var r Request
	if err := DecodeOne([]byte(valid), &r); err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{valid + ` {}`, strings.Replace(valid, `"protocol":1`, `"protocol":2`, 1), strings.Replace(valid, `"operation":"search"`, `"operation":"<script>"`, 1), strings.Replace(valid, `"payload":{"newOptionalField":true}`, `"payload":[]`, 1), strings.Replace(valid, `"id":"x"`, `"id":"x","shell":"delete"`, 1)} {
		var bad Request
		if err := DecodeOne([]byte(raw), &bad); err == nil && bad.Validate() == nil {
			t.Fatal("invalid envelope accepted", raw)
		}
	}
	for _, response := range []Response{{Protocol: 1, ID: "other", Result: json.RawMessage(`{}`)}, {Protocol: 1, ID: "x", Result: json.RawMessage(`null`)}, {Protocol: 1, ID: "x", Result: json.RawMessage(`{}`), Error: &Failure{Code: "failure", Message: "test"}}} {
		if response.Validate(r) == nil {
			t.Fatal("invalid reply accepted")
		}
	}
	// New named capabilities need no change in the transport itself. The host
	// must still implement/authorize that capability before using its results.
	d := Descriptor{ID: "atelier.custom", Version: "1.0.0", Capabilities: []string{"atelier.waveforms.v1", "metadata.search.v1"}}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	d.Capabilities = append(d.Capabilities, d.Capabilities[0])
	if d.Validate() == nil {
		t.Fatal("duplicate capability accepted")
	}
}
