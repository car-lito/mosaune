// Package protocol is the experimental, language-neutral command contract.
// Capabilities evolve separately; no caller has to implement every media type.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

const Version = 1

type Descriptor struct {
	ID           string               `json:"id"`
	Version      string               `json:"version"`
	Capabilities []string             `json:"capabilities"`
	Services     []ServiceRequirement `json:"services,omitempty"`
}

// A request for access, never a grant. The host must independently authorize
// the service and its exact parameters on every invocation.
type ServiceRequirement struct {
	Name       string   `json:"name"`
	Operations []string `json:"operations"`
}
type Request struct {
	Protocol   int             `json:"protocol"`
	ID         string          `json:"id"`
	Capability string          `json:"capability"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
}
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	Protocol int             `json:"protocol"`
	ID       string          `json:"id"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    *Failure        `json:"error,omitempty"`
}

var name = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,47}$`)
var capability = regexp.MustCompile(`^[a-z][a-z0-9_-]*(\.[a-z][a-z0-9_-]*)+\.v[1-9][0-9]*$`)
var identifier = regexp.MustCompile(`^[a-z][a-z0-9_-]*(\.[a-z][a-z0-9_-]*)+$`)
var version = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})\.(0|[1-9][0-9]{0,4})$`)

func (d Descriptor) Validate() error {
	if len(d.ID) > 80 || !identifier.MatchString(d.ID) || !version.MatchString(d.Version) || len(d.Capabilities) == 0 || len(d.Capabilities) > 32 {
		return errors.New("invalid extension descriptor")
	}
	seen := map[string]bool{}
	for _, c := range d.Capabilities {
		if len(c) > 96 || !capability.MatchString(c) || seen[c] {
			return errors.New("invalid or duplicated capability")
		}
		seen[c] = true
	}
	if len(d.Services) > 16 {
		return errors.New("too many services")
	}
	seenServices := map[string]bool{}
	for _, s := range d.Services {
		if len(s.Name) > 96 || !capability.MatchString(s.Name) || seenServices[s.Name] || len(s.Operations) == 0 || len(s.Operations) > 16 {
			return errors.New("invalid service declaration")
		}
		seenServices[s.Name] = true
		ops := map[string]bool{}
		for _, op := range s.Operations {
			if !name.MatchString(op) || ops[op] {
				return errors.New("invalid service operation")
			}
			ops[op] = true
		}
	}
	return nil
}
func (r Request) Validate() error {
	if r.Protocol != Version || len(r.ID) == 0 || len(r.ID) > 80 || len(r.Capability) > 96 || !capability.MatchString(r.Capability) || !name.MatchString(r.Operation) || !Object(r.Payload) {
		return errors.New("invalid extension request")
	}
	return nil
}
func Object(raw []byte) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) > 1 && raw[0] == '{' && json.Valid(raw)
}

// DecodeOne rejects unknown envelope fields and trailing JSON. Payload schemas
// belong to their capability, not to the transport.
func DecodeOne(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected exactly one JSON message")
	}
	return nil
}
func (r Response) Validate(request Request) error {
	if r.Protocol != Version || r.ID != request.ID {
		return errors.New("response does not match request")
	}
	if r.Error != nil {
		if len(r.Result) != 0 || !name.MatchString(r.Error.Code) || len(r.Error.Message) == 0 || len(r.Error.Message) > 500 {
			return errors.New("invalid failure response")
		}
		return nil
	}
	if !Object(r.Result) {
		return errors.New("result must be an object")
	}
	return nil
}
