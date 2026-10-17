// Package demo embeds the example only for the standalone lab and tests.
package demo

import (
	_ "embed"
	"mosaune/extensionkit/protocol"
)

//go:embed provider.wasm
var Code []byte

func Descriptor() protocol.Descriptor {
	return protocol.Descriptor{ID: "atelier.catalogue", Version: "0.1.0", Capabilities: []string{"metadata.search.v1", "metadata.details.v1"}}
}
