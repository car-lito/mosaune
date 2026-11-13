package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed availability.json
var availabilityJSON []byte

func localAvailability(q a.AvailabilityQuery) (a.AvailabilityResult, error) {
	var policy struct {
		Mode string `json:"mode"`
	}
	out, err := q.Eligible()
	if err != nil {
		return out, err
	}
	if json.Unmarshal(availabilityJSON, &policy) != nil || (policy.Mode != "verified" && policy.Mode != "deferred") {
		return out, errors.New("invalid availability policy")
	}
	if policy.Mode == "deferred" {
		out.Film = false
		out.Episodes = []a.EpisodeKey{}
	}
	return out, out.Validate(q)
}
