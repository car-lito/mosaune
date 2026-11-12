package acquisition

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type PreviewQuery struct {
	Kind   string `json:"kind"`
	Season int    `json:"season"`
}

func (q PreviewQuery) Validate() error {
	if (q.Kind != "films" && q.Kind != "series") || q.Season < -1 || q.Season > 1000 || (q.Kind == "films" && q.Season != -1) {
		return errors.New("Œuvre ou saison invalide.")
	}
	return nil
}

// Only a media kind and requested season cross the module boundary. This plan
// names read operations and UI text; it contains no address, secret or SQL.
type PreviewPlan struct {
	Version        int      `json:"version"`
	Connector      string   `json:"connector"`
	Reads          []string `json:"reads"`
	Missing        string   `json:"missing"`
	MissingMessage string   `json:"missingMessage"`
	SeasonMessage  string   `json:"seasonMessage"`
	ProfilePrefix  string   `json:"profilePrefix"`
}

func (p PreviewPlan) Validate(q PreviewQuery) error {
	bad := errors.New("Plan de préparation incompatible.")
	if q.Validate() != nil || p.Version != 1 || len(p.Reads) != 4 || (p.Missing != "lookup" && p.Missing != "reject") {
		return bad
	}
	if (q.Kind == "films" && p.Connector != "radarr") || (q.Kind == "series" && p.Connector != "sonarr") {
		return bad
	}
	if p.Reads[0] != "status" {
		return bad
	}
	seen := map[string]bool{}
	for _, op := range p.Reads {
		if seen[op] {
			return bad
		}
		seen[op] = true
		switch op {
		case "status", "profiles", "roots", "find":
		default:
			return bad
		}
	}
	for _, v := range []string{p.MissingMessage, p.SeasonMessage, p.ProfilePrefix} {
		if !utf8.ValidString(v) || strings.TrimSpace(v) == "" || len(v) > 512 || strings.ContainsAny(v, "\x00\r\n") {
			return bad
		}
	}
	return nil
}

type Choice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
type PreviewDestination struct {
	Root    string
	Profile string
	Seasons []int
}

// Project runs in the trusted host. The module never sees the private values
// resolved here. Existing destinations cannot be replaced by configured defaults.
func (p PreviewPlan) Project(q PreviewQuery, existing bool, configured Bindings, obj map[string]any, profiles []Choice) (PreviewDestination, error) {
	out := PreviewDestination{Root: configured.Root, Seasons: []int{}}
	if err := p.Validate(q); err != nil {
		return out, err
	}
	profile := configured.Profile
	if existing {
		id, ok := scopeNumber(obj["id"])
		if !ok || id <= 0 {
			return out, errors.New("Identifiant distant invalide.")
		}
		out.Root, ok = obj["path"].(string)
		if !ok || out.Root == "" || len(out.Root) > 4096 {
			return out, errors.New("Destination distante absente ou invalide.")
		}
		profile, ok = scopeNumber(obj["qualityProfileId"])
		if !ok || profile <= 0 {
			return out, errors.New("Profil distant absent ou invalide.")
		}
	} else if id, ok := scopeNumber(obj["id"]); obj["id"] != nil && (!ok || id != 0) {
		return out, errors.New("Le résultat de recherche ne correspond pas à une nouvelle ressource.")
	}
	if len(profiles) > 5000 {
		return out, errors.New("Trop de profils dans la réponse du service.")
	}
	out.Profile = p.ProfilePrefix + strconv.Itoa(profile)
	for _, choice := range profiles {
		if choice.ID == profile {
			out.Profile = choice.Name
			break
		}
	}
	if q.Kind == "series" {
		seasons, ok := obj["seasons"].([]any)
		if !ok || len(seasons) > 1001 {
			return out, errors.New("Saisons distantes absentes ou invalides.")
		}
		seen := map[int]bool{}
		for _, raw := range seasons {
			row, ok := raw.(map[string]any)
			if !ok {
				return out, errors.New("Saison distante invalide.")
			}
			n, ok := scopeNumber(row["seasonNumber"])
			if !ok || n > 1000 || seen[n] {
				return out, errors.New("Saison distante invalide ou dupliquée.")
			}
			seen[n] = true
			if q.Season < 0 || n == q.Season {
				out.Seasons = append(out.Seasons, n)
			}
		}
		if len(out.Seasons) == 0 {
			return out, errors.New(p.SeasonMessage)
		}
		sort.Ints(out.Seasons)
	}
	return out, nil
}
