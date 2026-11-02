package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	discovery "mosaune/extensionkit/discovery/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/url"
	"strings"
)

func migrateMusicResults(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 34 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	ctx := context.Background()
	store := registry.SQLStore{DB: tx}
	s, e := store.Load(ctx)
	if e != nil {
		return e
	}
	var chosen string
	e = tx.QueryRow("SELECT id FROM recognition_selection WHERE kind='music'").Scan(&chosen)
	if e != nil && e != sql.ErrNoRows {
		return e
	}
	entry, present := registryEntryFor(s, chosen)
	id, e := recognitionID("music-results")
	if e != nil {
		return e
	}
	var selectionExists bool
	if e = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM recognition_selection WHERE kind='music-results')").Scan(&selectionExists); e != nil {
		return e
	}
	if present && !selectionExists {
		s, e = installRecognition(ctx, tx, "music-results", s.Revision)
		if e != nil {
			return e
		}
		if e = registryWanted(ctx, store, s, id, entry.Enabled); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("INSERT OR IGNORE INTO recognition_selection(kind,id) VALUES('music-results',?)", id); e != nil {
		return e
	}
	// Only the supplied recipe is extended; imported families retain their manifests.
	var raw string
	if e = tx.QueryRow("SELECT manifest FROM media_bundles WHERE id='org.mosaune.media.music'").Scan(&raw); e == nil {
		var old mediaBundleManifest
		if json.Unmarshal([]byte(raw), &old) == nil && old.Version == "1.0.0" {
			expected, e := bundledFamily("music")
			if e != nil {
				return e
			}
			legacy := expected
			legacy.Version = "1.0.0"
			legacy.Components = append([]bundleComponent(nil), expected.Components[:len(expected.Components)-1]...)
			a, _ := json.Marshal(old)
			b, _ := json.Marshal(legacy)
			if string(a) == string(b) {
				next, _ := json.Marshal(expected)
				if _, e = tx.Exec("UPDATE media_bundles SET manifest=? WHERE id=?", string(next), old.ID); e != nil {
					return e
				}
			}
		}
	} else if e != sql.ErrNoRows {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=34"); e != nil {
		return e
	}
	return tx.Commit()
}
func musicResultsImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	v := m.Module
	if strings.HasPrefix(v.ID, "mosaune.") || packageformat.Compare(m.MinCore, "0.0.55") < 0 || len(m.Contributions) != 1 || m.Contributions[0].Role != d.Capability || m.Contributions[0].Target != "music" || len(v.Capabilities) != 1 || v.Capabilities[0] != d.Capability || len(v.Services) != 0 || len(v.Dependencies) != 0 {
		return "", errors.New("Décodeur musical incompatible : résultats musicaux bornés, sans service ni dépendance.")
	}
	return "music-results", nil
}

type musicResultsKey struct{}
type musicResultsReader struct {
	owner  *musicBrainzProvider
	engine *cataloguer
}

func checkMusicRequests(ctx context.Context, r *cataloguer) error {
	var info struct{ Version int }
	if e := r.call(ctx, "describe-requests", struct{}{}, &info); e != nil || info.Version != 1 {
		return errors.New("Ce composant musical doit être mis à jour : la préparation des recherches (version 1) est requise. Vos fiches sont conservées.")
	}
	return nil
}

func (p *musicBrainzProvider) requestMusic(ctx context.Context, q d.Request) (d.Result, error) {
	var out d.Result
	if e := q.Validate(); e != nil {
		return out, errors.New("Recherche musicale invalide.")
	}
	reader, ok := ctx.Value(musicResultsKey{}).(*musicResultsReader)
	if !ok || reader.owner != p {
		return out, errors.New("Composant musical absent.")
	}
	var plan d.RequestPlan
	if e := reader.engine.call(ctx, "plan", q, &plan); e != nil {
		return out, e
	}
	if e := plan.Validate(q); e != nil {
		return out, errors.New("Requête du composant musical refusée.")
	}
	params := url.Values{}
	for k, v := range plan.Params {
		params.Set(k, v)
	}
	var raw json.RawMessage
	if e := p.get(ctx, plan.Path, params, &raw); e != nil {
		return out, e
	}
	return p.decodeMusic(ctx, plan.Decode, plan.ID, q.Page, raw)
}

func (p *musicBrainzProvider) openMusicResults(ctx context.Context) (context.Context, func(), error) {
	if v, ok := ctx.Value(musicResultsKey{}).(*musicResultsReader); ok && v.owner == p {
		return ctx, func() {}, ctx.Err()
	}
	select {
	case p.resultsGate <- struct{}{}:
	case <-ctx.Done():
		return ctx, func() {}, ctx.Err()
	}
	release := func() { <-p.resultsGate }
	var reader *cataloguer
	var done = func() {}
	if p.db == nil {
		desc, code, e := modules.Lookup("music-results")
		if e != nil {
			release()
			return ctx, func() {}, e
		}
		reader, e = loadFactProjector(ctx, desc, code, d.Capability)
		if e != nil {
			release()
			return ctx, func() {}, e
		}
	} else {
		call, stop, desc, code, e := p.owner.beginRecognition(ctx, "music-results")
		if e != nil {
			release()
			return ctx, func() {}, fmt.Errorf("Le composant Albums et artistes MusicBrainz est indisponible. Vérifiez-le dans Musique : %w", e)
		}
		ctx = call
		done = stop
		reader, e = loadFactProjector(ctx, desc, code, d.Capability)
		if e != nil {
			done()
			release()
			return ctx, func() {}, e
		}
	}
	if e := checkMusicRequests(ctx, reader); e != nil {
		reader.engine.Close()
		done()
		release()
		return ctx, func() {}, e
	}
	return context.WithValue(ctx, musicResultsKey{}, &musicResultsReader{p, reader}), func() { reader.engine.Close(); done(); release() }, nil
}
func (p *musicBrainzProvider) decodeMusic(ctx context.Context, operation, id string, page int, raw json.RawMessage) (d.Result, error) {
	q := d.Query{Operation: operation, ID: id, Page: page, Body: raw}
	var out d.Result
	if e := q.Validate(); e != nil {
		return out, errors.New("Réponse musicale invalide ou trop volumineuse (maximum 512 Kio).")
	}
	reader, ok := ctx.Value(musicResultsKey{}).(*musicResultsReader)
	if !ok || reader.owner != p {
		return out, errors.New("Décodeur musical absent.")
	}
	if e := reader.engine.call(ctx, "decode", q, &out); e != nil {
		return out, e
	}
	return out, out.Validate(q)
}
func musicResultMetadata(m d.Item) Metadata {
	out := Metadata{Presentation: m.Presentation, DetailVersion: m.DetailVersion, ReleaseDate: m.ReleaseDate, Edition: m.Edition, Title: m.Title, Artist: m.Artist, Album: m.Album, AlbumArtist: m.AlbumArtist, Year: m.Year, ArtworkRef: m.ArtworkRef, ProviderID: m.ProviderID, Provider: "musicbrainz", MatchState: "matched"}
	people := func(xs []d.Contributor) []Contributor {
		v := []Contributor{}
		for _, x := range xs {
			v = append(v, Contributor{ID: x.ID, Name: x.Name, Role: x.Role})
		}
		return v
	}
	out.Contributors = people(m.Contributors)
	for _, t := range m.Tracks {
		out.Tracks = append(out.Tracks, CatalogTrack{ID: t.ID, Title: t.Title, Disc: t.Disc, Number: t.Number, Duration: t.Duration, Artists: people(t.Artists)})
	}
	return out
}

func (p *musicBrainzProvider) DiscoverySurfaceContext(ctx context.Context) (discovery.Surface, error) {
	empty := discovery.Surface{Version: 1}
	if p.db == nil {
		return p.legacyDiscoverySurface(), ctx.Err()
	}
	if p.owner.extensionSafeMode {
		return empty, nil
	}
	var selected string
	if e := p.db.QueryRowContext(ctx, "SELECT id FROM recognition_selection WHERE kind='music-results'").Scan(&selected); e != nil {
		if e == sql.ErrNoRows {
			return empty, nil
		}
		return empty, e
	}
	s, e := (registry.SQLStore{DB: p.db}).Load(ctx)
	if e != nil {
		return empty, e
	}
	entry, ok := registryEntryFor(s, selected)
	if !ok || !entry.Enabled {
		return empty, nil
	}
	return p.legacyDiscoverySurface(), nil
}
