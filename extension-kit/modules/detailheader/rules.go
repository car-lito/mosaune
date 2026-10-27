package detailheader

import (
	"fmt"
	d "mosaune/extensionkit/capabilities/detailheader/v1"
	"strings"
)

func first(values ...string) string {
	for _, s := range values {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return "Média"
}
func Project(q d.Query) d.Result {
	title := first(q.Title, q.Name)
	label := q.Kind
	if !q.StoredWork && !q.Track {
		title = first(q.GroupTitle, q.Title, q.Name)
	}
	switch q.Kind {
	case "films":
		label = "FILM"
	case "series":
		label = "SÉRIE"
		if !q.StoredWork {
			title = first(q.GroupTitle, q.Series, q.Title, q.Name)
		}
	case "music":
		label = "ALBUM"
		if q.Track {
			label = "MORCEAU"
			title = first(q.Title, q.Name)
		} else if !q.StoredWork {
			title = first(q.GroupTitle, q.Album, q.Title, q.Name)
		}
	}
	available := "À DÉCOUVRIR"
	if q.LocalCount > 0 {
		available = "DANS VOTRE COLLECTION"
	}
	r := d.Result{Title: title, Eyebrow: label + " · " + available, Facts: []d.Fact{}, Tagline: q.Tagline}
	if q.Year > 0 {
		r.Facts = append(r.Facts, d.Fact{Key: "year", Text: fmt.Sprint(q.Year)})
	}
	if q.Runtime > 0 {
		r.Facts = append(r.Facts, d.Fact{Key: "runtime", Text: fmt.Sprintf("%d min", q.Runtime)})
	}
	if len(q.Genres) > 0 {
		r.Facts = append(r.Facts, d.Fact{Key: "genres", Text: strings.Join(q.Genres, " · ")})
	}
	if q.VoteCount > 0 {
		provider := q.Provider
		if provider == "tmdb" {
			provider = "TMDB"
		}
		note := fmt.Sprintf("%d votes", q.VoteCount)
		if provider != "" {
			note = provider + " · " + note
		}
		r.Facts = append(r.Facts, d.Fact{Key: "rating", Text: fmt.Sprintf("★ %.1f / 10", q.Rating), Note: note})
	}
	if q.Kind == "music" && q.Edition != "" {
		r.Footnote = "Édition : " + q.Edition
	}
	return r
}
