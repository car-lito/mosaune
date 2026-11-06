package main

import (
	"context"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"strconv"
)

func (client arrClient) trackingReader(ctx context.Context) (*cataloguer, error) {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || s.owner != client.config.owner {
		return nil, errors.New("Session de demandes absente.")
	}
	r, err := loadFactProjector(ctx, s.descriptor, s.code, a.Capability)
	if err != nil {
		return nil, err
	}
	var info struct {
		Version int `json:"version"`
	}
	if err = r.call(ctx, "tracking-info", struct{}{}, &info); err != nil || info.Version != 1 {
		r.engine.Close()
		return nil, errors.New("Le composant de demandes installé ne prend pas en charge le suivi. Mettez-le à jour ou choisissez un composant compatible.")
	}
	return r, nil
}

// Copy only observations needed by the installed policy. Full responses, paths,
// API keys, titles and download identifiers never cross the module boundary.
func (client arrClient) trackingObservations(ctx context.Context, v acquisitionRequest, obj arrObject) (a.Tracking, []arrObject, error) {
	q := a.Tracking{Connector: client.config.ID, Season: v.Season, Uncertain: v.Uncertain}
	var episodes []arrObject
	if obj == nil {
		return q, episodes, nil
	}
	q.RemoteID, q.HasFile, q.Monitored = number(obj["id"]), truth(obj["hasFile"]), truth(obj["monitored"])
	if q.RemoteID <= 0 {
		return q, nil, errors.New("Identifiant distant absent du suivi.")
	}
	if client.config.ID == "sonarr" {
		if err := client.call(ctx, "GET", "episode?seriesId="+strconv.Itoa(q.RemoteID), nil, &episodes); err != nil {
			return q, nil, err
		}
		if len(episodes) > 5000 {
			return q, nil, errors.New("Trop d’épisodes dans cette réponse de suivi.")
		}
		for _, ep := range episodes {
			q.Episodes = append(q.Episodes, a.Episode{ID: number(ep["id"]), Season: number(ep["seasonNumber"]), Number: number(ep["episodeNumber"]), HasFile: truth(ep["hasFile"])})
		}
		seasons, _ := obj["seasons"].([]any)
		if len(seasons) > 1001 {
			return q, nil, errors.New("Trop de saisons dans cette réponse de suivi.")
		}
		for _, raw := range seasons {
			s, ok := raw.(map[string]any)
			if !ok {
				return q, nil, errors.New("Saison invalide dans la réponse du service.")
			}
			q.Seasons = append(q.Seasons, a.Season{Number: number(s["seasonNumber"]), Monitored: truth(s["monitored"])})
		}
	}
	for page := 1; page <= 20; page++ {
		var remote struct {
			Records []arrObject `json:"records"`
			Total   int         `json:"totalRecords"`
		}
		path := "queue?pageSize=100&page=" + strconv.Itoa(page) + "&includeUnknownSeriesItems=false&includeUnknownMovieItems=false"
		if err := client.call(ctx, "GET", path, nil, &remote); err != nil {
			return q, nil, err
		}
		if len(remote.Records) > 100 || remote.Total < 0 || remote.Total > 2000 {
			return q, nil, errors.New("File du service trop volumineuse pour ce prototype ; consultez le service.")
		}
		for _, row := range remote.Records {
			key := "movieId"
			if client.config.ID == "sonarr" {
				key = "seriesId"
			}
			size, _ := row["size"].(float64)
			left, _ := row["sizeleft"].(float64)
			item := a.QueueItem{RemoteID: number(row[key]), EpisodeID: number(row["episodeId"]), Size: size, Remaining: left, Status: word(row["trackedDownloadStatus"])}
			if raw, ok := row["seasonNumber"]; ok && raw != nil {
				n := number(raw)
				item.Season = &n
			}
			q.Queue = append(q.Queue, item)
		}
		if page*100 >= remote.Total {
			break
		}
	}
	return q, episodes, q.Validate()
}

func applyAcquisitionTracking(ctx context.Context, r *cataloguer, q a.Tracking, v *acquisitionRequest) error {
	if err := q.Validate(); err != nil {
		return errors.New("Observations de suivi invalides ou trop volumineuses.")
	}
	var out a.TrackingResult
	if err := r.call(ctx, "track", q, &out); err != nil {
		return err
	}
	if err := out.Validate(q); err != nil {
		return errors.New("Le composant de demandes propose un état de suivi incompatible avec les fichiers vérifiés.")
	}
	v.RemoteID, v.Local = q.RemoteID, q.Local
	v.Total, v.Imported, v.Progress = out.Total, out.Imported, out.Progress
	v.State, v.Message = out.State, out.Message
	return nil
}
