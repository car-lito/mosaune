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
	raw, err := io.ReadAll(io.LimitReader(in, 64*1024+1))
	if err != nil {
		return err
	}
	var request protocol.Request
	if len(raw) > 64*1024 {
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
