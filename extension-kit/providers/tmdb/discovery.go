package tmdb

import (
	_ "embed"
	"encoding/json"
	discovery "mosaune/extensionkit/discovery/v1"
)

//go:embed discovery.json
var discoveryJSON []byte

// DiscoverySurface returns a copy of the bundled adapter declaration.
func DiscoverySurface() discovery.Surface {
	var s discovery.Surface
	if json.Unmarshal(discoveryJSON, &s) != nil {
		return discovery.Surface{}
	}
	return s
}
