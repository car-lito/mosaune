package acquisition

import (
	"errors"
	"sort"
)

type EpisodeKey struct {
	Season int `json:"season"`
	Number int `json:"number"`
}
type LocalRange struct {
	Season int `json:"season"`
	First  int `json:"first"`
	Last   int `json:"last"`
}

// Only readable, identity-checked files are represented here. No local path,
// library, provider ID, title or credential is exposed to the executable.
type AvailabilityQuery struct {
	Scope    PreviewQuery `json:"scope"`
	Film     bool         `json:"film"`
	Imported []EpisodeKey `json:"imported"`
	Files    []LocalRange `json:"files"`
}

func (q AvailabilityQuery) Validate() error {
	bad := errors.New("Preuves de disponibilité invalides.")
	if q.Scope.Validate() != nil || len(q.Imported) > 5000 || len(q.Files) > 5000 {
		return bad
	}
	if q.Scope.Kind == "films" && (len(q.Imported) != 0 || len(q.Files) != 0) {
		return bad
	}
	if q.Scope.Kind == "series" && q.Film {
		return bad
	}
	seen := map[EpisodeKey]bool{}
	for _, e := range q.Imported {
		if e.Season < 0 || e.Season > 1000 || e.Number < 0 || e.Number > 100000 || seen[e] {
			return bad
		}
		seen[e] = true
	}
	for _, f := range q.Files {
		if f.Season < 0 || f.Season > 1000 || f.First < 0 || f.Last < f.First || f.Last > 100000 {
			return bad
		}
	}
	return nil
}

type AvailabilityResult struct {
	Version  int          `json:"version"`
	Film     bool         `json:"film"`
	Episodes []EpisodeKey `json:"episodes"`
}

// Eligible intersects bounded imported identities with merged readable ranges.
// It does not expand ranges, so a multi-episode file cannot create an unbounded
// amount of work. It is also the host's upper bound for extension claims.
func (q AvailabilityQuery) Eligible() (AvailabilityResult, error) {
	out := AvailabilityResult{Version: 1, Film: q.Film, Episodes: []EpisodeKey{}}
	if err := q.Validate(); err != nil {
		return out, err
	}
	ranges := map[int][]LocalRange{}
	for _, f := range q.Files {
		ranges[f.Season] = append(ranges[f.Season], f)
	}
	for season, rows := range ranges {
		sort.Slice(rows, func(i, j int) bool { return rows[i].First < rows[j].First })
		merged := []LocalRange{}
		for _, f := range rows {
			n := len(merged)
			if n > 0 && f.First <= merged[n-1].Last+1 {
				if f.Last > merged[n-1].Last {
					merged[n-1].Last = f.Last
				}
			} else {
				merged = append(merged, f)
			}
		}
		ranges[season] = merged
	}
	for _, e := range q.Imported {
		if q.Scope.Season >= 0 && q.Scope.Season != e.Season {
			continue
		}
		rows := ranges[e.Season]
		i := sort.Search(len(rows), func(i int) bool { return rows[i].Last >= e.Number })
		if i < len(rows) && rows[i].First <= e.Number {
			out.Episodes = append(out.Episodes, e)
		}
	}
	return out, nil
}
func (r AvailabilityResult) Validate(q AvailabilityQuery) error {
	allowed, err := q.Eligible()
	if err != nil {
		return err
	}
	bad := errors.New("Le composant annonce un média sans fichier accessible correspondant.")
	if r.Version != 1 || len(r.Episodes) > 5000 || (r.Film && !allowed.Film) {
		return bad
	}
	keys := map[EpisodeKey]bool{}
	for _, e := range allowed.Episodes {
		keys[e] = true
	}
	for _, e := range r.Episodes {
		if !keys[e] {
			return bad
		}
		delete(keys, e)
	}
	return nil
}
func (r AvailabilityResult) Count() int {
	if r.Film {
		return 1
	}
	return len(r.Episodes)
}
