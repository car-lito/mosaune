package main

import (
	d "mosaune/extensionkit/capabilities/detailexperience/v1"

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
		q, err := d.DecodeQuery(r.Payload)
		if r.Operation != "project" || err != nil {
			return fail()
		}
		out := d.Result{Cover: "square", Actions: []d.Action{}, People: []int{}, Sections: []d.Section{}}
		s := d.Section{Key: "plain", Title: "Lecture directe", Rows: []d.Row{}, Actions: []d.Action{}}
		for _, f := range q.Files {
			if q.Focus != "" && q.Focus != f.ID {
				continue
			}
			s.Rows = append(s.Rows, d.Row{Key: f.ID, FileID: f.ID, Title: f.Name, Actions: []d.Action{{Operation: "play", Label: "Écouter ou regarder", FileID: f.ID}}})
		}
		out.Sections = append(out.Sections, s)
		// The host validates the result once; avoid serializing large lists twice in WASI.
		return out, nil
	}); err != nil {
		os.Exit(1)
	}
}
