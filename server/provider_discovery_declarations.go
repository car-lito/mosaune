package main

import (
	discovery "mosaune/extensionkit/discovery/v1"
	"mosaune/resources"
)

// Declarations owned by the current provider adapters, never by UI renderers.
// Packaging arbitrary provider adapters remains a separate migration.
//

var musicBrainzDiscoverySurface = resources.MusicBrainzDiscoverySurface

func (p *musicBrainzProvider) legacyDiscoverySurface() discovery.Surface {
	return readDiscoverySurface(musicBrainzDiscoverySurface)
}
