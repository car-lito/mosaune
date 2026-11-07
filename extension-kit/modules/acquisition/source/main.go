package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
)

//go:embed policy.json
var policyJSON []byte

func plan(q a.Query) (a.Plan, error) {
	var policy struct {
		SearchMode string `json:"searchMode"`
	}
	if json.Unmarshal(policyJSON, &policy) != nil || (policy.SearchMode != "auto" && policy.SearchMode != "none") {
		return a.Plan{}, errors.New("invalid submission policy")
	}
	search := policy.SearchMode == "auto"
	fresh := q.RemoteID == 0
	p := a.Plan{Write: "update", Seasons: []int{}}
	if fresh {
		p.Write = "create"
		p.SearchOnAdd = search
	}
	if q.Connector == "radarr" {
		if fresh {
			p.MinimumAvailability = "released"
		} else if q.HasFile {
			p.Write = "none"
		} else if search {
			p.Search = "movie"
		}
		return p, p.Validate(q)
	}
	for _, season := range q.Seasons {
		if q.Season < 0 || q.Season == season {
			p.Seasons = append(p.Seasons, season)
		}
	}
	if fresh {
		p.ResetSeasons = true
		p.SeasonFolder = true
		if q.SeriesType == "" {
			p.SeriesType = "standard"
		}
		p.MonitorNewItems = "none"
	} else {
		p.MonitorEpisodes = true
		if search {
			p.Search = "season"
			if q.Season < 0 {
				p.Search = "series"
			}
		}
	}
	if q.Season < 0 {
		p.MonitorNewItems = "all"
	}
	return p, p.Validate(q)
}

func main() {
	if err := sdk.ServeSized(os.Stdin, os.Stdout, 1<<20, func(r protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_acquisition_plan", Message: "Demande ou saisons incompatibles avec le connecteur."}
		}
		if r.Capability != a.Capability {
			return fail()
		}
		if r.Operation == "describe-wire" {
			var q a.TransportQuery
			if protocol.DecodeOne(r.Payload, &q) != nil {
				return fail()
			}
			format, err := wireFormat(q.Connector)
			if err != nil {
				return fail()
			}
			return format, nil
		}
		if r.Operation == "describe-settings" {
			var q a.TransportQuery
			if protocol.DecodeOne(r.Payload, &q) != nil {
				return fail()
			}
			form, err := settings(q.Connector)
			if err != nil {
				return fail()
			}
			return form, nil
		}
		if r.Operation == "describe-transport" {
			var q a.TransportQuery
			if protocol.DecodeOne(r.Payload, &q) != nil {
				return fail()
			}
			p, err := transport(q.Connector)
			if err != nil {
				return fail()
			}
			return p, nil
		}
		if r.Operation == "describe" || r.Operation == "tracking-info" || r.Operation == "workflow-info" {
			var empty struct{}
			if protocol.DecodeOne(r.Payload, &empty) != nil {
				return fail()
			}
			return struct{ Version int }{1}, nil
		}
		if r.Operation == "track" {
			var q a.Tracking
			if protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil {
				return fail()
			}
			out, err := track(q)
			if err != nil {
				return fail()
			}
			return out, nil
		}
		var q a.Query
		if (r.Operation != "plan" && r.Operation != "submission-workflow") || protocol.DecodeOne(r.Payload, &q) != nil || q.Validate() != nil {
			return fail()
		}
		if r.Operation == "submission-workflow" {
			w, err := workflow(q)
			if err != nil {
				return fail()
			}
			return w, nil
		}
		p, err := plan(q)
		if err != nil {
			return fail()
		}
		return p, nil
	}); err != nil {
		os.Exit(1)
	}
}
