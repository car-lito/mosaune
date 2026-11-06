package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed transport.json
var transportJSON []byte

func transport(connector string) (a.Transport, error) {
	var profiles map[string]a.Transport
	if err := json.Unmarshal(transportJSON, &profiles); err != nil {
		return a.Transport{}, err
	}
	p, ok := profiles[connector]
	if !ok {
		return p, errors.New("unknown connector")
	}
	return p, p.Validate(connector)
}
