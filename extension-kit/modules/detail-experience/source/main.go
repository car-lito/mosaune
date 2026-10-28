package main

import (
	d "mosaune/extensionkit/capabilities/detailexperience/v1"
	"mosaune/extensionkit/modules/detailexperience"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func main() {
	if err := sdk.ServeSized(os.Stdin, os.Stdout, 1<<20, func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_detail", Message: "Fiche indisponible."}
		}
		if r.Capability != d.Capability {
			return fail()
		}
		if r.Operation == "describe" {
			var empty struct{}
			if protocol.DecodeOne(r.Payload, &empty) != nil {
				return fail()
			}
			return struct {
				Version int `json:"version"`
			}{1}, nil
		}
		var q d.Query
		if r.Operation != "project" || protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil {
			return fail()
		}
		out := detailexperience.Project(q)
		// The host validates the result once; avoid serializing large lists twice in WASI.
		return out, nil
	}); err != nil {
		os.Exit(1)
	}
}
