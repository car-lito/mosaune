package main

// Official composition is separate from catalogue startup. This is an explicit
// list of trusted host adapters, not permission to load arbitrary Go code.
func suppliedMetadataProviders() []metadataProviderFactory {
	return []metadataProviderFactory{
		{ID: "tmdb", Create: func(c *catalog) (metadataProviderRuntime, error) {
			p := newTMDBProvider(c.db)
			p.extension.blocked = func() bool { return c.extensionSafeMode }
			return p, nil
		}},
		{ID: "musicbrainz", Create: func(c *catalog) (metadataProviderRuntime, error) {
			p := newMusicBrainzProvider(c.db)
			p.owner = c
			return p, nil
		}},
	}
}
