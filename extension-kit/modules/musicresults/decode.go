package musicresults

import (
	"encoding/json"
	"errors"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	presentation "mosaune/extensionkit/presentation/v1"
	"strconv"
	"strings"
)

type Item = d.Item
type Contributor = d.Contributor
type CatalogTrack = d.CatalogTrack
type Result = d.Result

var mbID = d.ID

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

type mbCredit struct {
	Name   string
	Join   string `json:"joinphrase"`
	Artist struct {
		ID   string
		Name string
	}
}
type mbRelease struct {
	ID             string
	Title          string
	Date           string
	Country        string
	Disambiguation string
	Group          struct {
		ID          string
		PrimaryType string `json:"primary-type"`
	} `json:"release-group"`
	Credits []mbCredit `json:"artist-credit"`
	Media   []struct {
		Position int
		Format   string
		Tracks   []struct {
			ID        string
			Title     string
			Position  int
			Length    int
			Credits   []mbCredit `json:"artist-credit"`
			Recording struct {
				Title   string
				Length  int
				Credits []mbCredit `json:"artist-credit"`
			}
		}
	}
	Cover struct{ Front bool } `json:"cover-art-archive"`
}

func mbPeople(credits []mbCredit) []Contributor {
	out := []Contributor{}
	for _, c := range credits {
		if len(out) >= 30 {
			break
		}
		if mbID.MatchString(c.Artist.ID) {
			out = append(out, Contributor{ID: c.Artist.ID, Name: truncate(c.Artist.Name, 200), Role: "Artiste"})
		}
	}
	return out
}
func mbArtist(credits []mbCredit) string {
	var b strings.Builder
	for _, c := range credits {
		name := c.Name
		if name == "" {
			name = c.Artist.Name
		}
		b.WriteString(name + c.Join)
	}
	return truncate(b.String(), 500)
}
func mbMetadata(r mbRelease) Item {
	year := 0
	if len(r.Date) >= 4 {
		year, _ = strconv.Atoi(r.Date[:4])
	}
	ref := r.ID
	if mbID.MatchString(r.Group.ID) {
		ref += "/" + r.Group.ID
	}
	label := map[string]string{"Album": "Album", "Single": "Single", "EP": "EP", "Broadcast": "Émission"}[r.Group.PrimaryType]
	if label == "" {
		label = "Sortie musicale"
	}
	return Item{Presentation: &presentation.Result{Version: 1, Label: label, Preview: &presentation.Preview{AspectRatio: 1, Shape: "rectangle", Fit: "cover"}}, Title: truncate(r.Title, 500), Album: truncate(r.Title, 500), Artist: mbArtist(r.Credits), AlbumArtist: mbArtist(r.Credits), Year: year, ReleaseDate: r.Date, ProviderID: r.ID, Contributors: mbPeople(r.Credits), ArtworkRef: ref, Edition: truncate(strings.Trim(r.Country+" · "+r.Date+" · "+r.Disambiguation, " ·"), 400)}
}

func Decode(q d.Query) (Result, error) {
	if e := q.Validate(); e != nil {
		return Result{}, e
	}
	out := Result{Pages: 1}
	switch q.Operation {
	case "release":
		var r mbRelease
		if e := json.Unmarshal(q.Body, &r); e != nil {
			return out, e
		}
		if r.ID != q.ID {
			return out, errors.New("release identity mismatch")
		}
		return releaseDetails(r)
	case "releases":
		var body struct {
			Count    int
			Releases []mbRelease
		}
		if e := json.Unmarshal(q.Body, &body); e != nil {
			return out, e
		}
		out.Pages = max(1, min(50, (body.Count+19)/20))
		seen := map[string]bool{}
		for _, r := range body.Releases {
			if mbID.MatchString(r.ID) && strings.TrimSpace(r.Title) != "" && !seen[r.ID] {
				out.Items = append(out.Items, mbMetadata(r))
				seen[r.ID] = true
			}
			if len(out.Items) >= 20 {
				break
			}
		}
	case "seed":
		var r mbRelease
		if e := json.Unmarshal(q.Body, &r); e != nil {
			return out, e
		}
		if r.ID != "" && r.ID != q.ID {
			return out, errors.New("seed identity mismatch")
		}
		if len(r.Credits) > 0 && mbID.MatchString(r.Credits[0].Artist.ID) {
			out.ArtistID = r.Credits[0].Artist.ID
		}
	case "artist":
		var a struct{ ID, Name, Country, Disambiguation string }
		if e := json.Unmarshal(q.Body, &a); e != nil {
			return out, e
		}
		if a.ID != "" && a.ID != q.ID {
			return out, errors.New("artist identity mismatch")
		}
		out.Name = truncate(a.Name, 300)
		out.Detail = truncate(strings.Trim(a.Country+" · "+a.Disambiguation, " ·"), 500)
	case "artists":
		var a struct {
			Artists []struct{ ID, Name, Disambiguation string }
		}
		if e := json.Unmarshal(q.Body, &a); e != nil {
			return out, e
		}
		for _, v := range a.Artists {
			if mbID.MatchString(v.ID) {
				out.Artists = append(out.Artists, Contributor{ID: v.ID, Name: truncate(v.Name, 200), Role: truncate(v.Disambiguation, 300)})
			}
			if len(out.Artists) >= 8 {
				break
			}
		}
	}
	return out, out.Validate(q)
}
func releaseDetails(r mbRelease) (Result, error) {
	m := mbMetadata(r)
	m.DetailVersion = 2
	m.Tracks = []CatalogTrack{}
	for _, disc := range r.Media {
		for _, track := range disc.Tracks {
			if len(m.Tracks) >= 200 {
				m.Edition += " · liste limitée à 200 pistes"
				return Result{Items: []Item{m}, Pages: 1}, nil
			}
			title := track.Title
			if title == "" {
				title = track.Recording.Title
			}
			credits := track.Credits
			if len(credits) == 0 {
				credits = track.Recording.Credits
			}
			if len(credits) == 0 {
				credits = r.Credits
			}
			length := track.Length
			if length == 0 {
				length = track.Recording.Length
			}
			m.Tracks = append(m.Tracks, CatalogTrack{ID: track.ID, Title: truncate(title, 500), Disc: disc.Position, Number: track.Position, Duration: length / 1000, Artists: mbPeople(credits)})
		}
	}
	return Result{Items: []Item{m}, Pages: 1}, nil
}
