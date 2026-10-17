// Package services defines requests to named host services, not network access.
// The host grants individual services for each invocation and keeps credentials.
package services

import (
	"encoding/json"
	"errors"
	"strings"
)

const Module = "mosaune_services_v1"
const MaxRequest = 8 << 10
const MaxResponse = (2 << 20) + 1024

type Request struct {
	Service   string          `json:"service"`
	Operation string          `json:"operation"`
	Params    json.RawMessage `json:"params"`
}
type Reply struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func (r Request) Validate() error {
	if len(r.Service) < 1 || len(r.Service) > 96 || len(r.Operation) < 1 || len(r.Operation) > 48 || len(r.Params) > MaxRequest || !json.Valid(r.Params) || !strings.HasPrefix(strings.TrimSpace(string(r.Params)), "{") {
		return errors.New("invalid service request")
	}
	return nil
}
