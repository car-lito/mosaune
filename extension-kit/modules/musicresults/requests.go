package musicresults

import (
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"strconv"
	"strings"
)

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func Plan(q d.Request) (d.RequestPlan, error) {
	p := d.RequestPlan{Path: "/release/", Decode: "releases", Params: map[string]string{}}
	if e := q.Validate(); e != nil {
		return p, e
	}
	switch q.Operation {
	case "discover":
		p.Params["query"] = "release:" + quote(q.Title) + " OR artist:" + quote(q.Title)
	case "match":
		p.Params["query"] = "release:" + quote(q.Title)
		if q.Artist != "" {
			p.Params["query"] += " AND artist:" + quote(q.Artist)
		}
	case "artist-releases":
		p.Params["query"] = "arid:" + q.ID
	case "upcoming":
		p.Params["query"] = "arid:" + q.ID + " AND date:[" + q.Date + " TO *]"
	case "release", "seed":
		p.Path += q.ID
		p.ID = q.ID
		p.Decode = q.Operation
		p.Params["inc"] = "artist-credits"
		if q.Operation == "release" {
			p.Params["inc"] = "recordings+artist-credits+labels+release-groups"
		}
	case "artist":
		p.Path = "/artist/" + q.ID
		p.ID = q.ID
		p.Decode = "artist"
	case "artists":
		p.Path = "/artist/"
		p.Decode = "artists"
		p.Params["query"] = "artist:" + quote(q.Title)
	}
	if p.Decode == "releases" {
		p.Params["limit"] = "20"
		p.Params["offset"] = strconv.Itoa((q.Page - 1) * 20)
	} else if p.Decode == "artists" {
		p.Params["limit"] = "8"
	}
	return p, p.Validate(q)
}
