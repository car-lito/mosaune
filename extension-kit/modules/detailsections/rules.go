package detailsections

import (
	"fmt"
	d "mosaune/extensionkit/capabilities/detailsections/v1"
	"strings"
)

func Project(q d.Query) d.Result {
	r := d.Result{Blocks: []d.Block{}}
	if q.Kind != "films" && q.Kind != "series" {
		return r
	}
	overview := q.Overview
	if strings.TrimSpace(overview) == "" {
		overview = "Le résumé n’est pas encore disponible. Vous pouvez identifier cette œuvre depuis les informations du fichier."
	}
	r.Blocks = append(r.Blocks, d.Block{Key: "story", Region: "main", Kind: "paragraph", Title: "L’histoire", Text: overview})
	if len(q.Trailers) > 0 {
		b := d.Block{Key: "trailers", Region: "main", Kind: "trailers", Title: "Bandes-annonces", Text: "Les vidéos s’ouvrent dans un nouvel onglet."}
		for i, t := range q.Trailers {
			label := strings.ToUpper(t.Language)
			if t.Language == "fr" {
				label = "Français"
			}
			if label == "" {
				label = "Langue non précisée"
			}
			b.References = append(b.References, d.Reference{Index: i, Label: label})
		}
		r.Blocks = append(r.Blocks, b)
	}
	b := d.Block{Key: "facts", Region: "aside", Kind: "facts", Title: "En quelques mots", Fields: []d.Field{}}
	add := func(label, value string) {
		if value != "" {
			b.Fields = append(b.Fields, d.Field{Label: label, Value: value})
		}
	}
	add("Première sortie", q.ReleaseDate)
	add("Statut", q.Status)
	add("Production", strings.Join(q.Countries, ", "))
	add("Langues d’origine", strings.Join(q.Languages, ", "))
	available := "Absent du serveur"
	if q.LocalCount > 0 {
		available = fmt.Sprintf("%d fichier(s) disponible(s)", q.LocalCount)
	}
	add("Dans Mosaune", available)
	r.Blocks = append(r.Blocks, b)
	if len(q.People) > 0 {
		b := d.Block{Key: "cast", Region: "full", Kind: "people", Title: "Les visages de l’histoire"}
		for i := range q.People {
			b.References = append(b.References, d.Reference{Index: i})
		}
		r.Blocks = append(r.Blocks, b)
	} else if len(q.Cast) > 0 {
		r.Blocks = append(r.Blocks, d.Block{Key: "cast", Region: "full", Kind: "paragraph", Title: "Distribution", Text: strings.Join(q.Cast, " · ")})
	}
	return r
}
