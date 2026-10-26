// Package cataloguer is optional SDK glue, not linked into the host.
package cataloguer

import (
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func Serve(project func(catalogue.Facts) catalogue.Item) {
	err := sdk.Serve(os.Stdin, os.Stdout, func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_catalogue", Message: "Projection indisponible."}
		}
		if r.Capability != catalogue.Capability {
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
		var q catalogue.Query
		if r.Operation != "project" || protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil {
			return fail()
		}
		out := catalogue.Result{Items: make([]catalogue.Item, len(q.Items))}
		for i, f := range q.Items {
			out.Items[i] = project(f)
		}
		if out.Validate(q) != nil {
			return fail()
		}
		return out, nil
	})
	if err != nil {
		os.Exit(1)
	}
}
