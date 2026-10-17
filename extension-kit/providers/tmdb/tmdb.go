// Package tmdb embeds the first bundled executable provider. The installer
// does not accept arbitrary executable packages yet.
package tmdb

import (
	_ "embed"
	"encoding/json"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	video "mosaune/extensionkit/capabilities/video/v1"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
)

//go:embed provider.wasm
var Code []byte

//go:embed settings.json
var settingsJSON []byte

func Settings() configuration.Schema {
	var schema configuration.Schema
	if err := json.Unmarshal(settingsJSON, &schema); err != nil {
		panic(err)
	}
	if err := schema.Validate(); err != nil {
		panic(err)
	}
	return schema
}

func Descriptor() protocol.Descriptor {
	return protocol.Descriptor{ID: "mosaune.tmdb-search", Version: "0.4.0", Capabilities: []string{metadata.SearchCapability, video.DetailsCapability, packageformat.HealthCapability}, Services: []protocol.ServiceRequirement{{Name: "tmdb.api.v1", Operations: []string{"get"}}}}
}
