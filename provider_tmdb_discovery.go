package main

import (
	"context"
	"database/sql"
	"errors"
	discovery "mosaune/extensionkit/discovery/v1"
	bundled "mosaune/extensionkit/providers/tmdb"
	"sync"
)

// Only the bounded declaration is retained, never the archive or Wasm program.
// Querying the persisted checksum makes update, rollback and removal visible
// without another mutable installation flag or database migration.
type tmdbDiscoveryCache struct {
	mu       sync.Mutex
	checksum string
	surface  discovery.Surface
}

func cloneDiscoverySurface(s discovery.Surface) discovery.Surface {
	s.Search = append([]discovery.Search(nil), s.Search...)
	s.Shelves = append([]discovery.Shelf(nil), s.Shelves...)
	return s
}
func (p *tmdbProvider) DiscoverySurfaceContext(ctx context.Context) (discovery.Surface, error) {
	empty := discovery.Surface{Version: 1}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	// Non-persistent adapters are used by embedded engine fixtures only.
	if p.db == nil {
		return bundled.DiscoverySurface(), nil
	}
	cache := &p.discoveryCache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	var checksum string
	err := p.db.QueryRowContext(ctx, "SELECT checksum FROM provider_packages WHERE id='tmdb'").Scan(&checksum)
	if err != nil {
		cache.checksum, cache.surface = "", empty
		if errors.Is(err, sql.ErrNoRows) {
			return empty, nil
		}
		return empty, err
	}
	if checksum != "" && checksum == cache.checksum {
		return cloneDiscoverySurface(cache.surface), nil
	}
	pack, err := p.installedPackage(ctx)
	if err != nil {
		cache.checksum = ""
		return empty, err
	}
	surface := empty
	if pack.Manifest.Discovery != nil {
		surface = *pack.Manifest.Discovery
	} else {
		// Compatibility for immutable packages released before declarations existed.
		// Absence of a package or an explicitly empty declaration never falls back.
		for _, cap := range pack.Manifest.Descriptor.Capabilities {
			if cap == "metadata.video-catalog.v1" {
				surface = bundled.DiscoverySurface()
				break
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	cache.checksum, cache.surface = pack.Checksum, cloneDiscoverySurface(surface)
	return cloneDiscoverySurface(surface), nil
}
