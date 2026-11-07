package acquisition

import (
	"encoding/json"
	"errors"
)

type ObservationQuery struct {
	Connector string `json:"connector"`
	RemoteID  int    `json:"remoteId"`
}
type ObservationRead struct {
	Operation string `json:"operation"`
	Limit     int    `json:"limit"`
}

// Limit is an episode count for episodes, a page count for queue. Pages contain
// at most 100 rows, as required by the acquisition transport contract.
type ObservationPlan struct {
	Version int               `json:"version"`
	Seasons bool              `json:"seasons"`
	Reads   []ObservationRead `json:"reads"`
}

func (p ObservationPlan) Validate(q ObservationQuery) error {
	bad := errors.New("invalid acquisition observation plan")
	if p.Version != 1 || (q.Connector != "radarr" && q.Connector != "sonarr") || q.RemoteID < 0 {
		return bad
	}
	if q.RemoteID == 0 {
		if p.Seasons || len(p.Reads) != 0 {
			return bad
		}
		return nil
	}
	series := q.Connector == "sonarr"
	count := 1
	if series {
		count = 2
	}
	if p.Seasons != series || len(p.Reads) != count {
		return bad
	}
	seen := map[string]bool{}
	for _, r := range p.Reads {
		if seen[r.Operation] || r.Limit < 1 {
			return bad
		}
		seen[r.Operation] = true
		switch r.Operation {
		case "queue":
			if r.Limit > 20 {
				return bad
			}
		case "episodes":
			if !series || r.Limit > 5000 {
				return bad
			}
		default:
			return bad
		}
	}
	if !seen["queue"] || series && !seen["episodes"] {
		return bad
	}
	return nil
}
func evidenceInt(v any) (int, error) {
	if v == nil {
		return 0, nil
	}
	n, ok := scopeNumber(v)
	if !ok {
		return 0, errors.New("Identifiant ou compteur du service invalide.")
	}
	return n, nil
}
func evidenceBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, errors.New("Indicateur du service invalide.")
	}
	return b, nil
}
func evidenceFloat(v any) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case json.Number:
		return n.Float64()
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	}
	return 0, errors.New("Taille de téléchargement invalide.")
}

// Begin and the append helpers run inside the host. Only their typed Tracking
// result may be passed to a module; full remote objects and private paths stay here.
func (p ObservationPlan) Begin(query ObservationQuery, season int, uncertain bool, obj map[string]any) (Tracking, error) {
	q := Tracking{Connector: query.Connector, Season: season, Uncertain: uncertain}
	if err := p.Validate(query); err != nil {
		return q, err
	}
	if obj == nil {
		if query.RemoteID != 0 {
			return q, errors.New("Ressource de suivi absente.")
		}
		return q, q.Validate()
	}
	id, err := evidenceInt(obj["id"])
	if err != nil || id <= 0 || id != query.RemoteID {
		return q, errors.New("Identifiant distant absent ou incohérent.")
	}
	q.RemoteID = id
	if q.HasFile, err = evidenceBool(obj["hasFile"]); err != nil {
		return q, err
	}
	if q.Monitored, err = evidenceBool(obj["monitored"]); err != nil {
		return q, err
	}
	if p.Seasons {
		list, ok := obj["seasons"].([]any)
		if obj["seasons"] != nil && !ok {
			return q, errors.New("Saisons du service invalides.")
		}
		if len(list) > 1001 {
			return q, errors.New("Trop de saisons dans cette réponse de suivi.")
		}
		for _, raw := range list {
			row, ok := raw.(map[string]any)
			if !ok {
				return q, errors.New("Saison invalide dans la réponse du service.")
			}
			number, err := evidenceInt(row["seasonNumber"])
			if err != nil {
				return q, err
			}
			monitored, err := evidenceBool(row["monitored"])
			if err != nil {
				return q, err
			}
			q.Seasons = append(q.Seasons, Season{Number: number, Monitored: monitored})
		}
	}
	return q, q.Validate()
}
func (q *Tracking) ObserveEpisodes(rows []map[string]any, limit int) error {
	if limit < 1 || limit > 5000 || len(rows) > limit {
		return errors.New("Trop d’épisodes pour la limite de suivi du connecteur.")
	}
	for _, row := range rows {
		id, err := evidenceInt(row["id"])
		if err != nil {
			return err
		}
		season, err := evidenceInt(row["seasonNumber"])
		if err != nil {
			return err
		}
		number, err := evidenceInt(row["episodeNumber"])
		if err != nil {
			return err
		}
		hasFile, err := evidenceBool(row["hasFile"])
		if err != nil {
			return err
		}
		q.Episodes = append(q.Episodes, Episode{ID: id, Season: season, Number: number, HasFile: hasFile})
	}
	return q.Validate()
}
func (q *Tracking) ObserveQueue(rows []map[string]any, total, page, maxPages int) (bool, error) {
	bad := errors.New("File du service trop volumineuse ou incomplète pour la limite de suivi du connecteur.")
	if maxPages < 1 || maxPages > 20 || page < 1 || page > maxPages || len(rows) > 100 || total < 0 || total > maxPages*100 {
		return false, bad
	}
	more := page*100 < total
	if more && len(rows) < 100 {
		return false, bad
	}
	// A shrinking queue can legitimately return fewer rows on its final page.
	for _, row := range rows {
		id, err := evidenceInt(row["resourceId"])
		if err != nil {
			return false, err
		}
		episode, err := evidenceInt(row["episodeId"])
		if err != nil {
			return false, err
		}
		size, err := evidenceFloat(row["size"])
		if err != nil {
			return false, err
		}
		left, err := evidenceFloat(row["sizeleft"])
		if err != nil {
			return false, err
		}
		status, ok := row["trackedDownloadStatus"].(string)
		if row["trackedDownloadStatus"] != nil && !ok {
			return false, errors.New("État de téléchargement invalide.")
		}
		item := QueueItem{RemoteID: id, EpisodeID: episode, Size: size, Remaining: left, Status: status}
		if row["seasonNumber"] != nil {
			season, err := evidenceInt(row["seasonNumber"])
			if err != nil {
				return false, err
			}
			item.Season = &season
		}
		q.Queue = append(q.Queue, item)
	}
	return more, q.Validate()
}
