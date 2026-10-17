package sdk

import (
	"encoding/json"
	"errors"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/services"
)

// ReadService cannot choose a URL, method, header or credential. The host must
// grant the named service and validate its operations and parameters separately.
func ReadService(service, operation string, params any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	request := services.Request{Service: service, Operation: operation, Params: raw}
	if err = request.Validate(); err != nil {
		return nil, err
	}
	input, err := json.Marshal(request)
	if err != nil || len(input) > services.MaxRequest {
		return nil, errors.New("service request too large")
	}
	output := make([]byte, services.MaxResponse)
	n := readService(input, output)
	if n <= 0 || int(n) > len(output) {
		return nil, errors.New("service unavailable")
	}
	var reply services.Reply
	if err = protocol.DecodeOne(output[:n], &reply); err != nil {
		return nil, err
	}
	if reply.Error != "" {
		return nil, errors.New(reply.Error)
	}
	if !json.Valid(reply.Data) {
		return nil, errors.New("invalid service data")
	}
	return reply.Data, nil
}
