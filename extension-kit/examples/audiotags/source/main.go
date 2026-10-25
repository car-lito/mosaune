package main

import (
	"encoding/json"
	tags "mosaune/extensionkit/capabilities/audiotags/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
	"strings"
)

func main() {
	if err := sdk.Serve(os.Stdin, os.Stdout, func(q protocol.Request) (any, *protocol.Failure) {
		var in tags.Query
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_request", Message: "Lecture ID3v1 impossible."}
		}
		if q.Capability != tags.Capability || q.Operation != "extract" || protocol.DecodeOne(q.Payload, &in) != nil || in.Size < 0 {
			return fail()
		}
		if in.Size < 128 {
			return tags.Result{}, nil
		}
		raw, err := sdk.ReadServiceSized(tags.Service, "range", tags.Range{Offset: in.Size - 128, Length: 128}, 1024)
		var block tags.Block
		if err != nil || json.Unmarshal(raw, &block) != nil || len(block.Data) != 128 {
			return fail()
		}
		b := block.Data
		if string(b[:3]) != "TAG" {
			return tags.Result{}, nil
		}
		// ID3v1 text is ISO-8859-1, not UTF-8. Preserve accented letters.
		text := func(b []byte) string {
			r := make([]rune, len(b))
			for i, v := range b {
				r[i] = rune(v)
			}
			return strings.Trim(string(r), " \x00")
		}
		return tags.Result{Matched: true, Title: text(b[3:33]), Artist: text(b[33:63]), Album: text(b[63:93])}, nil
	}); err != nil {
		os.Exit(1)
	}
}
