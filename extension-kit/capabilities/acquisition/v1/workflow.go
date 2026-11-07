package acquisition

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
)

// A workflow operates on symbolic host bindings, never on a key, destination
// or full remote object supplied to the module.
type Value struct {
	Ref     string          `json:"ref,omitempty"`
	Literal json.RawMessage `json:"literal,omitempty"`
}

func Constant(v any) Value        { b, _ := json.Marshal(v); return Value{Literal: b} }
func Reference(name string) Value { return Value{Ref: name} }

type Action struct {
	Operation string           `json:"operation"`
	Fields    map[string]Value `json:"fields,omitempty"`
	Drop      []string         `json:"drop,omitempty"`
	Seasons   bool             `json:"seasons,omitempty"`
}
type Workflow struct {
	Version int      `json:"version"`
	Plan    Plan     `json:"plan"`
	Actions []Action `json:"actions"`
}

func sameValue(v Value, w Value) bool {
	if v.Ref != "" {
		return v.Ref == w.Ref && len(v.Literal) == 0
	}
	if w.Ref != "" || len(v.Literal) == 0 {
		return false
	}
	var a, b any
	return json.Unmarshal(v.Literal, &a) == nil && json.Unmarshal(w.Literal, &b) == nil && reflect.DeepEqual(a, b)
}

// This is an authorization contract, not a fallback workflow. The extension
// must supply every action and binding; extra fields and writes are rejected.
func (w Workflow) Validate(q Query) error {
	bad := errors.New("workflow exceeds confirmed acquisition scope")
	p := w.Plan
	if w.Version != 1 || p.Validate(q) != nil || len(w.Actions) > 3 {
		return bad
	}
	ops := []string{}
	if p.Write != "none" {
		ops = append(ops, p.Write)
		if p.MonitorEpisodes {
			ops = append(ops, "monitor")
		}
		if p.Search != "" {
			ops = append(ops, "search")
		}
	}
	if len(ops) != len(w.Actions) {
		return bad
	}
	for i, action := range w.Actions {
		if action.Operation != ops[i] {
			return bad
		}
		expected := map[string]Value{}
		drop := []string{}
		seasons := false
		switch action.Operation {
		case "create", "update":
			expected["monitored"] = Constant(true)
			seasons = q.Connector == "sonarr"
			if action.Operation == "create" {
				expected["qualityProfileId"] = Reference("destination.profile")
				expected["rootFolderPath"] = Reference("destination.root")
				drop = append(drop, "path")
				if q.Connector == "radarr" {
					expected["minimumAvailability"] = Constant(p.MinimumAvailability)
					expected["addOptions"] = Constant(map[string]any{"searchForMovie": p.SearchOnAdd})
				} else {
					expected["seasonFolder"] = Constant(p.SeasonFolder)
					if p.SeriesType != "" {
						expected["seriesType"] = Constant(p.SeriesType)
					}
					expected["addOptions"] = Constant(map[string]any{"monitor": "skip", "searchForMissingEpisodes": p.SearchOnAdd})
				}
			}
			if seasons && p.MonitorNewItems != "" {
				expected["monitorNewItems"] = Constant(p.MonitorNewItems)
			}
		case "monitor":
			expected["episodeIds"] = Reference("selected.episodes")
			expected["monitored"] = Constant(true)
		case "search":
			expected["name"] = Constant(p.Search)
			if p.Search == "movie" {
				expected["movieIds"] = Reference("resource.ids")
			} else {
				expected["seriesId"] = Reference("resource.id")
			}
			if p.Search == "season" {
				expected["seasonNumber"] = Reference("requested.season")
			}
		}
		if action.Seasons != seasons || !slices.Equal(action.Drop, drop) || len(action.Fields) != len(expected) {
			return bad
		}
		for name, want := range expected {
			if !sameValue(action.Fields[name], want) {
				return bad
			}
		}
	}
	return nil
}

// Bindings exist only inside the host. Modules return references to these values.
type Bindings struct {
	Profile  int
	Root     string
	ID       int
	Season   int
	Episodes []int
}

func (action Action) Resolve(b Bindings) (map[string]any, error) {
	out := map[string]any{}
	for key, value := range action.Fields {
		var v any
		if value.Ref != "" {
			switch value.Ref {
			case "destination.profile":
				v = b.Profile
			case "destination.root":
				v = b.Root
			case "resource.id":
				v = b.ID
			case "resource.ids":
				v = []int{b.ID}
			case "requested.season":
				v = b.Season
			case "selected.episodes":
				v = b.Episodes
			default:
				return nil, errors.New("unknown acquisition binding")
			}
		} else {
			d := json.NewDecoder(bytes.NewReader(value.Literal))
			d.UseNumber()
			if err := d.Decode(&v); err != nil {
				return nil, err
			}
		}
		out[key] = v
	}
	return out, nil
}

// Preserve all unknown remote fields and unrelated season flags. Work on a
// shallow copy with copied season objects so preparation never mutates input.
func (action Action) Resource(original map[string]any, p Plan, b Bindings) (map[string]any, error) {
	out := make(map[string]any, len(original))
	for k, v := range original {
		out[k] = v
	}
	fields, err := action.Resolve(b)
	if err != nil {
		return nil, err
	}
	for k, v := range fields {
		out[k] = v
	}
	for _, k := range action.Drop {
		delete(out, k)
	}
	if action.Seasons {
		list, ok := original["seasons"].([]any)
		if !ok {
			return nil, errors.New("Saisons absentes de la réponse du service.")
		}
		seasons := make([]any, 0, len(list))
		for _, raw := range list {
			old, ok := raw.(map[string]any)
			if !ok {
				return nil, errors.New("Saison invalide dans la réponse du service.")
			}
			current := map[string]any{}
			for k, v := range old {
				current[k] = v
			}
			n, ok := scopeNumber(old["seasonNumber"])
			if !ok {
				return nil, errors.New("Numéro de saison invalide.")
			}
			if p.ResetSeasons {
				current["monitored"] = false
			}
			if slices.Contains(p.Seasons, n) {
				current["monitored"] = true
			}
			seasons = append(seasons, current)
		}
		out["seasons"] = seasons
	}
	return out, nil
}
func scopeNumber(v any) (int, bool) {
	switch n := v.(type) {
	case json.Number:
		i, e := n.Int64()
		return int(i), e == nil && i >= 0 && int64(int(i)) == i
	case float64:
		i := int(n)
		return i, n >= 0 && float64(i) == n
	case int:
		return n, n >= 0
	}
	return 0, false
}
func SelectedEpisodes(episodes []map[string]any, seasons []int) ([]int, error) {
	if len(episodes) > 5000 {
		return nil, errors.New("Trop d’épisodes dans cette réponse.")
	}
	ids := []int{}
	seen := map[int]bool{}
	for _, ep := range episodes {
		season, ok := scopeNumber(ep["seasonNumber"])
		if !ok {
			return nil, errors.New("Saison d’épisode invalide.")
		}
		if !slices.Contains(seasons, season) {
			continue
		}
		id, ok := scopeNumber(ep["id"])
		if !ok || id == 0 || seen[id] {
			return nil, errors.New("Identifiant d’épisode invalide ou dupliqué.")
		}
		ids = append(ids, id)
		seen[id] = true
	}
	return ids, nil
}
