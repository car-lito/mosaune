package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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
}

type arrConfig struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Key        string `json:"key,omitempty"`
	Configured bool   `json:"configured"`
	Enabled    bool   `json:"enabled"`
	Profile    int    `json:"profile"`
	Root       string `json:"root"`
	Library    string `json:"library"`
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

func number(v any) int  { n, _ := v.(float64); return int(n) }
func truth(v any) bool  { b, _ := v.(bool); return b }
func word(v any) string { s, _ := v.(string); return s }
func (a arrClient) call(ctx context.Context, method, path string, body, out any) error {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return errors.New("Demande invalide.")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, a.config.URL+"/api/v3/"+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("Adresse du service invalide.")
	}
	req.Header.Set("X-Api-Key", a.config.Key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection refusée") }}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("Service inaccessible ou réponse trop lente. Vérifiez son adresse et son accès réseau.")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return errors.New("Le service refuse la clé API.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Le service a refusé l’opération (HTTP %d). Consultez ses événements.", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil || len(b) > 8<<20 {
		return errors.New("Réponse du service incomplète ou trop volumineuse.")
	}
	if out != nil && json.Unmarshal(b, out) != nil {
		return errors.New("Réponse inattendue : vérifiez l’URL du service.")
	}
	return nil
}

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
	var status struct {
		AppName string `json:"appName"`
	}
	var result arrOptions
	if err := a.call(ctx, "GET", "system/status", nil, &status); err != nil {
		return result, err
	}
	if !strings.EqualFold(status.AppName, a.config.ID) {
		return result, errors.New("Ce service ne correspond pas au connecteur choisi.")
	}
	if err := a.call(ctx, "GET", "qualityprofile", nil, &result.Profiles); err != nil {
		return result, err
	}
	err := a.call(ctx, "GET", "rootfolder", nil, &result.Roots)
	return result, err
}
func (a arrClient) resource() string {
	if a.config.ID == "radarr" {
		return "movie"
	}
	return "series"
}
func (a arrClient) identifier() string {
	if a.config.ID == "radarr" {
		return "tmdbId"
	}
	return "tvdbId"
}
func (a arrClient) find(ctx context.Context, id int) (arrObject, error) {
	var all []arrObject
	err := a.call(ctx, "GET", a.resource()+"?"+a.identifier()+"="+strconv.Itoa(id), nil, &all)
	if err != nil {
		return nil, err
	}
	for _, v := range all {
		if number(v[a.identifier()]) == id {
			return v, nil
		}
	}
	return nil, nil
}
func (a arrClient) lookup(ctx context.Context, id int) (arrObject, error) {
	prefix := "tmdb:"
	if a.config.ID == "sonarr" {
		prefix = "tvdb:"
	}
	var all []arrObject
	if err := a.call(ctx, "GET", a.resource()+"/lookup?term="+url.QueryEscape(prefix+strconv.Itoa(id)), nil, &all); err != nil {
		return nil, err
	}
	for _, v := range all {
		if number(v[a.identifier()]) == id {
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
	id := number(obj["id"])
	fresh := id == 0
	obj["monitored"] = true
	if fresh {
		obj["qualityProfileId"] = a.config.Profile
		obj["rootFolderPath"] = a.config.Root
		delete(obj, "path")
	}
	if a.config.ID == "radarr" {
		if fresh {
			obj["minimumAvailability"] = "released"
			obj["addOptions"] = map[string]any{"searchForMovie": true}
		}
	} else {
		seasons, ok := obj["seasons"].([]any)
		if !ok || len(selectedSeasons(obj, season)) == 0 {
			return id, errors.New("Saisons absentes de la réponse du service.")
		}
		for _, s := range seasons {
			v, ok := s.(map[string]any)
			if !ok {
				return id, errors.New("Saison invalide dans la réponse du service.")
			}
			if fresh {
				v["monitored"] = false
			}
			if season < 0 || number(v["seasonNumber"]) == season {
				v["monitored"] = true
			}
		}
		if fresh {
			obj["seasonFolder"] = true
			if word(obj["seriesType"]) == "" {
				obj["seriesType"] = "standard"
			}
			obj["monitorNewItems"] = "none"
			if season < 0 {
				obj["monitorNewItems"] = "all"
			}
			obj["addOptions"] = map[string]any{"monitor": "skip", "searchForMissingEpisodes": true}
		}
		if !fresh && season < 0 {
			obj["monitorNewItems"] = "all"
		}
	}
	if fresh {
		var saved arrObject
		err := a.call(ctx, "POST", a.resource(), obj, &saved)
		if err != nil {
			return 0, err
		}
		id = number(saved["id"])
		if id <= 0 {
			return 0, errors.New("Le service n’a pas confirmé l’identifiant ajouté.")
		}
		return id, nil
	}
	if a.config.ID == "radarr" && truth(obj["hasFile"]) {
		return id, nil
	}
	if err := a.call(ctx, "PUT", a.resource()+"/"+strconv.Itoa(id), obj, nil); err != nil {
		return id, err
	}
	cmd := map[string]any{"name": "MoviesSearch", "movieIds": []int{id}}
	if a.config.ID == "sonarr" {
		var episodes []arrObject
		if err := a.call(ctx, "GET", "episode?seriesId="+strconv.Itoa(id), nil, &episodes); err != nil {
			return id, err
		}
		ids := []int{}
		for _, ep := range episodes {
			if season < 0 || number(ep["seasonNumber"]) == season {
				ids = append(ids, number(ep["id"]))
			}
		}
		if len(ids) > 0 {
			if err := a.call(ctx, "PUT", "episode/monitor", map[string]any{"episodeIds": ids, "monitored": true}, nil); err != nil {
				return id, err
			}
		}
		cmd = map[string]any{"name": "SeriesSearch", "seriesId": id}
		if season >= 0 {
			cmd["name"] = "SeasonSearch"
			cmd["seasonNumber"] = season
		}
	}
	return id, a.call(ctx, "POST", "command", cmd, nil)
}
