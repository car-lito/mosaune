package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWireRoundTripAndGuards(t *testing.T) {
	f, err := wireFormat("sonarr")
	if err != nil {
		t.Fatal(err)
	}
	f.Fields["resource"]["id"] = "pk"
	f.Fields["resource"]["seasons"] = "chapters"
	f.Fields["season"]["seasonNumber"] = "index"
	original := []byte(`{"pk":12,"tvdbId":84,"path":"/keep","chapters":[{"index":1,"monitored":true,"extra":{"n":9007199254740993}}],"foreign":9007199254740995}`)
	canonical, err := f.Convert("sonarr", "create", false, original)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(canonical, []byte(`"externalId":84`)) || !bytes.Contains(canonical, []byte(`"seasonNumber":1`)) {
		t.Fatal(string(canonical))
	}
	wire, err := f.Convert("sonarr", "update", true, canonical)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	decode := func(raw []byte, out *any) {
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	decode(original, &a)
	decode(wire, &b)
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if !bytes.Equal(aa, bb) {
		t.Fatal("existing data changed", string(wire))
	}
	for _, raw := range []string{
		`{"pk":1,"id":2}`, `{"pk":1,"ID":2}`, `{"pk":1,"pk":2}`, `{} {}`, `{"chapters":{}}`,
		`{"chapters":[{"index":1,"seasonNumber":2}]}`, strings.Repeat("[", 50) + "0" + strings.Repeat("]", 50),
	} {
		if _, err := f.Convert("sonarr", "create", false, []byte(raw)); err == nil {
			t.Fatalf("ambiguous JSON accepted: %.100s", raw)
		}
	}
	if _, err := f.Convert("sonarr", "status", true, []byte(`{}`)); err == nil {
		t.Fatal("GET payload")
	}
	f.Fields["resource"]["title"] = "pk"
	if f.Validate("sonarr") == nil {
		t.Fatal("duplicate destination")
	}
}

func TestWireOperations(t *testing.T) {
	for _, connector := range []string{"radarr", "sonarr"} {
		f, err := wireFormat(connector)
		if err != nil {
			t.Fatal(err)
		}
		inputs := map[string]string{"status": `{"appName":"fixture"}`, "profiles": `[{"id":1,"name":"A"}]`, "roots": `[{"id":1,"path":"/a"}]`, "find": `[]`, "lookup": `[]`, "create": `{"id":1}`, "queue": `{"totalRecords":1,"records":[{"movieId":12,"seriesId":12,"sizeleft":0}]}`}
		if connector == "sonarr" {
			inputs["episodes"] = `[{"id":1,"episodeNumber":2}]`
			inputs["episodefiles"] = `[{"id":1,"relativePath":"e.mkv"}]`
		}
		for op, raw := range inputs {
			if _, err := f.Convert(connector, op, false, []byte(raw)); err != nil {
				t.Fatal(connector, op, err)
			}
		}
		for name, command := range f.Commands {
			raw, _ := json.Marshal(map[string]any{"name": name})
			out, err := f.Convert(connector, "search", true, raw)
			if err != nil || !bytes.Contains(out, []byte(command)) {
				t.Fatal("command not declared", err)
			}
		}
		if _, err := f.Convert(connector, "search", true, []byte(`{"name":"unknown"}`)); err == nil {
			t.Fatal("unknown search action")
		}
	}
}
