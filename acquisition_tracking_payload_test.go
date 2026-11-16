package main

import (
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"reflect"
	"testing"
)

func TestAcquisitionTrackingPayloadRoundTrip(t *testing.T) {
	zero := 0
	for _, q := range []a.Tracking{
		{Connector: "radarr", RemoteID: 42, Season: -1, HasFile: true, Monitored: true, Uncertain: true, Local: 1},
		{Connector: "sonarr", RemoteID: 42, Season: 0, Monitored: true, Local: 1, Seasons: []a.Season{{Number: 0, Monitored: true}}, Episodes: []a.Episode{{ID: 1, Season: 0, Number: 1, HasFile: true}, {ID: 2, Season: 0, Number: 2}}, Queue: []a.QueueItem{{RemoteID: 42, EpisodeID: 1, Season: &zero, Size: 10, Remaining: 2, Status: "warning"}, {RemoteID: 42}}},
		{Episodes: []a.Episode{}, Queue: []a.QueueItem{}},
	} {
		data, e := json.Marshal(acquisitionTrackingPayload(q))
		if e != nil {
			t.Fatal(e)
		}
		var decoded a.Tracking
		if e = json.Unmarshal(data, &decoded); e != nil || !reflect.DeepEqual(q, decoded) {
			t.Fatal("wire observations changed", q, decoded, e)
		}
	}
}
