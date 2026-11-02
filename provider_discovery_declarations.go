package main

import (
	_ "embed"
	discovery "mosaune/extensionkit/discovery/v1"
)

// Declarations owned by the current provider adapters, never by UI renderers.
// Packaging arbitrary provider adapters remains a separate migration.
//

//go:embed provider_musicbrainz_discovery.json
var musicBrainzDiscoverySurface []byte

func (p *musicBrainzProvider) legacyDiscoverySurface() discovery.Surface {
	return readDiscoverySurface(musicBrainzDiscoverySurface)
}
