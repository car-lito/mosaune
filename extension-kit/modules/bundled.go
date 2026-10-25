// Package modules contains only the official recognition executables and descriptors.
// It does not import their implementations into the host executable.
package modules

import (
	"embed"
	"encoding/json"
	"fmt"
	"mosaune/extensionkit/protocol"
)

//go:embed audiotags/tags.wasm audiotags/module.json films/recognizer.wasm series/recognizer.wasm music/recognizer.wasm films/module.json series/module.json music/module.json
var assets embed.FS

func Lookup(kind string) (protocol.Descriptor, []byte, error) {
	d, err := Descriptor(kind)
	if err != nil {
		return d, nil, err
	}
	file := "/recognizer.wasm"
	if kind == "audiotags" {
		file = "/tags.wasm"
	}
	code, err := assets.ReadFile(kind + file)
	return d, code, err
}

// Descriptor reads only the tiny manifest. Listing or toggling installed
// modules must not allocate a copy of each multi-megabyte factory executable.
func Descriptor(kind string) (protocol.Descriptor, error) {
	var d protocol.Descriptor
	switch kind {
	case "films", "series", "music", "audiotags":
	default:
		return d, fmt.Errorf("unknown recognition module")
	}
	raw, err := assets.ReadFile(kind + "/module.json")
	if err != nil {
		return d, err
	}
	if err = json.Unmarshal(raw, &d); err != nil {
		return d, err
	}
	if err = d.Validate(); err != nil {
		return d, err
	}
	return d, nil
}
