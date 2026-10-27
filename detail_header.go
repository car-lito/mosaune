package main

import (
	"context"
	"database/sql"
	"errors"
	d "mosaune/extensionkit/capabilities/detailheader/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"

	"time"
)

func migrateDetailHeader(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 25 {
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
	s, e = installRecognition(ctx, tx, "detail-header", s.Revision)
	if e != nil {
		return e
	}
	id, e := recognitionID("detail-header")
	if e != nil {
		return e
	}
	if e = registryWanted(ctx, store, s, id, true); e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO recognition_selection(kind,id) VALUES('detail-header',?);", id); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=25"); e != nil {
		return e
	}
	return tx.Commit()
}
func detailHeaderImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	v := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != d.Capability || m.Contributions[0].Target != "media" || len(v.Capabilities) != 1 || v.Capabilities[0] != d.Capability || len(v.Services) != 0 || len(v.Dependencies) != 0 {
		return "", errors.New("Fiche incompatible : en-têtes des fiches sans service ni dépendance uniquement.")
	}
	return "detail-header", nil
}

type detailHeaderView struct {
	Owner string `json:"owner"`
	d.Result
}

func (c *catalog) projectDetailHeader(ctx context.Context, q d.Query) (*detailHeaderView, error) {
	if e := q.Validate(); e != nil {
		return nil, e
	}
	c.detailMu.Lock()
	defer c.detailMu.Unlock()
	ctx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	call, done, desc, code, e := c.beginRecognition(ctx, "detail-header")
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
	return &detailHeaderView{Owner: desc.ID, Result: out}, nil
}

// Only visible, stored text facts enter the extension; no client-supplied titles.
func headerFacts(kind string, md *Metadata, first media, storedWork, track bool, count int) d.Query {
	q := d.Query{Kind: kind, Track: track, StoredWork: storedWork, Name: first.Name, LocalCount: count, Genres: []string{}}
	if first.Catalogue != nil {
		q.GroupTitle = first.Catalogue.Title
	}
	if md != nil {
		q.Title = md.Title
		q.Album = md.Album
		q.Series = md.Series
		q.Year = md.Year
		q.Runtime = md.Runtime
		q.Genres = md.Genres
		q.Rating = md.Rating
		q.VoteCount = md.VoteCount
		q.Provider = md.Provider
		q.Tagline = md.Tagline
		q.Edition = md.Edition
	}
	return q
}
func detailHeaderRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/detail-header", func(w http.ResponseWriter, r *http.Request) {
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
		md := facts.metadata
		first := facts.first
		if input.WorkID == "" && input.TrackID == "" && len(facts.items) > 0 {
			c.projectCatalogue(r.Context(), facts.items)
			first = facts.items[0]
		}
		if input.TrackID != "" {
			var found bool
			for _, item := range facts.items {
				if item.ID == input.TrackID {
					first = item
					md = item.Metadata
					found = true
					break
				}
			}
			if input.Kind != "music" || !found {
				problem(w, 404, "Morceau indisponible pour cette fiche.")
				return
			}
		}
		q := headerFacts(input.Kind, md, first, input.WorkID != "" && input.TrackID == "", input.TrackID != "", len(input.IDs))
		view, e := c.projectDetailHeader(r.Context(), q)
		warning := ""
		if e != nil {
			warning = "L’en-tête enrichi est indisponible. Vérifiez l’extension En-têtes des fiches. Vos fichiers et métadonnées sont conservés."
		}
		sendJSON(w, 200, map[string]any{"view": view, "warning": warning})
	})
}
