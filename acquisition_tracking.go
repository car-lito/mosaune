package main

import (
	"context"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
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
	if _, err = readAcquisitionObservations(ctx, r, a.ObservationQuery{Connector: client.config.ID}); err != nil {
		r.engine.Close()
		return nil, err
	}
	return r, nil
}

func readAcquisitionObservations(ctx context.Context, r *cataloguer, query a.ObservationQuery) (a.ObservationPlan, error) {
	var p a.ObservationPlan
	if err := r.call(ctx, "observation-plan", query, &p); err != nil {
		return p, errors.New("Le composant de demandes installé ne fournit pas de plan de suivi compatible. Mettez-le à jour.")
	}
	return p, p.Validate(query)
}

// Full responses and private paths remain in the host. The installed component
// chooses bounded reads; the shared contract projects only typed evidence.
func (client arrClient) trackingObservations(ctx context.Context, reader *cataloguer, v acquisitionRequest, obj arrObject) (a.Tracking, []arrObject, error) {
	query := a.ObservationQuery{Connector: client.config.ID, RemoteID: number(obj["id"])}
	p, err := readAcquisitionObservations(ctx, reader, query)
	if err != nil {
		return a.Tracking{}, nil, err
	}
	q, err := p.Begin(query, v.Season, v.Uncertain, obj)
	if err != nil {
		return q, nil, err
	}
	var episodes []arrObject
	for _, read := range p.Reads {
		switch read.Operation {
		case "episodes":
			if err = client.exchange(ctx, read.Operation, q.RemoteID, 0, nil, &episodes); err != nil {
				return q, nil, err
			}
			rows := make([]map[string]any, len(episodes))
			for i := range episodes {
				rows[i] = episodes[i]
			}
			if err = q.ObserveEpisodes(rows, read.Limit); err != nil {
				return q, nil, err
			}
		case "queue":
			for page := 1; page <= read.Limit; page++ {
				var remote struct {
					Records []map[string]any `json:"records"`
					Total   int              `json:"totalRecords"`
				}
				if err = client.exchange(ctx, read.Operation, 0, page, nil, &remote); err != nil {
					return q, nil, err
				}
				more, err := q.ObserveQueue(remote.Records, remote.Total, page, read.Limit)
				if err != nil {
					return q, nil, err
				}
				if !more {
					break
				}
			}
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
