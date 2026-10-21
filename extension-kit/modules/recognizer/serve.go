// Package recognizer is optional SDK glue, shared by the official media modules.
package recognizer

import (
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func Serve(formats recognition.Formats, identify func(string) recognition.Item) {
	handler := func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_recognition", Message: "Reconnaissance indisponible."}
		}
		if r.Capability != recognition.Capability {
			return fail()
		}
		switch r.Operation {
		case "formats":
			var empty struct{}
			if protocol.DecodeOne(r.Payload, &empty) != nil {
				return fail()
			}
			return formats, nil
		case "recognize":
			var q recognition.Query
			if protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil {
				return fail()
			}
			out := recognition.Result{Items: make([]recognition.Item, len(q.Paths))}
			for i, p := range q.Paths {
				if formats.Accepts(p) {
					out.Items[i] = identify(p)
					out.Items[i].Accepted = true
				}
			}
			if out.Validate(q, formats) != nil {
				return fail()
			}
			return out, nil
		}
		return fail()
	}
	if err := sdk.Serve(os.Stdin, os.Stdout, handler); err != nil {
		os.Exit(1)
	}
}
