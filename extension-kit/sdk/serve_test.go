package sdk

import (
	"bytes"
	"errors"
	"io"
	"mosaune/extensionkit/protocol"
	"strings"
	"testing"
)

func TestServeOneMessage(t *testing.T) {
	input := `{"protocol":1,"id":"q1","capability":"atelier.demo.v1","operation":"run","payload":{}}`
	out := &bytes.Buffer{}
	if err := Serve(strings.NewReader(input), out, func(r protocol.Request) (any, *protocol.Failure) { return map[string]bool{"ok": true}, nil }); err != nil {
		t.Fatal(err)
	}
	var response protocol.Response
	if err := protocol.DecodeOne(out.Bytes(), &response); err != nil || response.ID != "q1" {
		t.Fatal(response, err)
	}
	called := false
	handler := func(protocol.Request) (any, *protocol.Failure) { called = true; return nil, nil }
	if err := Serve(strings.NewReader(input+input), out, handler); err == nil || called {
		t.Fatal("multiple messages accepted")
	}
	if err := Serve(strings.NewReader(strings.Repeat("a", 65537)), out, handler); !errors.Is(err, io.ErrShortBuffer) || called {
		t.Fatal(err)
	}
}
