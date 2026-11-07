package main

import a "mosaune/extensionkit/capabilities/acquisition/v1"

func workflow(q a.Query) (a.Workflow, error) {
	p, err := plan(q)
	if err != nil {
		return a.Workflow{}, err
	}
	w := a.Workflow{Version: 1, Plan: p, Actions: []a.Action{}}
	if p.Write == "none" {
		return w, w.Validate(q)
	}
	fields := map[string]a.Value{"monitored": a.Constant(true)}
	write := a.Action{Operation: p.Write, Fields: fields}
	if p.Write == "create" {
		fields["qualityProfileId"] = a.Reference("destination.profile")
		fields["rootFolderPath"] = a.Reference("destination.root")
		write.Drop = []string{"path"}
		if q.Connector == "radarr" {
			fields["minimumAvailability"] = a.Constant(p.MinimumAvailability)
			fields["addOptions"] = a.Constant(map[string]any{"searchForMovie": p.SearchOnAdd})
		} else {
			fields["seasonFolder"] = a.Constant(p.SeasonFolder)
			if p.SeriesType != "" {
				fields["seriesType"] = a.Constant(p.SeriesType)
			}
			fields["addOptions"] = a.Constant(map[string]any{"monitor": "skip", "searchForMissingEpisodes": p.SearchOnAdd})
		}
	}
	if q.Connector == "sonarr" {
		write.Seasons = true
		if p.MonitorNewItems != "" {
			fields["monitorNewItems"] = a.Constant(p.MonitorNewItems)
		}
	}
	w.Actions = append(w.Actions, write)
	if p.MonitorEpisodes {
		w.Actions = append(w.Actions, a.Action{Operation: "monitor", Fields: map[string]a.Value{"episodeIds": a.Reference("selected.episodes"), "monitored": a.Constant(true)}})
	}
	if p.Search != "" {
		fields = map[string]a.Value{"name": a.Constant(p.Search)}
		if p.Search == "movie" {
			fields["movieIds"] = a.Reference("resource.ids")
		} else {
			fields["seriesId"] = a.Reference("resource.id")
		}
		if p.Search == "season" {
			fields["seasonNumber"] = a.Reference("requested.season")
		}
		w.Actions = append(w.Actions, a.Action{Operation: "search", Fields: fields})
	}
	return w, w.Validate(q)
}
