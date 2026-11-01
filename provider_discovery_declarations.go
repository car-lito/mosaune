package main

import (
	_ "embed"
	discovery "mosaune/extensionkit/discovery/v1"
	bundled "mosaune/extensionkit/providers/tmdb"
)

// Declarations owned by the current provider adapters, never by UI renderers.
// Packaging arbitrary provider adapters remains a separate migration.
//

//go:embed provider_musicbrainz_discovery.json
var musicBrainzDiscoverySurface []byte

func (p *tmdbProvider) DiscoverySurface() discovery.Surface {
	return bundled.DiscoverySurface()
}
func (p *musicBrainzProvider) DiscoverySurface() discovery.Surface {
	return readDiscoverySurface(musicBrainzDiscoverySurface)
}
