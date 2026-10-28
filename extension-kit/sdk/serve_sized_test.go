package sdk

import (
	"bytes"
	"io"
	"mosaune/extensionkit/protocol"
	"strings"
	"testing"
)

func TestSizedEnvelopesRemainBounded(t *testing.T) {
	raw := `{"protocol":1,"id":"test","capability":"test.large.v1","operation":"project","payload":{"text":"` + strings.Repeat("x", 70<<10) + `"}}`
	called := false
	handler := func(protocol.Request) (any, *protocol.Failure) { called = true; return struct{}{}, nil }
	if e := Serve(strings.NewReader(raw), &bytes.Buffer{}, handler); e != io.ErrShortBuffer || called {
		t.Fatal("default limit changed", e)
	}
	if e := ServeSized(strings.NewReader(raw), &bytes.Buffer{}, 1<<20, handler); e != nil || !called {
		t.Fatal(e)
	}
	called = false
	if e := ServeSized(strings.NewReader(raw), &bytes.Buffer{}, 2<<20, handler); e != io.ErrShortBuffer || called {
		t.Fatal("unbounded envelope", e)
	}
}
