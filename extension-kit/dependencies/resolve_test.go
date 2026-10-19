package dependencies

import (
	"errors"
	"mosaune/extensionkit/protocol"
	"reflect"
	"testing"
)

func TestSharedCapabilitySelection(t *testing.T) {
	owner := protocol.Descriptor{ID: "test.films", Version: "1.0.0", Capabilities: []string{"media.films.v1"}, Dependencies: []string{"playback.video.v1"}}
	provider := func(id string, enabled bool) Candidate {
		return Candidate{Descriptor: protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{"playback.video.v1"}}, Enabled: enabled}
	}
	a, b := provider("test.native", true), provider("test.alternative", true)
	for _, tc := range []struct {
		name       string
		candidates []Candidate
		choices    map[string]string
		code       string
	}{
		{"missing", nil, nil, "missing"},
		{"disabled", []Candidate{provider("test.native", false)}, nil, "disabled"},
		{"ambiguous", []Candidate{a, b}, nil, "ambiguous"},
		{"unknown choice", []Candidate{a}, map[string]string{"playback.video.v1": "test.absent"}, "invalid_choice"},
		{"undeclared", []Candidate{a}, map[string]string{"secrets.read.v1": "test.native"}, "invalid_choice"},
		{"disabled choice", []Candidate{a, provider("test.alternative", false)}, map[string]string{"playback.video.v1": "test.alternative"}, "disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(owner, tc.candidates, tc.choices)
			var issue *Issue
			if !errors.As(err, &issue) || issue.Code != tc.code {
				t.Fatal(err)
			}
		})
	}
	for _, list := range [][]Candidate{{a, b}, {b, a}} {
		got, err := Resolve(owner, list, map[string]string{"playback.video.v1": "test.alternative"})
		if err != nil || got["playback.video.v1"] != "test.alternative" {
			t.Fatal(got, err)
		}
		_, err = Resolve(owner, list, nil)
		var issue *Issue
		if !errors.As(err, &issue) || !reflect.DeepEqual(issue.Candidates, []string{"test.alternative", "test.native"}) {
			t.Fatal(err)
		}
	}
	got, err := Resolve(owner, []Candidate{a}, nil)
	if err != nil || got["playback.video.v1"] != "test.native" {
		t.Fatal(got, err)
	}
	if _, err = Resolve(owner, []Candidate{a, a}, nil); err == nil {
		t.Fatal("duplicate IDs accepted")
	}
	wrong := a
	wrong.Descriptor.Capabilities = []string{"playback.video.v2"}
	if _, err = Resolve(owner, []Candidate{wrong}, nil); err == nil {
		t.Fatal("incompatible API accepted")
	}
}

func TestDependencyDescriptorBoundaries(t *testing.T) {
	base := protocol.Descriptor{ID: "test.module", Version: "1.0.0", Capabilities: []string{"media.films.v1"}}
	for _, deps := range [][]string{{"no-version"}, {"media.films.v1"}, {"playback.video.v1", "playback.video.v1"}, make([]string, 17)} {
		d := base
		d.Dependencies = deps
		if d.Validate() == nil {
			t.Fatal("invalid dependencies accepted", deps)
		}
	}
	// Old descriptors are unchanged; no hidden default player is required.
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
}
