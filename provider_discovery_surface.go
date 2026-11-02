package main

import (
	"context"
	"encoding/json"
	"errors"
	discovery "mosaune/extensionkit/discovery/v1"
)

// Optional contract. Absence preserves legacy text search only; an explicit
// empty or invalid declaration must never regain features through a fallback.
type DiscoverySurfaceProvider interface{ DiscoverySurface() discovery.Surface }

type ContextDiscoverySurfaceProvider interface {
	DiscoverySurfaceContext(context.Context) (discovery.Surface, error)
}

func providerDiscoverySurface(ctx context.Context, p MetadataProvider) (*discovery.Surface, error) {
	var surface discovery.Surface
	if v, ok := p.(ContextDiscoverySurfaceProvider); ok {
		var err error
		surface, err = v.DiscoverySurfaceContext(ctx)
		if err != nil {
			return nil, err
		}
	} else if v, ok := p.(DiscoverySurfaceProvider); ok {
		surface = v.DiscoverySurface()
	} else {
		return nil, nil
	}
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
func providerSearchAllowed(ctx context.Context, p MetadataProvider, kind string, similar bool) bool {
	s, e := providerDiscoverySurface(ctx, p)
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
