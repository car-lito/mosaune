package main

import (
	"context"
	"errors"
	policy "mosaune/extensionkit/capabilities/acquisition/v1"
)

func readAcquisitionAvailability(ctx context.Context, r *cataloguer, q policy.AvailabilityQuery) (policy.AvailabilityResult, error) {
	var out policy.AvailabilityResult
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := r.call(ctx, "local-availability", q, &out); err != nil {
		return out, errors.New("Mettez à jour le composant de demandes pour vérifier la disponibilité locale.")
	}
	return out, out.Validate(q)
}
func (c *catalog) acquisitionAvailability(ctx context.Context, r *cataloguer, v acquisitionRequest, observations policy.Tracking) (int, error) {
	if err := observations.Validate(); err != nil {
		return 0, err
	}
	q := policy.AvailabilityQuery{Scope: policy.PreviewQuery{Kind: v.Kind, Season: v.Season}, Imported: []policy.EpisodeKey{}, Files: []policy.LocalRange{}}
	for _, ep := range observations.Episodes {
		if ep.HasFile {
			q.Imported = append(q.Imported, policy.EpisodeKey{Season: ep.Season, Number: ep.Number})
		}
	}
	work, err := c.work(ctx, v.WorkID)
	if err != nil {
		return 0, err
	}
	if work.Kind != v.Kind || work.Metadata.Provider == "" || work.Metadata.ProviderID == "" {
		return 0, errors.New("Identité du média absente ou incompatible.")
	}
	items, err := c.items(ctx)
	if err != nil {
		return 0, err
	}
	for _, m := range items {
		if err = ctx.Err(); err != nil {
			return 0, err
		}
		md := m.Metadata
		if m.LibraryID != v.Library || m.Kind != v.Kind || md == nil || md.Provider != work.Metadata.Provider || md.ProviderID != work.Metadata.ProviderID {
			continue
		}
		f, _, openErr := c.openMedia(ctx, m.ID)
		if openErr != nil {
			continue
		}
		f.Close()
		if v.Kind == "films" {
			q.Film = observations.HasFile
			break
		}
		end := md.EpisodeEnd
		if end < md.Episode {
			end = md.Episode
		}
		q.Files = append(q.Files, policy.LocalRange{Season: md.Season, First: md.Episode, Last: end})
		if len(q.Files) > 5000 {
			return 0, errors.New("Trop de fichiers pour vérifier cette demande.")
		}
	}
	out, err := readAcquisitionAvailability(ctx, r, q)
	if err != nil {
		return 0, err
	}
	return out.Count(), nil
}
