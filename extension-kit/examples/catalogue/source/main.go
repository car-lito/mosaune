package main

import (
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func main() {
	if err := sdk.Serve(os.Stdin, os.Stdout, func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_catalogue", Message: "Requête invalide."}
		}
		if r.Capability != catalogue.Capability {
			return fail()
		}
		if r.Operation == "describe" {
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
			out.Items[i] = catalogue.Item{Title: f.Name, Subtitle: "Exemple contributeur", SortText: f.Name}
		}
		if out.Validate(q) != nil {
			return fail()
		}
		return out, nil
	}); err != nil {
		os.Exit(1)
	}
}
