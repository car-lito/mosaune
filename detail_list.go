package main

import (
	"context"
	"database/sql"
	"errors"
	d "mosaune/extensionkit/capabilities/detaillist/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"strings"
	"time"
)

func migrateDetailLists(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 23 {
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
	s, e = installRecognition(ctx, tx, "detail-series", s.Revision)
	if e != nil {
		return e
	}
	id, e := recognitionID("detail-series")
	if e != nil {
		return e
	}
	if e = registryWanted(ctx, store, s, id, true); e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO recognition_selection(kind,id) VALUES('detail-series',?);", id); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=23"); e != nil {
		return e
	}
	return tx.Commit()
}
func detailListImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	v := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != d.Capability || m.Contributions[0].Target != "series" || len(v.Capabilities) != 1 || v.Capabilities[0] != d.Capability || len(v.Services) != 0 || len(v.Dependencies) != 0 {
		return "", errors.New("Fiche incompatible : présentation des séries sans service ni dépendance uniquement.")
	}
	return "detail-series", nil
}

type detailListView struct {
	Owner string `json:"owner"`
	d.Result
	Images []string `json:"images"`
}

func (c *catalog) projectDetailList(ctx context.Context, q d.Query, images []string) (*detailListView, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	c.detailMu.Lock()
	defer c.detailMu.Unlock()
	ctx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	call, done, desc, code, err := c.beginRecognition(ctx, "detail-series")
	if err != nil {
		return nil, err
	}
	defer done()
	r, err := loadFactProjector(call, desc, code, d.Capability)
	if err != nil {
		return nil, err
	}
	defer r.engine.Close()
	var out d.Result
	if err = r.call(call, "project", q, &out); err != nil {
		return nil, err
	}
	if err = out.Validate(q); err != nil {
		return nil, err
	}
	unlock, err := c.recognitionCommit(call)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return &detailListView{Owner: desc.ID, Result: out, Images: images}, nil
}
func detailListRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/detail-list", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Kind   string   `json:"kind"`
			WorkID string   `json:"workId"`
			IDs    []string `json:"ids"`
		}
		if !readJSONLimit(w, r, &input, 128<<10) {
			return
		}
		if input.Kind != "series" || !allowed(input.Kind) || !canKind(r.Context(), input.Kind) {
			problem(w, 404, "Présentation indisponible.")
			return
		}
		if len(input.IDs) > 1000 {
			problem(w, 400, "Cette fiche contient trop de fichiers pour ce module.")
			return
		}
		items, err := c.items(r.Context())
		if err != nil {
			problem(w, 500, "Catalogue indisponible.")
			return
		}
		visible := map[string]media{}
		for _, m := range items {
			if m.Kind == input.Kind {
				visible[m.ID] = m
			}
		}
		var md *Metadata
		linked := map[string]bool{}
		if input.WorkID != "" {
			work, e := c.work(r.Context(), input.WorkID)
			if e != nil || work.Kind != input.Kind {
				problem(w, 404, "Fiche introuvable.")
				return
			}
			attachAvailability(&work, items)
			for _, id := range work.LocalIDs {
				linked[id] = true
			}
			md = &work.Metadata
		}
		q := d.Query{Entries: []d.Entry{}, Seasons: []d.Season{}}
		seen := map[string]bool{}
		for _, id := range input.IDs {
			m, ok := visible[id]
			if !ok || seen[id] || (input.WorkID != "" && !linked[id]) {
				problem(w, 404, "Fichiers indisponibles pour cette fiche.")
				return
			}
			seen[id] = true
			entry := d.Entry{ID: m.ID, Title: catalogueText(m.Name)}
			if m.Metadata != nil {
				entry.Season = m.Metadata.Season
				entry.Episode = m.Metadata.Episode
				if m.Metadata.Title != "" {
					entry.Title = catalogueText(m.Metadata.Title)
				}
				if md == nil {
					md = m.Metadata
				}
			}
			q.Entries = append(q.Entries, entry)
		}
		images := []string{}
		if md != nil {
			for _, s := range md.Seasons {
				q.Seasons = append(q.Seasons, d.Season{Number: s.Number, Name: catalogueText(s.Name), Episodes: s.Episodes})
				image := ""
				if strings.HasPrefix(s.Image, "/api/v1/metadata/images/") {
					image = s.Image
				}
				images = append(images, image)
			}
		}
		result, err := c.projectDetailList(r.Context(), q, images)
		warning := ""
		if err != nil {
			warning = "Présentation des épisodes indisponible. Vos fichiers restent accessibles individuellement. Vérifiez l’extension Fiches des séries."
		}
		sendJSON(w, 200, map[string]any{"view": result, "warning": warning})
	})
}
