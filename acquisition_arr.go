package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// Each acquisition adapter owns its remote API. Metadata providers only resolve
// stable identifiers; neither component needs to know about the other's storage.
type AcquisitionIdentifierProvider interface {
	AcquisitionID(context.Context, string, string, string) (int, error)
}

func (p *tmdbProvider) AcquisitionID(ctx context.Context, secret, kind, id string) (int, error) {
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return 0, errors.New("Identifiant TMDB invalide.")
	}
	if kind == "films" {
		return n, nil
	}
	return providerResult(ctx, p, func(ctx context.Context) (int, error) {
		if e := p.moduleReady(); e != nil {
			return 0, e
		}
		var out struct {
			TVDB int `json:"tvdb_id"`
		}
		if err = p.get(ctx, secret, "/tv/"+id+"/external_ids", nil, &out); err != nil {
			return 0, err
		}
		if out.TVDB <= 0 {
			return 0, errors.New("TMDB ne fournit pas d’identifiant TVDB pour cette série. Aucune demande envoyée.")
		}
		return out.TVDB, nil
	})
}

type arrConfig struct {
	owner        *catalog
	planChecksum string
	ID           string `json:"id"`
	URL          string `json:"url"`
	Key          string `json:"key,omitempty"`
	Configured   bool   `json:"configured"`
	Enabled      bool   `json:"enabled"`
	Profile      int    `json:"profile"`
	Root         string `json:"root"`
	Library      string `json:"library"`
}

func arrKind(id string) string {
	if id == "radarr" {
		return "films"
	}
	if id == "sonarr" {
		return "series"
	}
	return ""
}
func cleanArrURL(raw string) (string, error) {
	u, e := url.Parse(strings.TrimSpace(raw))
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("Indiquez l’URL HTTP ou HTTPS du service, sans clé, identifiants ni paramètres.")
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/api/v3")
	u.RawPath = ""
	return strings.TrimRight(u.String(), "/"), nil
}

type arrClient struct{ config arrConfig }
type arrObject map[string]any

func number(v any) int {
	if n, ok := v.(json.Number); ok {
		i, _ := n.Int64()
		return int(i)
	}
	n, _ := v.(float64)
	return int(n)
}
func arrFloat(v any) float64 {
	if n, ok := v.(json.Number); ok {
		f, _ := n.Float64()
		return f
	}
	n, _ := v.(float64)
	return n
}
func truth(v any) bool  { b, _ := v.(bool); return b }
func word(v any) string { s, _ := v.(string); return s }

type arrOption struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
type arrOptions struct {
	Profiles []arrOption `json:"profiles"`
	Roots    []arrOption `json:"roots"`
}

func (a arrClient) options(ctx context.Context) (arrOptions, error) {
	ctx, done, err := a.openSession(ctx)
	if err != nil {
		return arrOptions{}, err
	}
	defer done()
	var status struct {
		AppName string `json:"appName"`
	}
	var result arrOptions
	if err := a.exchange(ctx, "status", 0, 0, nil, &status); err != nil {
		return result, err
	}
	if !strings.EqualFold(status.AppName, a.config.ID) {
		return result, errors.New("Ce service ne correspond pas au connecteur choisi.")
	}
	if err := a.exchange(ctx, "profiles", 0, 0, nil, &result.Profiles); err != nil {
		return result, err
	}
	err = a.exchange(ctx, "roots", 0, 0, nil, &result.Roots)
	return result, err
}
func (a arrClient) find(ctx context.Context, id int) (arrObject, error) {
	var all []arrObject
	err := a.exchange(ctx, "find", id, 0, nil, &all)
	if err != nil {
		return nil, err
	}
	for _, v := range all {
		if number(v["externalId"]) == id {
			return v, nil
		}
	}
	return nil, nil
}
func (a arrClient) lookup(ctx context.Context, id int) (arrObject, error) {
	var all []arrObject
	if err := a.exchange(ctx, "lookup", id, 0, nil, &all); err != nil {
		return nil, err
	}
	for _, v := range all {
		if number(v["externalId"]) == id {
			return v, nil
		}
	}
	return nil, errors.New("Le service ne connaît pas cet identifiant. Aucune demande envoyée.")
}
func selectedSeasons(obj arrObject, season int) []int {
	result := []int{}
	seasons, _ := obj["seasons"].([]any)
	for _, s := range seasons {
		v, ok := s.(map[string]any)
		if ok && (season < 0 || number(v["seasonNumber"]) == season) {
			result = append(result, number(v["seasonNumber"]))
		}
	}
	return result
}

// Never PUT a reduced resource: preserve existing paths, quality profiles and
// all unrelated season flags. Explicit episode monitoring covers seasons that
// were already monitored but contained manually unmonitored episodes.
func (a arrClient) submit(ctx context.Context, obj arrObject, season int) (int, error) {
	ctx, done, err := a.openSession(ctx)
	if err != nil {
		return 0, err
	}
	defer done()
	plan, err := a.submissionPlan(ctx, obj, season)
	id := number(obj["id"])
	if err != nil {
		return id, err
	}
	if plan.Write == "none" {
		return id, nil
	}
	fresh := plan.Write == "create"
	obj["monitored"] = true
	if fresh {
		obj["qualityProfileId"] = a.config.Profile
		obj["rootFolderPath"] = a.config.Root
		delete(obj, "path")
		if a.config.ID == "radarr" {
			obj["minimumAvailability"] = plan.MinimumAvailability
			obj["addOptions"] = map[string]any{"searchForMovie": plan.SearchOnAdd}
		} else {
			obj["seasonFolder"] = plan.SeasonFolder
			if plan.SeriesType != "" {
				obj["seriesType"] = plan.SeriesType
			}
			obj["addOptions"] = map[string]any{"monitor": "skip", "searchForMissingEpisodes": plan.SearchOnAdd}
		}
	}
	if a.config.ID == "sonarr" {
		seasons, ok := obj["seasons"].([]any)
		if !ok {
			return id, errors.New("Saisons absentes de la réponse du service.")
		}
		for _, raw := range seasons {
			current, ok := raw.(map[string]any)
			if !ok {
				return id, errors.New("Saison invalide dans la réponse du service.")
			}
			if plan.ResetSeasons {
				current["monitored"] = false
			}
			if slices.Contains(plan.Seasons, number(current["seasonNumber"])) {
				current["monitored"] = true
			}
		}
		if plan.MonitorNewItems != "" {
			obj["monitorNewItems"] = plan.MonitorNewItems
		}
	}
	if fresh {
		var saved arrObject
		err := a.exchange(ctx, "create", 0, 0, obj, &saved)
		if err != nil {
			return 0, err
		}
		id = number(saved["id"])
		if id <= 0 {
			return 0, errors.New("Le service n’a pas confirmé l’identifiant ajouté.")
		}
		return id, nil
	}
	if err := a.exchange(ctx, "update", id, 0, obj, nil); err != nil {
		return id, err
	}
	cmd := map[string]any{"name": "movie", "movieIds": []int{id}}
	if plan.MonitorEpisodes {
		var episodes []arrObject
		if err := a.exchange(ctx, "episodes", id, 0, nil, &episodes); err != nil {
			return id, err
		}
		ids := []int{}
		for _, ep := range episodes {
			if slices.Contains(plan.Seasons, number(ep["seasonNumber"])) {
				ids = append(ids, number(ep["id"]))
			}
		}
		if len(ids) > 0 {
			if err := a.exchange(ctx, "monitor", 0, 0, map[string]any{"episodeIds": ids, "monitored": true}, nil); err != nil {
				return id, err
			}
		}
	}
	if plan.Search == "series" || plan.Search == "season" {
		cmd = map[string]any{"name": "series", "seriesId": id}
		if season >= 0 {
			cmd["name"] = "season"
			cmd["seasonNumber"] = season
		}
	}
	if plan.Search == "" {
		return id, nil
	}
	return id, a.exchange(ctx, "search", 0, 0, cmd, nil)
}
