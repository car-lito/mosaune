// Package acquisition describes a bounded submission plan. Credentials, remote
// objects, HTTP and persistence belong to the host, outside the module.
package acquisition

import (
	"errors"
	"slices"
)

const Capability = "acquisition.plan.v1"

type Query struct {
	Connector  string `json:"connector"`
	RemoteID   int    `json:"remoteId"`
	HasFile    bool   `json:"hasFile"`
	Season     int    `json:"season"`
	Seasons    []int  `json:"seasons"`
	SeriesType string `json:"seriesType"`
}

func (q Query) Validate() error {
	if (q.Connector != "radarr" && q.Connector != "sonarr") || q.RemoteID < 0 || q.Season < -1 || q.Season > 1000 || len(q.Seasons) > 1001 || len(q.SeriesType) > 64 {
		return errors.New("invalid acquisition query")
	}
	if q.Connector == "radarr" && (q.Season != -1 || len(q.Seasons) != 0) {
		return errors.New("invalid film scope")
	}
	seen := map[int]bool{}
	for _, season := range q.Seasons {
		if season < 0 || season > 1000 || seen[season] {
			return errors.New("invalid seasons")
		}
		seen[season] = true
	}
	return nil
}

type Plan struct {
	Write               string `json:"write"`
	MinimumAvailability string `json:"minimumAvailability,omitempty"`
	SeriesType          string `json:"seriesType,omitempty"`
	SeasonFolder        bool   `json:"seasonFolder"`
	MonitorNewItems     string `json:"monitorNewItems,omitempty"`
	ResetSeasons        bool   `json:"resetSeasons"`
	Seasons             []int  `json:"seasons"`
	SearchOnAdd         bool   `json:"searchOnAdd"`
	MonitorEpisodes     bool   `json:"monitorEpisodes"`
	Search              string `json:"search,omitempty"`
}

// Validate bounds the author's choices to the user's confirmed resource and
// season scope. A plan cannot change an existing path or quality profile.
func (p Plan) Validate(q Query) error {
	bad := errors.New("invalid acquisition plan")
	if q.Validate() != nil || !slices.Contains([]string{"none", "create", "update"}, p.Write) || (q.RemoteID == 0) != (p.Write == "create") || len(p.Seasons) > len(q.Seasons) {
		return bad
	}
	if p.Write == "none" && (p.Search != "" || p.MonitorEpisodes || p.SearchOnAdd) {
		return bad
	}
	if q.RemoteID != 0 && (p.ResetSeasons || p.SearchOnAdd || p.MinimumAvailability != "" || p.SeriesType != "" || p.SeasonFolder) {
		return bad
	}
	if q.RemoteID == 0 && (p.Search != "" || p.MonitorEpisodes) {
		return bad
	}
	if q.Connector == "radarr" {
		if len(p.Seasons) != 0 || p.ResetSeasons || p.MonitorEpisodes || p.MonitorNewItems != "" || p.SeriesType != "" || p.SeasonFolder || (p.Search != "" && p.Search != "movie") {
			return bad
		}
		if q.RemoteID == 0 && !slices.Contains([]string{"announced", "inCinemas", "released"}, p.MinimumAvailability) {
			return bad
		}
		return nil
	}
	if p.MinimumAvailability != "" || (p.SeriesType != "" && !slices.Contains([]string{"standard", "daily", "anime"}, p.SeriesType)) || !slices.Contains([]string{"", "none", "all"}, p.MonitorNewItems) || (q.Season >= 0 && p.MonitorNewItems == "all") {
		return bad
	}
	// A lookup response may already contain enabled season flags or a future
	// monitoring policy. A new series must explicitly clear those inherited flags.
	if q.RemoteID == 0 && (!p.ResetSeasons || p.MonitorNewItems == "") {
		return bad
	}
	if p.Search != "" && (q.Season < 0 && p.Search != "series" || q.Season >= 0 && p.Search != "season") {
		return bad
	}
	seen := map[int]bool{}
	for _, season := range p.Seasons {
		if seen[season] || !slices.Contains(q.Seasons, season) || (q.Season >= 0 && q.Season != season) {
			return bad
		}
		seen[season] = true
	}
	if p.Write != "none" && len(p.Seasons) == 0 {
		return bad
	}
	return nil
}
