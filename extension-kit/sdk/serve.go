// Package sdk is optional convenience code for Go authors. Other languages can
// implement the same JSON-over-stdin/stdout contract without depending on Go.
package sdk

import (
	"encoding/json"
	"io"
	"mosaune/extensionkit/protocol"
)

type Handler func(protocol.Request) (any, *protocol.Failure)

// Serve handles one invocation. The host starts a fresh instance per request;
// durable state will be an explicit host capability, never guest global memory.
func Serve(in io.Reader, out io.Writer, handler Handler) error {
	return ServeSized(in, out, 64*1024, handler)
}

// ServeSized opts into a larger bounded envelope; host limits remain authoritative.
func ServeSized(in io.Reader, out io.Writer, limit int, handler Handler) error {
	if limit < 128 || limit > 1<<20 {
		return io.ErrShortBuffer
	}
	raw, err := io.ReadAll(io.LimitReader(in, int64(limit)+1))
	if err != nil {
		return err
	}
	var request protocol.Request
	if len(raw) > limit {
		return io.ErrShortBuffer
	}
	if err = protocol.DecodeOne(raw, &request); err != nil {
		return err
	}
	if err = request.Validate(); err != nil {
		return err
	}
	result, failure := handler(request)
	response := protocol.Response{Protocol: protocol.Version, ID: request.ID, Error: failure}
	if failure == nil {
		response.Result, err = json.Marshal(result)
		if err != nil {
			return err
		}
	}
	if err = response.Validate(request); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(response)
}
