package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/protocol"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDeclaredServicesIntersectHostGrantsAndAreCopied(t *testing.T) {
	e := engine(t, DefaultLimits())
	code, err := os.ReadFile("testdata/probe.wasm")
	if err != nil {
		t.Fatal(err)
	}
	d := protocol.Descriptor{ID: "test.declared", Version: "1.0.0", Capabilities: []string{"test.probe.v1"}, Services: []protocol.ServiceRequirement{{Name: "test.api.v1", Operations: []string{"get"}}}}
	if err = e.Load(context.Background(), d, code); err != nil {
		t.Fatal(err)
	}
	if err = e.Enable(d.ID); err != nil {
		t.Fatal(err)
	}
	d.Services[0].Operations[0] = "delete" // Caller mutation cannot expand rights.
	calls := 0
	grants := ServiceGrants{"test.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"ok":true}`), nil
	}, "other.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
		t.Fatal("undeclared service reached")
		return nil, nil
	}}
	for _, payload := range []string{`{"service":"test.api.v1","operation":"delete","params":{}}`, `{"service":"other.api.v1","operation":"get","params":{}}`} {
		if _, err = e.CallWithServices(context.Background(), d.ID, request("test.probe.v1", "service", payload), grants); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("undeclared operation reached callback")
	}
	if _, err = e.CallWithServices(context.Background(), d.ID, request("test.probe.v1", "service", `{"service":"test.api.v1","operation":"get","params":{}}`), grants); err != nil || calls != 1 {
		t.Fatal(calls, err)
	}
	// No declarations means no host service, even if the host offers one.
	d.ID = "test.offline"
	d.Services = nil
	if err = e.Load(context.Background(), d, code); err != nil {
		t.Fatal(err)
	}
	_ = e.Enable(d.ID)
	if _, err = e.CallWithServices(context.Background(), d.ID, request("test.probe.v1", "service", `{"service":"test.api.v1","operation":"get","params":{}}`), grants); err != nil || calls != 1 {
		t.Fatal(calls, err)
	}
}

func TestServiceGrantsBudgetsAndRedaction(t *testing.T) {
	e := engine(t, DefaultLimits())
	probe(t, e)
	ctx := context.Background()
	req := request("test.probe.v1", "service", `{"service":"test.api.v1","operation":"get","params":{},"count":5}`)
	count := 0
	grants := ServiceGrants{"test.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
		count++
		return json.RawMessage(`{"public":true}`), nil
	}}
	denied, err := e.Call(ctx, "test.probe", req)
	if err != nil || !strings.Contains(string(denied.Response.Result), "service unavailable") {
		t.Fatal(denied, err)
	}
	allowed, err := e.CallWithServices(ctx, "test.probe", req, grants)
	if err != nil || count != 4 || strings.Count(string(allowed.Response.Result), `"public":true`) != 4 || !strings.Contains(string(allowed.Response.Result), "service unavailable") {
		t.Fatal(count, allowed, err)
	}
	denied, err = e.CallWithServices(ctx, "test.probe", req, ServiceGrants{"different.api.v1": grants["test.api.v1"]})
	if err != nil || count != 4 || !strings.Contains(string(denied.Response.Result), "service_denied") {
		t.Fatal(denied, err)
	}
	secret := "fixture-private-token"
	reply, err := e.CallWithServices(ctx, "test.probe", req, ServiceGrants{"test.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
		return nil, errors.New(secret)
	}})
	if err != nil || strings.Contains(string(reply.Response.Result), secret) || !strings.Contains(string(reply.Response.Result), "service_failed") {
		t.Fatal(reply, err)
	}
	for _, body := range []json.RawMessage{json.RawMessage(`not json`), json.RawMessage(`"` + strings.Repeat("x", (2<<20)+1) + `"`)} {
		reply, err = e.CallWithServices(ctx, "test.probe", req, ServiceGrants{"test.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) { return body, nil }})
		if err != nil || !strings.Contains(string(reply.Response.Result), "service_invalid_response") {
			t.Fatal(reply, err)
		}
	}
}
func TestDisableCancelsHostService(t *testing.T) {
	e := engine(t, DefaultLimits())
	probe(t, e)
	started := make(chan struct{})
	finished := make(chan error, 1)
	req := request("test.probe.v1", "service", `{"service":"test.api.v1","operation":"get","params":{}}`)
	go func() {
		_, err := e.CallWithServices(context.Background(), "test.probe", req, ServiceGrants{"test.api.v1": func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}})
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("service not started")
	}
	if err := e.Disable("test.probe"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("service not canceled")
	}
}

func TestHostSelectedServiceBudgetSurvivesDeclarationIntersection(t *testing.T) {
	e := engine(t, DefaultLimits())
	probe(t, e)
	count := 0
	grants := ServiceGrants{"test.api.v1": func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
		count++
		return json.RawMessage(`{"ok":true}`), nil
	}}
	req := request("test.probe.v1", "service", `{"service":"test.api.v1","operation":"get","params":{},"count":8}`)
	if _, err := e.CallWithServiceBudget(context.Background(), "test.probe", req, grants, 6); err != nil || count != 6 {
		t.Fatal(count, err)
	}
	for _, budget := range []int{0, 513} {
		if _, err := e.CallWithServiceBudget(context.Background(), "test.probe", req, grants, budget); err == nil {
			t.Fatal("unbounded budget")
		}
	}
}
