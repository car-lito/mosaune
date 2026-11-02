package main

import (
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"mosaune/extensionkit/modules/musicresults"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func main() {
	if e := sdk.ServeSized(os.Stdin, os.Stdout, 1<<20, func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_music_results", Message: "Réponse musicale invalide ou trop volumineuse."}
		}
		if r.Capability != d.Capability {
			return fail()
		}
		if r.Operation == "describe" {
			var empty struct{}
			if protocol.DecodeOne(r.Payload, &empty) != nil {
				return fail()
			}
			return struct{ Version int }{1}, nil
		}
		var q d.Query
		if r.Operation != "decode" || protocol.DecodeOne(r.Payload, &q) != nil {
			return fail()
		}
		out, e := musicresults.Decode(q)
		if e != nil || out.Validate(q) != nil {
			return fail()
		}
		return out, nil
	}); e != nil {
		os.Exit(1)
	}
}
