package main

import "context"

func (p *tmdbProvider) Search(ctx context.Context, secret string, q MetadataQuery) ([]MetadataCandidate, error) {
	return providerResult(ctx, p, func(call context.Context) ([]MetadataCandidate, error) { return p.activeSearch(call, secret, q) })
}
func (p *tmdbProvider) Details(ctx context.Context, secret string, s MetadataSelection) (Metadata, error) {
	return providerResult(ctx, p, func(call context.Context) (Metadata, error) { return p.activeDetails(call, secret, s) })
}
func (p *tmdbProvider) Discover(ctx context.Context, secret string, q DiscoveryQuery) (DiscoveryPage, error) {
	return providerResult(ctx, p, func(call context.Context) (DiscoveryPage, error) { return p.activeDiscover(call, secret, q) })
}
func (p *tmdbProvider) Artwork(ctx context.Context, secret string, ref string) ([]byte, error) {
	return providerResult(ctx, p, func(call context.Context) ([]byte, error) { return p.activeArtwork(call, secret, ref) })
}
func (p *tmdbProvider) Shelf(ctx context.Context, secret, shelf, kind string) (DiscoveryPage, error) {
	return providerResult(ctx, p, func(call context.Context) (DiscoveryPage, error) { return p.activeShelf(call, secret, shelf, kind) })
}
func (p *tmdbProvider) Profile(ctx context.Context, secret, id string, page int) (PersonProfile, error) {
	return providerResult(ctx, p, func(call context.Context) (PersonProfile, error) { return p.activeProfile(call, secret, id, page) })
}
func (p *tmdbProvider) Posters(ctx context.Context, secret string, s MetadataSelection, language string) ([]PosterOption, error) {
	return providerResult(ctx, p, func(call context.Context) ([]PosterOption, error) { return p.activePosters(call, secret, s, language) })
}
