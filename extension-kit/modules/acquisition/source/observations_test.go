package main

import (
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"testing"
)

func TestObservationPolicyReplacement(t *testing.T) {
	original := observationsJSON
	defer func() { observationsJSON = original }()
	observationsJSON = []byte(`{"sonarr":{"version":1,"seasons":true,"reads":[{"operation":"queue","limit":2},{"operation":"episodes","limit":30}]}}`)
	p, err := observationPlan(a.ObservationQuery{Connector: "sonarr", RemoteID: 12})
	if err != nil || p.Reads[0].Operation != "queue" || p.Reads[1].Limit != 30 {
		t.Fatal(p, err)
	}
	p, err = observationPlan(a.ObservationQuery{Connector: "sonarr"})
	if err != nil || len(p.Reads) != 0 || p.Seasons {
		t.Fatal(p, err)
	}
	if _, err = observationPlan(a.ObservationQuery{Connector: "unknown"}); err == nil {
		t.Fatal("unknown connector")
	}
}
