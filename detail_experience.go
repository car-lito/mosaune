package main

import (
	"context"
	"database/sql"
	"errors"
	d "mosaune/extensionkit/capabilities/detailexperience/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"

	"path/filepath"
	"strings"
	"time"
)

func migrateDetailExperience(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 26 {
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
	s, e = installRecognition(ctx, tx, "detail-experience", s.Revision)
	if e != nil {
		return e
	}
	id, e := recognitionID("detail-experience")
	if e != nil {
		return e
	}
	if e = registryWanted(ctx, store, s, id, true); e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO recognition_selection(kind,id) VALUES('detail-experience',?);", id); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=26"); e != nil {
		return e
	}
	return tx.Commit()
}
func detailExperienceImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	v := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != d.Capability || m.Contributions[0].Target != "media" || len(v.Capabilities) != 1 || v.Capabilities[0] != d.Capability || len(v.Services) != 0 || len(v.Dependencies) != 0 {
		return "", errors.New("Fiche incompatible : comportements des fiches sans service ni dépendance uniquement.")
	}
	return "detail-experience", nil
}

type detailExperienceView struct {
	Owner           string        `json:"owner"`
	PeopleResources []Contributor `json:"peopleResources"`
	Provider        string        `json:"provider"`
	d.Result
}

func (c *catalog) projectDetailExperience(ctx context.Context, q d.Query) (*detailExperienceView, error) {
	if e := q.Validate(); e != nil {
		return nil, e
	}
	c.detailMu.Lock()
	defer c.detailMu.Unlock()
	ctx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	call, done, desc, code, e := c.beginRecognition(ctx, "detail-experience")
	if e != nil {
		return nil, e
	}
	defer done()
	engine, e := loadFactProjector(call, desc, code, d.Capability)
	if e != nil {
		return nil, e
	}
	defer engine.engine.Close()
	var out d.Result
	if e = engine.call(call, "project", q, &out); e != nil {
		return nil, e
	}
	if e = out.Validate(q); e != nil {
		return nil, e
	}
	unlock, e := c.recognitionCommit(call)
	if e != nil {
		return nil, e
	}
	defer unlock()
	return &detailExperienceView{Owner: desc.ID, Result: out}, nil
}

// Address-free facts. All item identities have already been filtered by library permissions.
func experienceFacts(kind, focus string, work bool, facts detailFacts) (d.Query, []Contributor) {
	q := d.Query{Kind: kind, Focus: focus, Work: work, Files: []d.File{}, Tracks: []d.Track{}, People: []string{}, Contributors: []int{}}
	people := []Contributor{}
	add := func(p Contributor) int {
		for i, a := range people {
			if a.ID == p.ID && a.Name == p.Name {
				return i
			}
		}
		people = append(people, p)
		q.People = append(q.People, p.Name)
		return len(people) - 1
	}
	if md := facts.metadata; md != nil {
		q.Provider = md.Provider
		for _, p := range md.Contributors {
			if p.Name != "" {
				q.Contributors = append(q.Contributors, add(p))
			}
		}
		for _, t := range md.Tracks {
			x := d.Track{Title: t.Title, Disc: t.Disc, Number: t.Number, Duration: t.Duration, People: []int{}}
			for _, p := range t.Artists {
				if p.Name != "" {
					x.People = append(x.People, add(p))
				}
			}
			q.Tracks = append(q.Tracks, x)
		}
	}
	for _, m := range facts.items {
		x := d.File{ID: m.ID, Name: filepath.Base(strings.ReplaceAll(m.Name, "\\", "/")), Position: m.Position, Completed: m.Completed}
		if md := m.Metadata; md != nil {
			x.Title = md.Title
			x.Artist = md.Artist
			x.AlbumArtist = md.AlbumArtist
			x.Album = md.Album
			x.Disc = md.Disc
			x.Number = md.Track
			x.Season = md.Season
			x.Episode = md.Episode
			for _, name := range []string{md.Artist, md.AlbumArtist} {
				if name != "" {
					add(Contributor{Name: name})
				}
			}
		}
		q.Files = append(q.Files, x)
	}
	return q, people
}
func detailExperienceRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/detail-experience", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Kind    string   `json:"kind"`
			WorkID  string   `json:"workId"`
			IDs     []string `json:"ids"`
			TrackID string   `json:"trackId"`
		}
		if !readJSONLimit(w, r, &input, 128<<10) {
			return
		}
		facts, ok := c.visibleDetailFacts(w, r, allowed, input.Kind, input.WorkID, input.IDs)
		if !ok {
			return
		}
		if input.TrackID != "" {
			found := false
			for _, f := range facts.items {
				if f.ID == input.TrackID {
					found = true
				}
			}
			if !found {
				problem(w, 404, "Fichier indisponible pour cette fiche.")
				return
			}
		}
		q, people := experienceFacts(input.Kind, input.TrackID, input.WorkID != "", facts)
		view, e := c.projectDetailExperience(r.Context(), q)
		warning := ""
		if e != nil {
			warning = "Les commandes et listes enrichies sont indisponibles. Vérifiez l’extension Comportements des fiches. Vos fichiers et métadonnées sont conservés."
		} else {
			view.PeopleResources = people
			view.Provider = q.Provider
		}
		sendJSON(w, 200, map[string]any{"view": view, "warning": warning})
	})
}
