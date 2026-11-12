package acquisition

import "errors"

// ImportPlan requests only bounded read operations. Private paths never cross
// the module boundary; candidates refer to host-owned slots instead.
type ImportPlan struct {
	Version int      `json:"version"`
	Mode    string   `json:"mode"`
	Limit   int      `json:"limit"`
	Reads   []string `json:"reads"`
}

func (p ImportPlan) Validate(q PreviewQuery) error {
	if q.Validate() != nil || p.Version != 1 || p.Limit < 1 || p.Limit > 5000 || (p.Mode != "verified" && p.Mode != "deferred") {
		return errors.New("Plan d’association des imports incompatible.")
	}
	if p.Mode == "verified" && q.Kind == "series" {
		if len(p.Reads) != 1 || p.Reads[0] != "episodefiles" {
			return errors.New("Lecture des imports incompatible.")
		}
	} else if len(p.Reads) != 0 {
		return errors.New("Lecture des imports inattendue.")
	}
	return nil
}

type ImportCandidate struct {
	Slot    int `json:"slot"`
	Season  int `json:"season"`
	Episode int `json:"episode"`
}
type ImportQuery struct {
	Scope      PreviewQuery      `json:"scope"`
	Candidates []ImportCandidate `json:"candidates"`
}

func (q ImportQuery) Validate() error {
	if q.Scope.Validate() != nil || len(q.Candidates) > 5000 {
		return errors.New("Candidats d’import invalides.")
	}
	seen := map[int]bool{}
	for _, c := range q.Candidates {
		if c.Slot < 0 || c.Slot >= 5000 || seen[c.Slot] || c.Season < 0 || c.Season > 1000 || c.Episode < 0 || c.Episode > 100000 || (q.Scope.Kind == "films" && (c.Season != 0 || c.Episode != 0)) || (q.Scope.Kind == "series" && c.Episode == 0) {
			return errors.New("Identité d’import invalide.")
		}
		seen[c.Slot] = true
	}
	return nil
}

type ImportResult struct {
	Slots []int `json:"slots"`
}

func (r ImportResult) Validate(q ImportQuery, p ImportPlan) error {
	if q.Validate() != nil || p.Validate(q.Scope) != nil || len(r.Slots) > p.Limit || (p.Mode == "deferred" && len(r.Slots) > 0) {
		return errors.New("Sélection d’import incompatible.")
	}
	allowed := map[int]bool{}
	for _, c := range q.Candidates {
		if q.Scope.Kind == "films" || q.Scope.Season < 0 || c.Season == q.Scope.Season {
			allowed[c.Slot] = true
		}
	}
	for _, slot := range r.Slots {
		if !allowed[slot] {
			return errors.New("Le composant a choisi un fichier absent, dupliqué ou hors de la saison demandée.")
		}
		delete(allowed, slot)
	}
	return nil
}

// ProjectImportCandidates is host-side evidence reduction. A duplicate file ID,
// duplicate episode identity or multi-episode file is never silently assigned.
func ProjectImportCandidates(q PreviewQuery, episodes, files []map[string]any) ([]ImportCandidate, error) {
	out := []ImportCandidate{}
	if q.Validate() != nil || len(episodes) > 5000 || len(files) > 5000 {
		return out, errors.New("Trop de fichiers ou d’épisodes à rapprocher.")
	}
	if q.Kind == "films" {
		if len(files) > 0 {
			out = append(out, ImportCandidate{Slot: 0})
		}
		return out, nil
	}
	fileCounts := map[int]int{}
	for _, f := range files {
		id, ok := scopeNumber(f["id"])
		if !ok || id <= 0 {
			return out, errors.New("Identifiant de fichier invalide.")
		}
		fileCounts[id]++
	}
	matches := map[int][]ImportCandidate{}
	identities := map[[2]int]int{}
	for _, ep := range episodes {
		has, err := evidenceBool(ep["hasFile"])
		if err != nil {
			return out, err
		}
		if !has {
			continue
		}
		id, ok := scopeNumber(ep["episodeFileId"])
		if !ok || id <= 0 {
			return out, errors.New("Référence de fichier invalide.")
		}
		season, ok := scopeNumber(ep["seasonNumber"])
		if !ok || season > 1000 {
			return out, errors.New("Saison invalide.")
		}
		number, ok := scopeNumber(ep["episodeNumber"])
		if !ok || number < 1 || number > 100000 {
			return out, errors.New("Épisode invalide.")
		}
		matches[id] = append(matches[id], ImportCandidate{Season: season, Episode: number})
		identities[[2]int{season, number}]++
	}
	for slot, f := range files {
		id, _ := scopeNumber(f["id"])
		rows := matches[id]
		if fileCounts[id] != 1 || len(rows) != 1 {
			continue
		}
		row := rows[0]
		if identities[[2]int{row.Season, row.Episode}] != 1 {
			continue
		}
		row.Slot = slot
		out = append(out, row)
	}
	return out, nil
}
