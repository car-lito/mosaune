package main

import (
	"encoding/json"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

func track(q a.Tracking) (a.TrackingResult, error) {
	total, imported := q.Counts()
	r := a.TrackingResult{Total: total, Imported: imported, State: "waiting", Message: "En attente d’une disponibilité dans le service."}
	if q.RemoteID == 0 {
		r.State, r.Message = "missing", "Absent du service. Vérifiez la demande dans "+q.Connector+" ; Mosaune ne la renvoie pas automatiquement."
		return r, r.Validate(q)
	}
	monitored := q.Monitored
	if q.Connector == "sonarr" && q.Season >= 0 {
		for _, s := range q.Seasons {
			if s.Number == q.Season {
				monitored = monitored && s.Monitored
			}
		}
	}
	if !monitored {
		r.State, r.Message = "unmonitored", "La surveillance est désactivée dans le service."
	}
	if q.Uncertain {
		r.State, r.Message = "uncertain", "Envoi non confirmé. L’œuvre existe dans le service, mais vérifiez sa surveillance et sa recherche avant de relancer."
	}
	episodeSeasons := map[int]int{}
	for _, ep := range q.Episodes {
		episodeSeasons[ep.ID] = ep.Season
	}
	queued, blocked := false, false
	var size, left float64
	for _, row := range q.Queue {
		if row.RemoteID != q.RemoteID {
			continue
		}
		if q.Connector == "sonarr" && q.Season >= 0 {
			s, ok := episodeSeasons[row.EpisodeID]
			if !ok && row.Season != nil {
				s, ok = *row.Season, true
			}
			if !ok || s != q.Season {
				continue
			}
		}
		queued = true
		size += row.Size
		left += row.Remaining
		blocked = blocked || row.Status == "warning" || row.Status == "error"
	}
	if size > 0 {
		r.Progress = max(0, min(100, (size-left)*100/size))
	}
	if queued {
		r.State, r.Message = "downloading", "Téléchargement en cours ou en attente d’importation."
		if blocked {
			r.State, r.Message = "blocked", "Le service signale un blocage. Consultez sa file d’activité."
		}
	}
	if q.Local > 0 {
		if total > 0 && q.Local >= total {
			r.State, r.Message = "available", "Disponible dans votre bibliothèque."
		} else if !queued {
			r.State, r.Message = "partial", "Des épisodes sont disponibles. La série ou la saison n’est pas encore complète."
		}
	} else if imported > 0 && !queued {
		r.State, r.Message = "imported", "Importé par le service, mais pas encore lisible et identifié dans Mosaune. Vérifiez le dossier associé ; analyse automatique toutes les cinq minutes."
	}
	var policy struct {
		WaitingMessage string `json:"waitingMessage"`
	}
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return r, err
	}
	if r.State == "waiting" && policy.WaitingMessage != "" {
		r.Message = policy.WaitingMessage
	}
	return r, r.Validate(q)
}
