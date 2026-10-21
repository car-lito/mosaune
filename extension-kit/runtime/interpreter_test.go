package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInterpreterKeepsIsolationAndCancellation(t *testing.T) {
	limits := DefaultLimits()
	limits.Interpreter = true
	limits.Timeout = time.Second
	e := engine(t, limits)
	probe(t, e)
	sentinel := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOSAUNE_TEST_SECRET", "not-visible")
	raw, _ := json.Marshal(map[string]string{"path": sentinel})
	out, err := e.Call(context.Background(), "test.probe", request("test.probe.v1", "probe", string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]bool
	if json.Unmarshal(out.Response.Result, &result) != nil || !result["readDenied"] || !result["writeDenied"] || !result["environmentEmpty"] {
		t.Fatal(string(out.Response.Result))
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "unchanged" {
		t.Fatal("host file changed", err)
	}
	out, err = e.Call(context.Background(), "test.probe", request("test.probe.v1", "service", `{"Service":"test.api.v1","Operation":"get"}`))
	var service struct {
		Results []struct {
			Error string `json:"error"`
		} `json:"results"`
	}
	if err != nil || json.Unmarshal(out.Response.Result, &service) != nil || len(service.Results) != 1 || service.Results[0].Error == "" {
		t.Fatal("ungranted service", out, err)
	}
	if _, err = e.Call(context.Background(), "test.probe", request("test.probe.v1", "loop", `{}`)); err == nil {
		t.Fatal("infinite loop was not stopped")
	}
	if _, err = e.Call(context.Background(), "test.probe", request("test.probe.v1", "healthy", `{}`)); err != nil {
		t.Fatal("failure poisoned next invocation", err)
	}
	if err = e.Disable("test.probe"); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Call(context.Background(), "test.probe", request("test.probe.v1", "healthy", `{}`)); !errors.Is(err, ErrUnavailable) {
		t.Fatal("disabled module ran", err)
	}
}
