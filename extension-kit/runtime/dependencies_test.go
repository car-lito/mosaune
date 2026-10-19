package runtime

import (
	"context"
	"errors"
	"mosaune/extensionkit/dependencies"
	"mosaune/extensionkit/examples/demo"
	"mosaune/extensionkit/protocol"
	"os"
	"reflect"
	"testing"
	"time"
)

// These are real WASI modules using the catalogue demo. The playback capability
// names exercise lifecycle wiring only; these fixtures do not decode media.
func loadDependencyFixture(t *testing.T, e *Engine, id, capability string, deps ...string) {
	t.Helper()
	d := protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{capability}, Dependencies: deps}
	if err := e.Load(context.Background(), d, demo.Code); err != nil {
		t.Fatal(err)
	}
	// The engine must own a copy, including the new slice.
	if len(deps) > 0 {
		deps[0] = "test.mutated.v1"
	}
}
func TestSharedDependencyLifecycleAndReplacement(t *testing.T) {
	l := DefaultLimits()
	l.Modules = 8
	e := engine(t, l)
	loadDependencyFixture(t, e, "test.player", "playback.video.v1")
	loadDependencyFixture(t, e, "test.films", "metadata.search.v1", "playback.video.v1")
	loadDependencyFixture(t, e, "test.series", "media.series.v1", "playback.video.v1")
	if err := e.Enable("test.films"); err == nil {
		t.Fatal("disabled dependency accepted")
	}
	for _, id := range []string{"test.player", "test.films", "test.series"} {
		if err := e.Enable(id); err != nil {
			t.Fatal(id, err)
		}
	}
	bindings, err := e.Bindings("test.films")
	if err != nil || bindings["playback.video.v1"] != "test.player" {
		t.Fatal(bindings, err)
	}
	bindings["playback.video.v1"] = "tamper"
	for _, mutate := range []func(string) error{e.Disable, e.Unload} {
		var blocked *DependentsError
		if err = mutate("test.player"); !errors.As(err, &blocked) || !reflect.DeepEqual(blocked.Dependents, []string{"test.films", "test.series"}) {
			t.Fatal(err)
		}
	}
	if _, err = e.Call(context.Background(), "test.films", request("metadata.search.v1", "search", `{"kind":"films","title":"Voyage"}`)); err != nil {
		t.Fatal("blocked deletion changed consumer", err)
	}
	if err = e.Unload("test.films"); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Bindings("test.player"); err != nil {
		t.Fatal("shared provider was removed", err)
	}
	loadDependencyFixture(t, e, "test.alternative", "playback.video.v1")
	if err = e.Enable("test.alternative"); err != nil {
		t.Fatal(err)
	}
	// Installing a second provider cannot silently steal an existing binding.
	if err = e.Enable("test.series"); err != nil {
		t.Fatal(err)
	}
	if err = e.EnableWithProviders("test.series", map[string]string{"playback.video.v1": "test.alternative"}); err == nil {
		t.Fatal("live binding silently changed")
	}
	if err = e.Disable("test.series"); err != nil {
		t.Fatal(err)
	}
	var issue *dependencies.Issue
	if err = e.Enable("test.series"); !errors.As(err, &issue) || issue.Code != "ambiguous" {
		t.Fatal(err)
	}
	if err = e.EnableWithProviders("test.series", map[string]string{"playback.video.v1": "test.alternative"}); err != nil {
		t.Fatal(err)
	}
	if err = e.Unload("test.player"); err != nil {
		t.Fatal("old provider still retained", err)
	}
	if _, err = e.DisableCascade("test.alternative"); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Bindings("test.series"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("consumer remained active", err)
	}
	if err = e.Enable("test.series"); err == nil {
		t.Fatal("consumer bypassed disabled dependency")
	}
}

func TestDependencyCascadeAndCycleCannotActivate(t *testing.T) {
	l := DefaultLimits()
	l.Modules = 8
	e := engine(t, l)
	loadDependencyFixture(t, e, "test.base", "test.base.v1")
	loadDependencyFixture(t, e, "test.middle", "test.middle.v1", "test.base.v1")
	loadDependencyFixture(t, e, "test.leaf", "metadata.search.v1", "test.middle.v1")
	for _, id := range []string{"test.base", "test.middle", "test.leaf"} {
		if err := e.Enable(id); err != nil {
			t.Fatal(err)
		}
	}
	impact, err := e.Dependents("test.base")
	if err != nil || !reflect.DeepEqual(impact, []string{"test.leaf", "test.middle"}) {
		t.Fatal(impact, err)
	}
	order, err := e.DisableCascade("test.base")
	if err != nil || !reflect.DeepEqual(order, []string{"test.leaf", "test.middle", "test.base"}) {
		t.Fatal(order, err)
	}
	if _, err = e.Call(context.Background(), "test.leaf", request("metadata.search.v1", "search", `{}`)); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	loadDependencyFixture(t, e, "test.cycle-a", "test.cycle-a.v1", "test.cycle-b.v1")
	loadDependencyFixture(t, e, "test.cycle-b", "test.cycle-b.v1", "test.cycle-a.v1")
	for _, id := range []string{"test.cycle-a", "test.cycle-b"} {
		if err = e.Enable(id); err == nil {
			t.Fatal("cycle activated")
		}
	}
}

func TestDependencyCascadeCancelsRunningConsumer(t *testing.T) {
	l := DefaultLimits()
	l.Timeout = 5 * time.Second
	e := engine(t, l)
	loadDependencyFixture(t, e, "test.base", "test.base.v1")
	code, err := os.ReadFile("testdata/probe.wasm")
	if err != nil {
		t.Fatal(err)
	}
	d := protocol.Descriptor{ID: "test.consumer", Version: "1.0.0", Capabilities: []string{"test.probe.v1"}, Dependencies: []string{"test.base.v1"}}
	if err = e.Load(context.Background(), d, code); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"test.base", "test.consumer"} {
		if err = e.Enable(id); err != nil {
			t.Fatal(err)
		}
	}
	finished := make(chan error, 1)
	go func() {
		_, err := e.Call(context.Background(), "test.consumer", request("test.probe.v1", "loop", `{}`))
		finished <- err
	}()
	running := func() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.plugins["test.consumer"].running }
	deadline := time.Now().Add(2 * time.Second)
	for !running() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !running() {
		t.Fatal("consumer never started")
	}
	if _, err = e.DisableCascade("test.base"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"test.base", "test.consumer"} {
		if err = e.Enable(id); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case err = <-finished:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatal("old generation survived", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cascade did not cancel consumer")
	}
	if _, err = e.Call(context.Background(), "test.consumer", request("test.probe.v1", "count", `{}`)); err != nil {
		t.Fatal("new generation broken", err)
	}
}
