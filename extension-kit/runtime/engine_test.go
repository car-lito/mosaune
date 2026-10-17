package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/examples/demo"
	"mosaune/extensionkit/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func engine(t *testing.T, l Limits) *Engine {
	t.Helper()
	e, err := New(context.Background(), l)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}
func request(cap, op string, payload string) protocol.Request {
	return protocol.Request{Protocol: 1, ID: "test-1", Capability: cap, Operation: op, Payload: json.RawMessage(payload)}
}
func probe(t *testing.T, e *Engine) {
	t.Helper()
	code, err := os.ReadFile("testdata/probe.wasm")
	if err != nil {
		t.Fatal("build fixtures using scripts/build-lab.ps1:", err)
	}
	if err = e.Load(context.Background(), protocol.Descriptor{ID: "test.probe", Version: "1.0.0", Capabilities: []string{"test.probe.v1"}, Services: []protocol.ServiceRequirement{{Name: "test.api.v1", Operations: []string{"get"}}}}, code); err != nil {
		t.Fatal(err)
	}
	if err = e.Enable("test.probe"); err != nil {
		t.Fatal(err)
	}
}

func TestOfflineProviderLifecycle(t *testing.T) {
	e := engine(t, DefaultLimits())
	ctx := context.Background()
	d := demo.Descriptor()
	if err := e.Load(ctx, d, demo.Code); err != nil {
		t.Fatal(err)
	}
	r := request("metadata.search.v1", "search", `{"kind":"films","title":"Voyage"}`)
	if _, err := e.Call(ctx, d.ID, r); !errors.Is(err, ErrUnavailable) {
		t.Fatal("loaded extension auto-enabled", err)
	}
	if err := e.Enable(d.ID); err != nil {
		t.Fatal(err)
	}
	out, err := e.Call(ctx, d.ID, r)
	if err != nil {
		t.Fatal(err)
	}
	var search struct{ Items []struct{ ID, Title string } }
	if err = json.Unmarshal(out.Response.Result, &search); err != nil || len(search.Items) != 1 || search.Items[0].ID != "atelier:etoiles" {
		t.Fatal(string(out.Response.Result), err)
	}
	detail, err := e.Call(ctx, d.ID, request("metadata.details.v1", "details", `{"id":"atelier:etoiles"}`))
	if err != nil || !strings.Contains(string(detail.Response.Result), "astronome") {
		t.Fatal(detail, err)
	}
	missing, err := e.Call(ctx, d.ID, request("metadata.details.v1", "details", `{"id":"absent"}`))
	if err != nil || missing.Response.Error == nil || missing.Response.Error.Code != "not_found" {
		t.Fatal(missing, err)
	}
	if _, err = e.Call(ctx, d.ID, request("secrets.all.v1", "read", `{}`)); !errors.Is(err, ErrUnavailable) {
		t.Fatal("undeclared capability accepted", err)
	}
	_ = e.Disable(d.ID)
	if _, err = e.Call(ctx, d.ID, r); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	_ = e.Enable(d.ID)
	if _, err = e.Call(ctx, d.ID, r); err != nil {
		t.Fatal(err)
	}
	if err = e.Unload(d.ID); err != nil {
		t.Fatal(err)
	}
	if err = e.Enable(d.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if err = e.Load(ctx, d, demo.Code); err != nil {
		t.Fatal("cannot reload", err)
	}
}
func TestGuestCannotReadHostFilesOrEnvironment(t *testing.T) {
	e := engine(t, DefaultLimits())
	probe(t, e)
	sentinel := filepath.Join(t.TempDir(), "host-secret.txt")
	if err := os.WriteFile(sentinel, []byte("fixture-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOSAUNE_TEST_SECRET", "fixture-only")
	p, _ := json.Marshal(map[string]string{"path": sentinel})
	out, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "probe", string(p)))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]bool
	_ = json.Unmarshal(out.Response.Result, &result)
	for _, key := range []string{"readDenied", "writeDenied", "environmentEmpty"} {
		if !result[key] {
			t.Fatal(key, string(out.Response.Result))
		}
	}
	b, err := os.ReadFile(sentinel)
	if err != nil || string(b) != "fixture-secret" {
		t.Fatal("host file changed", err)
	}
}
func TestResourceFailuresDoNotPoisonNextInvocation(t *testing.T) {
	l := DefaultLimits()
	l.Timeout = 500 * time.Millisecond
	l.OutputBytes = 1024
	l.LogBytes = 1024
	e := engine(t, l)
	probe(t, e)
	for _, tc := range []struct {
		op   string
		want error
	}{{"loop", context.DeadlineExceeded}, {"panic", ErrExecution}, {"memory", ErrExecution}, {"output", ErrLimit}, {"logs", ErrLimit}, {"invalid", ErrProtocol}, {"extra", ErrProtocol}, {"wrong_id", ErrProtocol}} {
		t.Run(tc.op, func(t *testing.T) {
			_, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", tc.op, `{}`))
			// A Go guest's OOM diagnostics may themselves hit the stderr budget.
			if !errors.Is(err, tc.want) && !(tc.op == "memory" && errors.Is(err, ErrLimit)) {
				t.Fatalf("want %v got %v", tc.want, err)
			}
			out, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "count", `{}`))
			if err != nil || string(out.Response.Result) != `{"count":1}` {
				t.Fatal("state leaked or runner broken", out, err)
			}
		})
	}
}
func TestDisableCancelsRunningGenerationAndRejectsConcurrentWork(t *testing.T) {
	l := DefaultLimits()
	l.Timeout = 5 * time.Second
	e := engine(t, l)
	probe(t, e)
	finished := make(chan error, 1)
	go func() {
		_, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "loop", `{}`))
		finished <- err
	}()
	deadline := time.Now().Add(time.Second)
	running := func() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.plugins["test.probe"].running }
	for !running() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !running() {
		t.Fatal("call never started")
	}
	if _, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "count", `{}`)); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	_ = e.Disable("test.probe")
	_ = e.Enable("test.probe")
	select {
	case err := <-finished:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("disable did not interrupt guest")
	}
	if _, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "count", `{}`)); err != nil {
		t.Fatal(err)
	}
}
func TestCanceledContextAndClosedEngine(t *testing.T) {
	e := engine(t, DefaultLimits())
	probe(t, e)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Call(ctx, "test.probe", request("test.probe.v1", "count", `{}`)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_ = e.Close()
	if _, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "count", `{}`)); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestUnloadAndCloseInterruptRunningModule(t *testing.T) {
	for _, action := range []string{"unload", "close"} {
		t.Run(action, func(t *testing.T) {
			l := DefaultLimits()
			l.Timeout = 5 * time.Second
			e := engine(t, l)
			probe(t, e)
			result := make(chan error, 1)
			go func() {
				_, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "loop", `{}`))
				result <- err
			}()
			running := func() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.plugins["test.probe"].running }
			deadline := time.Now().Add(time.Second)
			for !running() && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if !running() {
				t.Fatal("call never started")
			}
			stopped := make(chan error, 1)
			go func() {
				if action == "unload" {
					stopped <- e.Unload("test.probe")
				} else {
					stopped <- e.Close()
				}
			}()
			select {
			case err := <-stopped:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("lifecycle operation blocked")
			}
			select {
			case err := <-result:
				if !errors.Is(err, ErrUnavailable) {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("guest still running")
			}
		})
	}
}

func TestModuleCountAndIdentityLimits(t *testing.T) {
	l := DefaultLimits()
	l.Modules = 1
	e := engine(t, l)
	d := demo.Descriptor()
	if err := e.Load(context.Background(), d, demo.Code); err != nil {
		t.Fatal(err)
	}
	if err := e.Load(context.Background(), d, demo.Code); err == nil {
		t.Fatal("duplicate module accepted")
	}
	d.ID = "atelier.second"
	if err := e.Load(context.Background(), d, demo.Code); err == nil {
		t.Fatal("module count exceeded")
	}
}
func TestLoadAndInputLimits(t *testing.T) {
	l := DefaultLimits()
	l.CodeBytes = 16
	l.Modules = 1
	e := engine(t, l)
	if err := e.Load(context.Background(), demo.Descriptor(), demo.Code); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	if err := e.Load(context.Background(), demo.Descriptor(), []byte("bad wasm")); err == nil {
		t.Fatal("bad wasm accepted")
	}
	e2 := engine(t, DefaultLimits())
	probe(t, e2)
	payload := `{"text":"` + strings.Repeat("a", 65<<10) + `"}`
	if _, err := e2.Call(context.Background(), "test.probe", request("test.probe.v1", "count", payload)); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}

// Minimal binaries exercise import/memory validation without running any code.
func wasmSection(id byte, b []byte) []byte { return append([]byte{id, byte(len(b))}, b...) }
func TestUnsupportedImportsAndMemoryMinimum(t *testing.T) {
	header := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	typ := wasmSection(1, []byte{1, 96, 0, 0})
	imp := []byte{1, 3, 'e', 'n', 'v', 4, 'e', 'v', 'i', 'l', 0, 0}
	module := append(append(append([]byte{}, header...), typ...), wasmSection(2, imp)...)
	module = append(module, wasmSection(7, []byte{1, 6, '_', 's', 't', 'a', 'r', 't', 0, 0})...)
	e := engine(t, DefaultLimits())
	if err := e.Load(context.Background(), demo.Descriptor(), module); err == nil || !strings.Contains(err.Error(), "import") {
		t.Fatal("unsupported host import accepted", err)
	}
	large := append(append([]byte{}, header...), wasmSection(5, []byte{1, 0, 0x80, 0x10})...)
	if err := e.Load(context.Background(), demo.Descriptor(), large); err == nil {
		t.Fatal("memory over limit accepted")
	}
}
