package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed wire.json
var wireJSON []byte

func wireFormat(connector string) (a.WireFormat, error) {
	var formats map[string]a.WireFormat
	if err := json.Unmarshal(wireJSON, &formats); err != nil {
		return a.WireFormat{}, err
	}
	format, ok := formats[connector]
	if !ok {
		return format, errors.New("unknown connector")
	}
	return format, format.Validate(connector)
}
