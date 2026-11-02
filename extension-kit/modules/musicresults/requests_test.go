package musicresults

import (
	d "mosaune/extensionkit/capabilities/musicresults/v1"
	"strings"
	"testing"
)

func TestRequestPlans(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	for _, op := range []string{"discover", "match", "artists", "release", "seed", "artist", "artist-releases", "upcoming"} {
		q := d.Request{Operation: op, Title: `A "quoted" album`, Artist: "Artist", ID: id, Date: "2026-11-02", Page: 3}
		p, e := Plan(q)
		if e != nil || p.Validate(q) != nil {
			t.Fatal(op, p, e)
		}
		if p.Decode == "releases" && p.Params["offset"] != "40" {
			t.Fatal("pagination", op, p)
		}
		if op == "discover" && !strings.Contains(p.Params["query"], `\"quoted\"`) {
			t.Fatal("query escaping", p)
		}
		if op == "match" && !strings.Contains(p.Params["query"], `AND artist:"Artist"`) {
			t.Fatal("artist filter", p)
		}
		if op == "upcoming" && !strings.Contains(p.Params["query"], "date:[2026-11-02 TO *]") {
			t.Fatal("caller date lost", p)
		}
	}
}

func TestRequestPlanBoundary(t *testing.T) {
	q := d.Request{Operation: "discover", Title: "Album", Page: 1}
	for _, mutate := range []func(*d.RequestPlan){
		func(p *d.RequestPlan) { p.Path = "https://example.org/" },
		func(p *d.RequestPlan) { p.Path = "/release/../artist/" },
		func(p *d.RequestPlan) { p.Params["token"] = "x" },
		func(p *d.RequestPlan) { p.Params["limit"] = "1000" },
		func(p *d.RequestPlan) { p.Params["offset"] = "-1" },
		func(p *d.RequestPlan) { p.Params["query"] = strings.Repeat("x", 4097) },
		func(p *d.RequestPlan) { p.Decode = "artist" },
		func(p *d.RequestPlan) { p.ID = "11111111-1111-1111-1111-111111111111" },
	} {
		p, _ := Plan(q)
		mutate(&p)
		if p.Validate(q) == nil {
			t.Fatal("unsafe plan accepted", p)
		}
	}
	for _, q := range []d.Request{
		{Operation: "unknown", Title: "Album", Page: 1}, {Operation: "discover", Title: "Album", Page: 51},
		{Operation: "release", ID: "../artist", Page: 1}, {Operation: "discover", Title: strings.Repeat("x", 1201), Page: 1},
		{Operation: "upcoming", ID: "11111111-1111-1111-1111-111111111111", Date: "2026-02-31", Page: 1},
	} {
		if _, e := Plan(q); e == nil {
			t.Fatal("invalid input accepted", q)
		}
	}
}
