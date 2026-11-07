package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed observations.json
var observationsJSON []byte

func observationPlan(q a.ObservationQuery) (a.ObservationPlan, error) {
	var plans map[string]a.ObservationPlan
	if err := json.Unmarshal(observationsJSON, &plans); err != nil {
		return a.ObservationPlan{}, err
	}
	p, ok := plans[q.Connector]
	if !ok {
		return p, errors.New("unknown connector")
	}
	if q.RemoteID == 0 {
		p.Seasons = false
		p.Reads = []a.ObservationRead{}
	}
	return p, p.Validate(q)
}
