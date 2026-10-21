// Package modules contains only the official recognition executables and descriptors.
// It does not import their implementations into the host executable.
package modules

import (
	"embed"
	"encoding/json"
	"fmt"
	"mosaune/extensionkit/protocol"
)

//go:embed films/recognizer.wasm series/recognizer.wasm music/recognizer.wasm films/module.json series/module.json music/module.json
var assets embed.FS

func Lookup(kind string) (protocol.Descriptor, []byte, error) {
	var d protocol.Descriptor
	switch kind {
	case "films", "series", "music":
	default:
		return d, nil, fmt.Errorf("unknown recognition module")
	}
	raw, err := assets.ReadFile(kind + "/module.json")
	if err != nil {
		return d, nil, err
	}
	if err = json.Unmarshal(raw, &d); err != nil {
		return d, nil, err
	}
	if err = d.Validate(); err != nil {
		return d, nil, err
	}
	code, err := assets.ReadFile(kind + "/recognizer.wasm")
	return d, code, err
}
