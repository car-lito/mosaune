package main

import (
	"context"
	"errors"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/url"
	"time"
)

func checkMusicVisuals(ctx context.Context, r *cataloguer) error {
	var info struct{ Version int }
	if e := r.call(ctx, "describe-visuals", struct{}{}, &info); e != nil || info.Version != 1 {
		return errors.New("Mettez à jour le composant musical pour rechercher les pochettes et portraits. Les images déjà conservées restent disponibles.")
	}
	return nil
}

// The extension chooses the workflow. The host only executes validated steps
// using its fixed origins, transport policies, shared cache and cancellation.
func (p *musicBrainzProvider) runMusicVisual(ctx context.Context, q d.VisualQuery) ([]byte, commonsPhoto, error) {
	var empty commonsPhoto
	if e := q.Validate(); e != nil {
		return nil, empty, e
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	ctx, done, e := p.openMusicResults(ctx)
	if e != nil {
		return nil, empty, e
	}
	defer done()
	reader := ctx.Value(musicResultsKey{}).(*musicResultsReader)
	if e = checkMusicVisuals(ctx, reader.engine); e != nil {
		return nil, empty, e
	}
	for i := 0; i < 6; i++ {
		if e = ctx.Err(); e != nil {
			return nil, empty, e
		}
		var step d.VisualStep
		if e = reader.engine.call(ctx, "visual", q, &step); e != nil {
			return nil, empty, e
		}
		if e = step.Validate(q); e != nil {
			return nil, empty, errors.New("Réponse d’image du composant refusée.")
		}
		if step.Missing {
			return nil, empty, errors.New("Aucune image renseignée.")
		}
		if step.Photo != nil {
			return nil, *step.Photo, nil
		}
		f := step.Fetch
		params := url.Values{}
		for k, v := range f.Params {
			params.Set(k, v)
		}
		q.State = step.State
		q.Body = nil
		q.Status = 200
		if f.Source == "musicbrainz" {
			if e = p.get(ctx, f.Path, params, &q.Body); e != nil {
				return nil, empty, e
			}
		} else {
			origins := map[string]string{"wikidata": "https://www.wikidata.org", "commons": "https://commons.wikimedia.org", "coverart": "https://coverartarchive.org"}
			endpoint := origins[f.Source] + f.Path
			if len(params) > 0 {
				endpoint += "?" + params.Encode()
			}
			if f.Image {
				data, status, err := p.fetchVisual(ctx, endpoint)
				if err == nil {
					if e = ctx.Err(); e != nil {
						return nil, empty, context.Cause(ctx)
					}
					return data, empty, nil
				}
				if status != 404 {
					return nil, empty, err
				}
				q.Status = 404
			} else if e = p.publicJSON(ctx, endpoint, &q.Body); e != nil {
				return nil, empty, e
			}
		}
		if e = q.Validate(); e != nil {
			return nil, empty, errors.New("Réponse d’image invalide ou trop volumineuse.")
		}
	}
	return nil, empty, errors.New("Le composant musical a dépassé le nombre d’étapes autorisé.")
}

func (p *musicBrainzProvider) artistPhoto(ctx context.Context, id string) (commonsPhoto, error) {
	_, photo, e := p.runMusicVisual(ctx, d.VisualQuery{Operation: "artist", Ref: id})
	return photo, e
}
func (p *musicBrainzProvider) visualArtwork(ctx context.Context, ref string) ([]byte, error) {
	data, _, e := p.runMusicVisual(ctx, d.VisualQuery{Operation: "artwork", Ref: ref})
	return data, e
}
