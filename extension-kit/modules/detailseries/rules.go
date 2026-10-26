package detailseries

import (
	"fmt"
	d "mosaune/extensionkit/capabilities/detaillist/v1"
	"sort"
	"strings"
)

func Project(q d.Query) d.Result {
	r := d.Result{Title: "Les saisons", All: "Tous mes épisodes", Files: "Vos épisodes", Help: "Seuls vos fichiers disponibles sont listés ; la série n’est pas nécessairement complète.", Empty: "Aucun épisode de ce groupe sur votre serveur.", Groups: []d.Group{}, Rows: []d.Row{}}
	local := map[int]bool{}
	for _, e := range q.Entries {
		local[e.Season] = true
	}
	known := map[int]bool{}
	add := func(n int, title string, total int, art *int) {
		if strings.TrimSpace(title) == "" {
			title = fmt.Sprintf("Saison %d", n)
			if n == 0 {
				title = "Épisodes spéciaux"
			}
		}
		r.Groups = append(r.Groups, d.Group{Key: fmt.Sprint(n), Title: title, Total: total, Artwork: art, RequestSeason: &n})
		known[n] = true
	}
	for i, s := range q.Seasons {
		if s.Episodes > 0 || local[s.Number] {
			index := i
			add(s.Number, s.Name, s.Episodes, &index)
		}
	}
	for n := range local {
		if !known[n] {
			add(n, "", 0, nil)
		}
	}
	sort.Slice(r.Groups, func(i, j int) bool { return *r.Groups[i].RequestSeason < *r.Groups[j].RequestSeason })
	entries := append([]d.Entry{}, q.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Season != b.Season {
			return a.Season < b.Season
		}
		if a.Episode != b.Episode {
			return a.Episode < b.Episode
		}
		if a.Title != b.Title {
			return a.Title < b.Title
		}
		return a.ID < b.ID
	})
	for _, e := range entries {
		title := e.Title
		if strings.TrimSpace(title) == "" {
			title = "Sans titre"
		}
		r.Rows = append(r.Rows, d.Row{ID: e.ID, Title: title, Ordinal: fmt.Sprintf("%02d", e.Episode), Group: fmt.Sprint(e.Season)})
	}
	return r
}
