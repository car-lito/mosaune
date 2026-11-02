package musicresults

import (
	"encoding/json"
	"errors"
	"html"
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

type visualState struct{ Stage, File string }

var htmlTag = regexp.MustCompile(`<[^>]*>`)

func plainCredit(s string) string {
	return truncate(strings.TrimSpace(html.UnescapeString(htmlTag.ReplaceAllString(s, ""))), 500)
}

func Visual(q d.VisualQuery) (d.VisualStep, error) {
	if e := q.Validate(); e != nil {
		return d.VisualStep{}, e
	}
	var state visualState
	if len(q.State) > 0 {
		if e := json.Unmarshal(q.State, &state); e != nil {
			return d.VisualStep{}, e
		}
	}
	step, e := visual(q, state)
	if e != nil {
		return step, e
	}
	return step, step.Validate(q)
}
func visual(q d.VisualQuery, state visualState) (d.VisualStep, error) {
	missing := d.VisualStep{Missing: true}
	fetch := func(source, path, stage, file string, params map[string]string, image bool) (d.VisualStep, error) {
		next, _ := json.Marshal(visualState{stage, file})
		return d.VisualStep{Fetch: &d.VisualFetch{Source: source, Path: path, Params: params, Image: image}, State: next}, nil
	}
	commons := func(file string) (d.VisualStep, error) {
		return fetch("commons", "/w/api.php", "commons", file, map[string]string{"action": "query", "format": "json", "prop": "imageinfo", "titles": "File:" + file, "iiprop": "url|extmetadata", "iiextmetadatafilter": "Artist|LicenseShortName", "iiurlwidth": "320", "redirects": "1"}, false)
	}
	cover := func(resource, id, stage string) (d.VisualStep, error) {
		return fetch("coverart", "/"+resource+"/"+id+"/front-500", stage, "", nil, true)
	}
	switch state.Stage {
	case "":
		if q.Operation == "artist" {
			return fetch("musicbrainz", "/artist/"+q.Ref, "artist-links", "", map[string]string{"inc": "url-rels"}, false)
		}
		if strings.HasPrefix(q.Ref, "commons:") {
			return commons(strings.TrimPrefix(q.Ref, "commons:"))
		}
		return cover("release", strings.Split(q.Ref, "/")[0], "edition-cover")
	case "edition-cover":
		if q.Status != 404 {
			return missing, nil
		}
		parts := strings.Split(q.Ref, "/")
		if len(parts) == 2 {
			return cover("release-group", parts[1], "group-cover")
		}
		return fetch("musicbrainz", "/release/"+parts[0], "release-group", "", map[string]string{"inc": "release-groups"}, false)
	case "release-group":
		var v struct {
			Group struct{ ID string } `json:"release-group"`
		}
		if json.Unmarshal(q.Body, &v) != nil {
			return d.VisualStep{}, errors.New("invalid release group reply")
		}
		if !d.ID.MatchString(v.Group.ID) {
			return missing, nil
		}
		return cover("release-group", v.Group.ID, "group-cover")
	case "group-cover", "image":
		return missing, nil
	case "artist-links":
		var v struct {
			Relations []struct {
				Type string
				URL  struct{ Resource string }
			}
		}
		if json.Unmarshal(q.Body, &v) != nil {
			return d.VisualStep{}, errors.New("invalid artist relations")
		}
		for _, rel := range v.Relations {
			u, e := url.Parse(rel.URL.Resource)
			if e != nil || rel.Type != "wikidata" || u.User != nil || u.Port() != "" || (u.Scheme != "https" && u.Scheme != "http") || (u.Hostname() != "www.wikidata.org" && u.Hostname() != "wikidata.org") || !strings.HasPrefix(u.Path, "/wiki/") {
				continue
			}
			entity := strings.TrimPrefix(u.Path, "/wiki/")
			if d.EntityID.MatchString(entity) {
				return fetch("wikidata", "/w/api.php", "claims", "", map[string]string{"action": "wbgetclaims", "format": "json", "entity": entity, "property": "P18"}, false)
			}
		}
		return missing, nil
	case "claims":
		var v struct {
			Claims map[string][]struct {
				Rank string
				Main struct {
					Data struct{ Value string } `json:"datavalue"`
				} `json:"mainsnak"`
			}
		}
		if json.Unmarshal(q.Body, &v) != nil {
			return d.VisualStep{}, errors.New("invalid image claims")
		}
		for _, claim := range v.Claims["P18"] {
			if claim.Rank != "deprecated" && d.SafeCommonsFile(claim.Main.Data.Value) {
				return commons(claim.Main.Data.Value)
			}
		}
		return missing, nil
	case "commons":
		if !d.SafeCommonsFile(state.File) {
			return missing, nil
		}
		var v struct {
			Query struct {
				Pages map[string]struct {
					Info []struct {
						Thumb    string `json:"thumburl"`
						URL      string
						Metadata map[string]struct{ Value any } `json:"extmetadata"`
					} `json:"imageinfo"`
				}
			}
		}
		if json.Unmarshal(q.Body, &v) != nil {
			return d.VisualStep{}, errors.New("invalid image information")
		}
		keys := make([]string, 0, len(v.Query.Pages))
		for k := range v.Query.Pages {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, info := range v.Query.Pages[key].Info {
				imageURL := info.Thumb
				if imageURL == "" {
					imageURL = info.URL
				}
				if !d.SafeUpload(imageURL) {
					continue
				}
				credit, _ := info.Metadata["Artist"].Value.(string)
				license, _ := info.Metadata["LicenseShortName"].Value.(string)
				photo := &d.Photo{File: state.File, URL: imageURL, Source: "https://commons.wikimedia.org/wiki/File:" + url.PathEscape(state.File), Credit: plainCredit(credit), License: plainCredit(license)}
				if q.Operation == "artist" {
					return d.VisualStep{Photo: photo}, nil
				}
				return fetch("image", imageURL, "image", "", nil, true)
			}
		}
		return missing, nil
	default:
		return d.VisualStep{}, errors.New("unknown image continuation")
	}
}
