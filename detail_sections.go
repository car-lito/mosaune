package main

import (
	"context"
	"database/sql"
	"errors"
	d "mosaune/extensionkit/capabilities/detailsections/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func migrateDetailSections(db *sql.DB) error {
	var v int
	if e := db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 24 {
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
	s, e = installRecognition(ctx, tx, "detail-information", s.Revision)
	if e != nil {
		return e
	}
	id, e := recognitionID("detail-information")
	if e != nil {
		return e
	}
	if e = registryWanted(ctx, store, s, id, true); e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO recognition_selection(kind,id) VALUES('detail-information',?);", id); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=24"); e != nil {
		return e
	}
	return tx.Commit()
}
func detailSectionsImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	v := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != d.Capability || m.Contributions[0].Target != "media" || len(v.Capabilities) != 1 || v.Capabilities[0] != d.Capability || len(v.Services) != 0 || len(v.Dependencies) != 0 {
		return "", errors.New("Fiche incompatible : rubriques des fiches sans service ni dépendance uniquement.")
	}
	return "detail-information", nil
}

type detailSectionsView struct {
	Owner string `json:"owner"`
	d.Result
	Provider string        `json:"provider"`
	People   []Contributor `json:"people"`
	Trailers []Trailer     `json:"trailers"`
}

func (c *catalog) projectDetailSections(ctx context.Context, q d.Query) (*detailSectionsView, error) {
	if e := q.Validate(); e != nil {
		return nil, e
	}
	c.detailMu.Lock()
	defer c.detailMu.Unlock()
	ctx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	call, done, desc, code, e := c.beginRecognition(ctx, "detail-information")
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
	return &detailSectionsView{Owner: desc.ID, Result: out}, nil
}

// Resource addresses remain host-owned. Modules only receive their array indices.
func safeSectionURL(raw string) string {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(raw) > 2048 {
		return ""
	}
	return raw
}
func sectionFacts(kind string, md *Metadata, count int) (d.Query, []Contributor, []Trailer) {
	q := d.Query{Kind: kind, LocalCount: count, People: []d.Person{}, Trailers: []d.Trailer{}}
	people := []Contributor{}
	trailers := []Trailer{}
	if md == nil {
		return q, people, trailers
	}
	q.Overview = md.Overview
	q.ReleaseDate = md.ReleaseDate
	q.Status = md.Status
	q.Countries = md.Countries
	q.Languages = md.Languages
	q.Cast = md.Cast
	for _, v := range md.Contributors {
		if !strings.HasPrefix(v.Image, "/api/v1/metadata/images/") {
			v.Image = ""
		}
		people = append(people, v)
		q.People = append(q.People, d.Person{Name: v.Name, Role: v.Role})
	}
	for _, v := range md.Trailers {
		v.URL = safeSectionURL(v.URL)
		if v.URL == "" {
			continue
		}
		v.Thumbnail = safeSectionURL(v.Thumbnail)
		trailers = append(trailers, v)
		q.Trailers = append(q.Trailers, d.Trailer{Name: v.Name, Language: v.Language})
	}
	return q, people, trailers
}
func detailSectionsRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	mux.HandleFunc("POST /api/v1/detail-sections", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Kind   string   `json:"kind"`
			WorkID string   `json:"workId"`
			IDs    []string `json:"ids"`
		}
		if !readJSONLimit(w, r, &input, 128<<10) {
			return
		}
		if !allowed(input.Kind) || !canKind(r.Context(), input.Kind) {
			problem(w, 404, "Présentation indisponible.")
			return
		}
		if len(input.IDs) > 1000 || (len(input.IDs) == 0 && input.WorkID == "") {
			problem(w, 400, "Fiche invalide ou trop volumineuse.")
			return
		}
		items, e := c.items(r.Context())
		if e != nil {
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
		seen := map[string]bool{}
		for _, id := range input.IDs {
			m, ok := visible[id]
			if !ok || seen[id] || (input.WorkID != "" && !linked[id]) {
				problem(w, 404, "Fichiers indisponibles pour cette fiche.")
				return
			}
			seen[id] = true
			if md == nil {
				md = m.Metadata
			}
		}
		q, people, trailers := sectionFacts(input.Kind, md, len(input.IDs))
		view, e := c.projectDetailSections(r.Context(), q)
		warning := ""
		if e != nil {
			warning = "Les rubriques de la fiche sont indisponibles. Vérifiez l’extension Informations des fiches. Vos fichiers et métadonnées sont conservés."
		} else {
			view.People = people
			view.Trailers = trailers
			if md != nil {
				view.Provider = md.Provider
			}
		}
		sendJSON(w, 200, map[string]any{"view": view, "warning": warning})
	})
}
