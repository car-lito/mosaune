package main

import (
	"context"
	"encoding/json"
	"errors"
	cfg "mosaune/extensionkit/acquisitionconfig"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
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
	revision     int64
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
func cleanArrURL(raw string) (string, error) { return cfg.NormalizeAddress(raw) }

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

// The installed module supplies the entire bounded sequence. The host only
// resolves private bindings and executes validated operations, stopping on error.
func (client arrClient) submit(ctx context.Context, obj arrObject, season int) (int, error) {
	ctx, done, err := client.openSession(ctx)
	if err != nil {
		return 0, err
	}
	defer done()
	workflow, err := client.submissionWorkflow(ctx, obj, season)
	id := number(obj["id"])
	if err != nil {
		return id, err
	}
	bindings := a.Bindings{ID: id, Season: season, Profile: client.config.Profile, Root: client.config.Root}
	for _, action := range workflow.Actions {
		switch action.Operation {
		case "create", "update":
			resource, err := action.Resource(map[string]any(obj), workflow.Plan, bindings)
			if err != nil {
				return id, err
			}
			if action.Operation == "create" {
				var saved arrObject
				if err = client.exchange(ctx, "create", 0, 0, resource, &saved); err != nil {
					return 0, err
				}
				id = number(saved["id"])
				if id <= 0 {
					return 0, errors.New("Le service n’a pas confirmé l’identifiant ajouté.")
				}
				bindings.ID = id
			} else if err = client.exchange(ctx, "update", id, 0, resource, nil); err != nil {
				return id, err
			}
		case "monitor":
			var episodes []arrObject
			if err = client.exchange(ctx, "episodes", id, 0, nil, &episodes); err != nil {
				return id, err
			}
			records := make([]map[string]any, 0, len(episodes))
			for _, ep := range episodes {
				records = append(records, map[string]any(ep))
			}
			bindings.Episodes, err = a.SelectedEpisodes(records, workflow.Plan.Seasons)
			if err != nil {
				return id, err
			}
			if len(bindings.Episodes) == 0 {
				continue
			}
			body, err := action.Resolve(bindings)
			if err != nil {
				return id, err
			}
			if err = client.exchange(ctx, "monitor", 0, 0, body, nil); err != nil {
				return id, err
			}
		case "search":
			body, err := action.Resolve(bindings)
			if err != nil {
				return id, err
			}
			if err = client.exchange(ctx, "search", 0, 0, body, nil); err != nil {
				return id, err
			}
		default:
			return id, errors.New("Opération du programme d’envoi inconnue.")
		}
	}
	return id, nil
}
