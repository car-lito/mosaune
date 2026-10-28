package detailexperience

import (
	"fmt"
	"golang.org/x/text/unicode/norm"
	d "mosaune/extensionkit/capabilities/detailexperience/v1"
	"sort"
	"strings"
	"unicode"
)

func action(op, label, id string) d.Action { return d.Action{Operation: op, Label: label, FileID: id} }
func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
func normalized(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFKD.String(s))
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }), " ")
}
func disc(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
func match(t d.Track, byTitle map[string][]d.File, used map[string]bool) *d.File {
	var exact, numbered []d.File
	for _, f := range byTitle[normalized(t.Title)] {
		if !used[f.ID] {
			exact = append(exact, f)
			if disc(f.Disc) == disc(t.Disc) && f.Number == t.Number {
				numbered = append(numbered, f)
			}
		}
	}
	if len(numbered) == 1 {
		return &numbered[0]
	}
	if len(exact) == 1 {
		return &exact[0]
	}
	return nil
}
func Project(q d.Query) d.Result {
	r := d.Result{Cover: "portrait", Actions: []d.Action{}, People: []int{}, Sections: []d.Section{}}
	files := append([]d.File{}, q.Files...)
	sort.SliceStable(files, func(i, j int) bool {
		a, b := files[i], files[j]
		if q.Kind == "series" {
			if a.Season != b.Season {
				return a.Season < b.Season
			}
			if a.Episode != b.Episode {
				return a.Episode < b.Episode
			}
		}
		if q.Kind == "music" {
			if disc(a.Disc) != disc(b.Disc) {
				return disc(a.Disc) < disc(b.Disc)
			}
			if a.Number != b.Number {
				return a.Number < b.Number
			}
		}
		return a.Name < b.Name
	})
	byTitle := map[string][]d.File{}
	for _, f := range files {
		k := normalized(first(f.Title, f.Name))
		byTitle[k] = append(byTitle[k], f)
	}
	var focus *d.File
	for i := range files {
		if files[i].ID == q.Focus {
			focus = &files[i]
		}
	}
	var start *d.File
	if focus != nil {
		start = focus
	} else {
		for i := range files {
			if !files[i].Completed {
				start = &files[i]
				break
			}
		}
		if start == nil && len(files) > 0 {
			start = &files[0]
		}
	}
	if start != nil {
		label := "Lire"
		if start.Position > 0 && !start.Completed {
			label = "Reprendre"
		}
		if q.Kind == "music" && focus == nil {
			label += " l’album"
		}
		a := action("play", label, start.ID)
		a.Primary = true
		r.Actions = append(r.Actions, a)
	}
	if q.Work {
		label := "+ À voir"
		if q.Kind == "music" {
			label = "+ À écouter"
		}
		r.Actions = append(r.Actions, action("watchlist", label, ""), action("favorite", "♡ Favori", ""))
		r.Suggestions = d.Suggestions{Enabled: true, Title: "Si cette histoire vous plaît", Help: "Suggestions du fournisseur associé à cette œuvre."}
	}
	if q.Kind == "music" {
		r.Cover = "square"
		seen := map[int]bool{}
		add := func(i int) {
			if !seen[i] {
				seen[i] = true
				r.People = append(r.People, i)
			}
		}
		contributors := q.Contributors
		if focus != nil {
			contributors = nil
			for _, t := range q.Tracks {
				if f := match(t, byTitle, map[string]bool{}); f != nil && f.ID == focus.ID {
					contributors = t.People
					break
				}
			}
		}
		for _, i := range contributors {
			add(i)
		}
		if len(r.People) == 0 {
			for _, f := range files {
				if focus != nil && f.ID != focus.ID {
					continue
				}
				name := first(f.AlbumArtist, f.Artist)
				if focus != nil {
					name = f.Artist
				}
				for i, p := range q.People {
					if p == name {
						add(i)
					}
				}
			}
		}
		if len(files) > 0 {
			id := ""
			if focus != nil {
				id = focus.ID
			}
			r.Actions = append(r.Actions, action("enqueue", "À la file", id), action("playlist", "Ajouter à une playlist", id))
		} else {
			r.Actions = append(r.Actions, action("unavailable", "Demander le téléchargement · bientôt", ""))
		}
		if q.Work {
			r.Suggestions.Title = "Du même artiste, à explorer"
			r.Suggestions.Help = "Autres éditions et projets du fournisseur associé."
		}
	} else {
		if q.Work && (len(files) == 0 || q.Kind == "series") {
			label := "Demander le téléchargement"
			if q.Kind == "series" {
				label = "Demander la série"
			}
			r.Actions = append(r.Actions, action("request", label, ""))
		}
		if len(files) > 0 || q.Work {
			id := ""
			if len(files) > 0 {
				id = files[0].ID
			}
			r.Actions = append(r.Actions, action("poster", "Changer l’affiche", id))
			if id != "" {
				r.Actions = append(r.Actions, action("inspect", "Corriger l’association", id))
			}
		}
	}
	if q.Kind == "series" {
		r.EpisodeList = true
		return r
	}
	if q.Kind == "music" && focus == nil {
		s := d.Section{Key: "tracks", Title: "Les morceaux", Help: "Fichiers présents sur votre serveur. Associez une édition pour connaître les pistes manquantes.", Rows: []d.Row{}, Actions: []d.Action{}}
		if len(files) > 0 && files[0].Album != "" {
			s.Actions = append(s.Actions, action("match", "Corriger l’association", files[0].ID))
		}
		used := map[string]bool{}
		if len(q.Tracks) > 0 {
			s.Help = "Liste de l’édition associée. Les pistes absentes sont grisées."
		}
		for i, t := range q.Tracks {
			row := d.Row{Key: fmt.Sprintf("track-%d", i), Title: t.Title, Ordinal: fmt.Sprint(i + 1), Duration: t.Duration, Actions: []d.Action{action("unavailable", "Demander", "")}}
			if t.Number > 0 {
				row.Ordinal = fmt.Sprint(t.Number)
			}
			if t.Disc > 1 {
				row.Ordinal = fmt.Sprintf("%d.%s", t.Disc, row.Ordinal)
			}
			names := []string{}
			for _, j := range t.People {
				names = append(names, q.People[j])
			}
			row.Subtitle = strings.Join(names, " · ")
			if f := match(t, byTitle, used); f != nil {
				row.FileID = f.ID
				used[f.ID] = true
				row.Actions = []d.Action{action("open", "Ouvrir", f.ID), action("play", "Lire", f.ID)}
			}
			s.Rows = append(s.Rows, row)
		}
		other := d.Section{Key: "other", Title: "Autres fichiers de cet album", Help: "Ces fichiers ne sont pas rapprochés avec certitude des pistes de l’édition.", Rows: []d.Row{}, Actions: []d.Action{}}
		for i, f := range files {
			if used[f.ID] {
				continue
			}
			row := d.Row{Key: f.ID, FileID: f.ID, Title: first(f.Title, f.Name), Subtitle: f.Artist, Ordinal: fmt.Sprint(i + 1), Actions: []d.Action{action("open", "Ouvrir", f.ID), action("playlist", "+ Playlist", f.ID), action("play", "Lire", f.ID)}}
			if f.Number > 0 {
				row.Ordinal = fmt.Sprint(f.Number)
			}
			if len(q.Tracks) == 0 {
				s.Rows = append(s.Rows, row)
			} else {
				other.Rows = append(other.Rows, row)
			}
		}
		r.Sections = append(r.Sections, s)
		if len(other.Rows) > 0 {
			r.Sections = append(r.Sections, other)
		}
		return r
	}
	s := d.Section{Key: "files", Title: "Vos fichiers", Rows: []d.Row{}, Actions: []d.Action{}}
	for _, f := range files {
		if focus != nil && f.ID != focus.ID {
			continue
		}
		label, done := "Marquer vu", "Vu"
		if q.Kind == "music" {
			label, done = "Marquer écouté", "Écouté"
		}
		if f.Completed {
			if q.Kind == "music" {
				label = "Marquer non écouté"
			} else {
				label = "Marquer non vu"
			}
		}
		sub := "Prêt à lire"
		if f.Completed {
			sub = done
		} else if f.Position > 0 {
			sub = fmt.Sprintf("Reprendre à %d:%02d", int(f.Position)/60, int(f.Position)%60)
		}
		s.Rows = append(s.Rows, d.Row{Key: f.ID, FileID: f.ID, Title: first(f.Title, f.Name), Subtitle: sub, Actions: []d.Action{action("play", "Lire", f.ID), action("mark", label, f.ID), action("inspect", "Informations du fichier", f.ID)}})
	}
	if len(s.Rows) > 0 {
		r.Sections = append(r.Sections, s)
	}
	r.AlbumLink = focus != nil && q.Kind == "music"
	return r
}
