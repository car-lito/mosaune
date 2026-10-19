package registry

import (
	"encoding/json"
	"errors"
	"mosaune/extensionkit/dependencies"
	"mosaune/extensionkit/protocol"
	"reflect"
	"strings"
	"testing"
)

func entry(id, cap string, deps ...string) Entry {
	return Entry{Descriptor: protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{cap}, Dependencies: deps}, CodeSHA256: strings.Repeat("a", 64)}
}
func apply(t *testing.T, s Snapshot, c Change) Snapshot {
	t.Helper()
	p, err := Preview(s, c)
	if err != nil {
		t.Fatal(err)
	}
	return p.After
}
func install(t *testing.T, s Snapshot, e Entry) Snapshot {
	return apply(t, s, Change{Operation: "install", ID: e.Descriptor.ID, Entry: &e})
}

func TestPlansPinSharedProvidersAndProtectConsumers(t *testing.T) {
	s := Empty()
	s = install(t, s, entry("test.player", "playback.video.v1"))
	s = install(t, s, entry("test.films", "media.films.v1", "playback.video.v1"))
	s = install(t, s, entry("test.series", "media.series.v1", "playback.video.v1"))
	for _, id := range []string{"test.player", "test.films", "test.series"} {
		s = apply(t, s, Change{Operation: "enable", ID: id})
	}
	before, _ := json.Marshal(s)
	for _, op := range []string{"disable", "remove"} {
		_, err := Preview(s, Change{Operation: op, ID: "test.player"})
		var dep *DependentsError
		if !errors.As(err, &dep) || !reflect.DeepEqual(dep.Dependents, []string{"test.films", "test.series"}) {
			t.Fatal(err)
		}
	}
	same, _ := json.Marshal(s)
	if string(before) != string(same) {
		t.Fatal("failed preview mutated source")
	}
	s = install(t, s, entry("test.alternative", "playback.video.v1"))
	s = apply(t, s, Change{Operation: "enable", ID: "test.alternative"})
	for _, e := range s.Entries {
		if e.Descriptor.ID == "test.films" && e.Choices["playback.video.v1"] != "test.player" {
			t.Fatal("choice stolen")
		}
	}
	s = apply(t, s, Change{Operation: "remove", ID: "test.films"})
	if _, err := Preview(s, Change{Operation: "remove", ID: "test.player"}); err == nil {
		t.Fatal("series lost provider")
	}
	p, err := Preview(s, Change{Operation: "disable", ID: "test.player", Cascade: true})
	if err != nil || !reflect.DeepEqual(p.Disabled, []string{"test.series", "test.player"}) {
		t.Fatal(p, err)
	}
	s = p.After
	s = apply(t, s, Change{Operation: "choose", ID: "test.series", Choices: map[string]string{"playback.video.v1": "test.alternative"}})
	s = apply(t, s, Change{Operation: "enable", ID: "test.series"})
	s = apply(t, s, Change{Operation: "remove", ID: "test.player"})
	if len(s.Entries) != 2 {
		t.Fatal(s)
	}
}

func TestRegistryAmbiguityCyclesAndUpdates(t *testing.T) {
	s := Empty()
	for _, id := range []string{"test.first", "test.second"} {
		s = install(t, s, entry(id, "playback.video.v1"))
		s = apply(t, s, Change{Operation: "enable", ID: id})
	}
	s = install(t, s, entry("test.films", "media.films.v1", "playback.video.v1"))
	_, err := Preview(s, Change{Operation: "enable", ID: "test.films"})
	var issue *dependencies.Issue
	if !errors.As(err, &issue) || issue.Code != "ambiguous" {
		t.Fatal(err)
	}
	s = apply(t, s, Change{Operation: "enable", ID: "test.films", Choices: map[string]string{"playback.video.v1": "test.first"}})
	if _, err = Preview(s, Change{Operation: "choose", ID: "test.films", Choices: map[string]string{"playback.video.v1": "test.second"}}); err == nil {
		t.Fatal("live rebind allowed")
	}
	upgrade := entry("test.first", "playback.video.v2")
	upgrade.Descriptor.Version = "2.0.0"
	if _, err = Preview(s, Change{Operation: "update", ID: "test.first", Entry: &upgrade}); err == nil {
		t.Fatal("breaking update accepted")
	}
	upgrade = entry("test.first", "playback.video.v1")
	upgrade.Descriptor.Version = "1.1.0"
	s = apply(t, s, Change{Operation: "update", ID: "test.first", Entry: &upgrade})
	// Mutually selected dependencies are structurally valid but must not start.
	a, b := entry("test.a", "test.a.v1", "test.b.v1"), entry("test.b", "test.b.v1", "test.a.v1")
	a.Enabled = true
	b.Enabled = true
	a.Choices = map[string]string{"test.b.v1": "test.b"}
	b.Choices = map[string]string{"test.a.v1": "test.a"}
	if (Snapshot{Schema: 1, Entries: []Entry{a, b}}).Validate() == nil {
		t.Fatal("cycle accepted")
	}
	for _, e := range s.Entries {
		if e.Enabled && len(e.Descriptor.Dependencies) > 0 {
			e.Choices = nil
			bad := clone(s)
			for i := range bad.Entries {
				if bad.Entries[i].Descriptor.ID == e.Descriptor.ID {
					bad.Entries[i] = e
				}
			}
			if bad.Validate() == nil {
				t.Fatal("unbound active module accepted")
			}
		}
	}
}

func TestRegistryLimitsAndIsolation(t *testing.T) {
	e := entry("test.module", "test.module.v1")
	s := install(t, Empty(), e)
	e.Descriptor.Capabilities[0] = "test.changed.v1"
	if s.Entries[0].Descriptor.Capabilities[0] != "test.module.v1" {
		t.Fatal("descriptor aliased")
	}
	for _, mutate := range []func(*Snapshot){func(s *Snapshot) { s.Schema = 2 }, func(s *Snapshot) { s.Revision = -1 }, func(s *Snapshot) { s.Entries[0].CodeSHA256 = "bad" }, func(s *Snapshot) { s.Entries = append(s.Entries, s.Entries[0]) }, func(s *Snapshot) { s.Entries[0].Choices = map[string]string{"secrets.all.v1": "test.secret"} }} {
		bad := clone(s)
		mutate(&bad)
		if bad.Validate() == nil {
			t.Fatal("bad registry accepted")
		}
	}
	if _, err := Preview(s, Change{Operation: "install", ID: "test.module", Entry: &e}); err == nil {
		t.Fatal("duplicate install accepted")
	}
	if _, err := Preview(s, Change{Operation: "unknown", ID: "test.module"}); err == nil {
		t.Fatal("unknown mutation accepted")
	}
}
