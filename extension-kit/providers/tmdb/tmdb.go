// Package tmdb describes the signed TMDB provider package. The installer
// does not accept arbitrary executable packages yet.
package tmdb

import (
	_ "embed"
	"encoding/json"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	video "mosaune/extensionkit/capabilities/video/v1"
	catalog "mosaune/extensionkit/capabilities/videocatalog/v1"
	"mosaune/extensionkit/configuration"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
)

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
	return protocol.Descriptor{ID: "mosaune.tmdb-search", Version: "0.5.0", Capabilities: []string{metadata.SearchCapability, video.DetailsCapability, catalog.Capability, packageformat.HealthCapability}, Services: []protocol.ServiceRequirement{{Name: "tmdb.api.v1", Operations: []string{"get"}}}}
}
