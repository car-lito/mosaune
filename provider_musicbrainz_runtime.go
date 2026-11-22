package main

import "context"

func (p *musicBrainzProvider) StartMetadataProvider(ctx context.Context) error { return ctx.Err() }

func (p *musicBrainzProvider) CloseMetadataProvider() error {
	// Music's executable calls belong to the shared recognition runtime, which
	// the catalogue cancels before closing any provider. Release idle HTTP pools.
	p.client.CloseIdleConnections()
	return nil
}
