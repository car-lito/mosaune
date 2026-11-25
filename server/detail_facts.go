package main

import (
	"net/http"
)

type detailFacts struct {
	metadata *Metadata
	first    media
	items    []media
}

func (c *catalog) visibleDetailFacts(w http.ResponseWriter, r *http.Request, allowed func(string) bool, kind, workID string, ids []string) (detailFacts, bool) {
	if !allowed(kind) || !canKind(r.Context(), kind) {
		problem(w, 404, "Présentation indisponible.")
		return detailFacts{}, false
	}
	if len(ids) > 1000 || (len(ids) == 0 && workID == "") {
		problem(w, 400, "Fiche invalide ou trop volumineuse.")
		return detailFacts{}, false
	}
	items, e := c.items(r.Context())
	if e != nil {
		problem(w, 500, "Catalogue indisponible.")
		return detailFacts{}, false
	}
	visible := map[string]media{}
	for _, m := range items {
		if m.Kind == kind {
			visible[m.ID] = m
		}
	}
	var md *Metadata
	linked := map[string]bool{}
	if workID != "" {
		work, e := c.work(r.Context(), workID)
		if e != nil || work.Kind != kind {
			problem(w, 404, "Fiche introuvable.")
			return detailFacts{}, false
		}
		attachAvailability(&work, items)
		for _, id := range work.LocalIDs {
			linked[id] = true
		}
		md = &work.Metadata
	}
	selected := []media{}
	seen := map[string]bool{}
	for _, id := range ids {
		m, ok := visible[id]
		if !ok || seen[id] || (workID != "" && !linked[id]) {
			problem(w, 404, "Fichiers indisponibles pour cette fiche.")
			return detailFacts{}, false
		}
		seen[id] = true
		selected = append(selected, m)
		if md == nil {
			md = m.Metadata
		}
	}

	var first media
	if len(selected) > 0 {
		first = selected[0]
	}
	return detailFacts{metadata: md, first: first, items: selected}, true
}
