package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
	"net/http"
	"slices"
)

// Declarative installation recipes, not complete media extensions.
// Status is derived from installed programs; no parallel activation state.
//
//go:embed media-sets.json
var mediaSetsJSON []byte

type mediaSet struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Slots []string `json:"slots"`
}
type mediaSetStep struct {
	Slot      string `json:"slot"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Operation string `json:"operation"`
	Shared    bool   `json:"shared"`
}
type mediaSetPlan struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Operation string         `json:"operation"`
	Revision  int64          `json:"revision"`
	Steps     []mediaSetStep `json:"steps"`
	Kept      []mediaSetStep `json:"kept"`
	Blockers  []string       `json:"blockers"`
	Checksum  string         `json:"checksum"`
}

func mediaSetDefinitions() ([]mediaSet, error) {
	var sets []mediaSet
	if err := json.Unmarshal(mediaSetsJSON, &sets); err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, set := range sets {
		if set.ID == "" || set.Name == "" || ids[set.ID] || len(set.Slots) == 0 {
			return nil, errors.New("Invalid media set")
		}
		ids[set.ID] = true
		seen := map[string]bool{}
		for _, slot := range set.Slots {
			if !slices.Contains(extractionKinds, slot) || seen[slot] {
				return nil, errors.New("Invalid media set slot")
			}
			seen[slot] = true
		}
	}
	return sets, nil
}

// Never replace a missing third-party selection with a factory implicitly.
// Disabled selected programs are explicitly shown as enable in the preview.
func planMediaSet(sets []mediaSet, id, operation string, packages []recognitionPackageView, selected map[string]string, revision int64, safe bool) (mediaSetPlan, error) {
	p := mediaSetPlan{ID: id, Operation: operation, Revision: revision, Steps: []mediaSetStep{}, Kept: []mediaSetStep{}, Blockers: []string{}}
	if operation != "prepare" && operation != "disable" && operation != "remove" {
		return p, errors.New("Action inconnue.")
	}
	var set *mediaSet
	counts := map[string]int{}
	for i := range sets {
		if sets[i].ID == id {
			set = &sets[i]
		}
		for _, slot := range sets[i].Slots {
			counts[slot]++
		}
	}
	if set == nil {
		return p, errors.New("Ensemble inconnu.")
	}
	p.Name = set.Name
	if safe && operation == "prepare" {
		p.Blockers = append(p.Blockers, "Quittez le mode de secours pour préparer cet ensemble.")
	}
	for _, slot := range set.Slots {
		var factory, chosen *recognitionPackageView
		for i := range packages {
			v := &packages[i]
			if v.Kind != slot {
				continue
			}
			if v.Factory {
				factory = v
			}
			if v.ID == selected[slot] {
				chosen = v
			}
		}
		if factory == nil {
			return p, fmt.Errorf("Composant fourni absent : %s", slot)
		}
		if chosen == nil && selected[slot] == "" {
			chosen = factory
		}
		if chosen == nil {
			p.Blockers = append(p.Blockers, "Le module choisi pour "+factory.Name+" a été retiré. Réimportez-le ou choisissez explicitement un remplaçant.")
			continue
		}
		row := mediaSetStep{Slot: slot, ID: chosen.ID, Name: chosen.Name, Version: chosen.Version, Shared: counts[slot] > 1}
		add := func(op string) { v := row; v.Operation = op; p.Steps = append(p.Steps, v) }
		keep := func(op string) { v := row; v.Operation = op; p.Kept = append(p.Kept, v) }
		if operation != "prepare" && row.Shared {
			keep("shared")
			continue
		}
		if operation == "prepare" {
			if !chosen.Installed {
				add("install")
			}
			if !chosen.Enabled || !chosen.Installed {
				add("enable")
			}
			if selected[slot] != chosen.ID {
				add("select")
			}
			if chosen.Installed && chosen.Enabled && selected[slot] == chosen.ID {
				keep("ready")
			}
		} else if operation == "disable" {
			if chosen.Installed && chosen.Enabled {
				add("disable")
			} else {
				keep("unchanged")
			}
		} else {
			if chosen.Installed {
				add("remove")
			} else {
				keep("unchanged")
			}
		}
	}
	raw, _ := json.Marshal(p)
	p.Checksum = recognitionHash(raw)
	return p, nil
}

// Called under the lifecycle lock also used by individual module changes.
func (c *catalog) mediaSetInputs(ctx context.Context, tx *sql.Tx) ([]recognitionPackageView, map[string]string, int64, error) {
	s, err := (registry.SQLStore{DB: tx}).Load(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	chosen := map[string]string{}
	rows, err := tx.QueryContext(ctx, "SELECT kind,id FROM recognition_selection")
	if err != nil {
		return nil, nil, 0, err
	}
	for rows.Next() {
		var k, id string
		if err = rows.Scan(&k, &id); err != nil {
			rows.Close()
			return nil, nil, 0, err
		}
		chosen[k] = id
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, 0, err
	}
	list := []recognitionPackageView{}
	names := map[string]string{}
	rows, err = tx.QueryContext(ctx, "SELECT id,kind,manifest FROM recognition_packages")
	if err != nil {
		return nil, nil, 0, err
	}
	kinds := map[string]string{}
	for rows.Next() {
		var id, k string
		var raw []byte
		if err = rows.Scan(&id, &k, &raw); err != nil {
			rows.Close()
			return nil, nil, 0, err
		}
		kinds[id] = k
		var m struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(raw, &m)
		names[id] = m.Name
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, 0, err
	}
	seen := map[string]bool{}
	for _, slot := range extractionKinds {
		d, e := modules.Descriptor(slot)
		if e != nil {
			return nil, nil, 0, e
		}
		entry, exists := registryEntryFor(s, d.ID)
		if exists && kinds[d.ID] != slot {
			return nil, nil, 0, errors.New("Paquet et registre incohérents.")
		}
		v := d.Version
		if exists {
			v = entry.Descriptor.Version
		}
		list = append(list, recognitionPackageView{Kind: slot, ID: d.ID, Name: mediaSlotName(slot), Version: v, Factory: true, Installed: exists, Enabled: exists && entry.Enabled})
		seen[d.ID] = true
	}
	for _, e := range s.Entries {
		id := e.Descriptor.ID
		if seen[id] || kinds[id] == "" {
			continue
		}
		name := names[id]
		if name == "" {
			name = id
		}
		list = append(list, recognitionPackageView{Kind: kinds[id], ID: id, Name: name, Version: e.Descriptor.Version, Installed: true, Enabled: e.Enabled})
	}
	return list, chosen, s.Revision, nil
}
func mediaSlotName(slot string) string {
	names := map[string]string{"films": "Reconnaissance des films", "series": "Reconnaissance des séries", "music": "Reconnaissance musicale", "audiotags": "Informations des fichiers musicaux", "catalogue-films": "Catalogue des films", "catalogue-series": "Catalogue des séries", "catalogue-music": "Catalogue musical", "detail-series": "Saisons et épisodes", "detail-information": "Informations des fiches", "detail-header": "En-têtes des fiches", "detail-experience": "Comportements des fiches"}
	if name := names[slot]; name != "" {
		return name
	}
	return slot
}

func (c *catalog) mediaSetPlans(ctx context.Context, id, op string, revision *int64, checksum string, apply bool) ([]mediaSetPlan, error) {
	c.recognition.mu.Lock()
	defer c.recognition.mu.Unlock()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	sets, err := mediaSetDefinitions()
	if err != nil {
		return nil, err
	}
	packages, selected, current, err := c.mediaSetInputs(ctx, tx)
	if err != nil {
		return nil, err
	}
	if revision != nil && *revision != current {
		return nil, registry.ErrConflict
	}
	plans := []mediaSetPlan{}
	for _, s := range sets {
		if id != "" && id != s.ID {
			continue
		}
		p, e := planMediaSet(sets, s.ID, op, packages, selected, current, c.extensionSafeMode)
		if e != nil {
			return nil, e
		}
		plans = append(plans, p)
	}
	if len(plans) == 0 {
		return nil, errors.New("Ensemble inconnu.")
	}
	if !apply {
		return plans, nil
	}
	if id == "" || revision == nil || checksum == "" || checksum != plans[0].Checksum {
		return nil, errors.New("Aperçu périmé. Vérifiez de nouveau les changements.")
	}
	p := plans[0]
	if len(p.Blockers) > 0 {
		return nil, errors.New("Résolvez les composants manquants avant cette action.")
	}
	for _, step := range p.Steps {
		if _, err = changeRecognitionTx(ctx, tx, step.ID, step.Operation, current); err != nil {
			return nil, err
		}
		s, e := (registry.SQLStore{DB: tx}).Load(ctx)
		if e != nil {
			return nil, e
		}
		current = s.Revision
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	for _, step := range p.Steps {
		if step.Operation == "remove" || step.Operation == "disable" {
			c.cancelRecognitionID(step.ID)
		}
		if step.Operation == "select" {
			for _, call := range c.recognition.calls {
				if call.kind == step.Slot {
					call.cancel(errRecognitionUnavailable)
				}
			}
		}
	}
	return plans, nil
}
func (c *catalog) mediaSetRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/media-sets", func(w http.ResponseWriter, r *http.Request) {
		plans, err := c.mediaSetPlans(r.Context(), "", "prepare", nil, "", false)
		if err != nil {
			problem(w, 500, "Ensembles indisponibles.")
			return
		}
		sendJSON(w, 200, map[string]any{"sets": plans})
	})
	for _, action := range []string{"preview", "apply"} {
		mux.HandleFunc("POST /api/v1/media-sets/{id}/"+action, func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Operation string `json:"operation"`
				Revision  *int64 `json:"revision"`
				Checksum  string `json:"checksum"`
			}
			if !readJSON(w, r, &in) {
				return
			}
			if in.Revision == nil {
				problem(w, 400, "Actualisez les ensembles avant cette action.")
				return
			}
			plans, err := c.mediaSetPlans(r.Context(), r.PathValue("id"), in.Operation, in.Revision, in.Checksum, action == "apply")
			if err != nil {
				problem(w, 409, "Aucune modification appliquée. "+err.Error())
				return
			}
			sendJSON(w, 200, plans[0])
		})
	}
}
