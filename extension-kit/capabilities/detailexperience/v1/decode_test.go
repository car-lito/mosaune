package detailexperience

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeQueryByteLimitAndFields(t *testing.T) {
	q := Query{Kind: "music", Files: []File{{ID: "track", Title: "Été"}}}
	raw, _ := json.Marshal(q)
	if _, e := DecodeQuery(raw); e != nil {
		t.Fatal(e)
	}
	if _, e := DecodeQuery(append(raw, []byte(strings.Repeat(" ", 512<<10))...)); e == nil {
		t.Fatal("oversized input accepted")
	}
	if _, e := DecodeQuery([]byte(`{"kind":"music","files":[{"id":"x"},{"id":"x"}]}`)); e == nil {
		t.Fatal("duplicate file accepted")
	}
	if _, e := DecodeQuery([]byte(`{"kind":"music","unexpected":true}`)); e == nil {
		t.Fatal("unknown field accepted")
	}
}
