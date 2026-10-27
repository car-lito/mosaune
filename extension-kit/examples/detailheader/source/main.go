package main

import (
	d "mosaune/extensionkit/capabilities/detailheader/v1"

	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

func main() {
	if err := sdk.Serve(os.Stdin, os.Stdout, func(r protocol.Request) (any, *protocol.Failure) {
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
		title := q.Title
		if !q.Track && !q.StoredWork && q.GroupTitle != "" {
			title = q.GroupTitle
		}
		if title == "" {
			title = q.GroupTitle
		}
		if title == "" {
			title = q.Name
		}
		if title == "" {
			title = "Média"
		}
		out := d.Result{Title: title, Eyebrow: "DANS LE CADRE", Facts: []d.Fact{}, Tagline: "Une autre façon de présenter votre collection."}
		if out.Validate(q) != nil {
			return fail()
		}
		return out, nil
	}); err != nil {
		os.Exit(1)
	}
}
