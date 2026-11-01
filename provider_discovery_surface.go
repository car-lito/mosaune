package main

import (
	"encoding/json"
	"errors"
	discovery "mosaune/extensionkit/discovery/v1"
)

// Optional contract. Absence preserves legacy text search only; an explicit
// empty or invalid declaration must never regain features through a fallback.
type DiscoverySurfaceProvider interface{ DiscoverySurface() discovery.Surface }

func providerDiscoverySurface(p MetadataProvider) (*discovery.Surface, error) {
	v, ok := p.(DiscoverySurfaceProvider)
	if !ok {
		return nil, nil
	}
	surface := v.DiscoverySurface()
	if err := surface.Validate(p.Info().Kinds); err != nil {
		return nil, err
	}
	if len(surface.Search) > 0 {
		if _, ok := p.(DiscoveryProvider); !ok {
			return nil, errors.New("missing discovery implementation")
		}
	}
	if len(surface.Shelves) > 0 {
		if _, ok := p.(ShelfProvider); !ok {
			return nil, errors.New("missing shelf implementation")
		}
	}
	return &surface, nil
}
func providerSearchAllowed(p MetadataProvider, kind string, similar bool) bool {
	s, e := providerDiscoverySurface(p)
	if e != nil {
		return false
	}
	if s == nil {
		return true
	}
	v, ok := s.ForSearch(kind)
	return ok && (!similar || v.Similar)
}
func readDiscoverySurface(raw []byte) discovery.Surface {
	var s discovery.Surface
	if json.Unmarshal(raw, &s) != nil {
		return discovery.Surface{}
	}
	return s
}
