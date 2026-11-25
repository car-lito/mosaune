// Bundled data only; executable behaviour belongs to extension-kit or server.
package resources

import "embed"

//go:embed examples/ambre/extension.json
var ExampleTheme []byte

//go:embed examples/ambre-v2/extension.json
var ExampleThemeV2 []byte

//go:embed all:web/dist
var Frontend embed.FS

//go:embed extensions/media-*/extension.json
var SuppliedMediaBundles embed.FS

//go:embed media-sets.json
var MediaSetsJSON []byte

//go:embed provider_musicbrainz_discovery.json
var MusicBrainzDiscoverySurface []byte

//go:embed provider-package-key.pub
var ProviderPublicKey string

//go:embed packages/tmdb-0.4.0.mosaune.zip
var ProviderExample []byte

//go:embed packages/tmdb-0.4.1.mosaune.zip
var ProviderUpdateExample []byte

//go:embed packages/tmdb-0.5.0.mosaune.zip
var ProviderCatalogueExample []byte

//go:embed packages/tmdb-0.6.0.mosaune.zip
var ProviderDiscoveryExample []byte

//go:embed packages/tmdb-0.7.0.mosaune.zip
var ProviderTransportExample []byte

//go:embed packages/tmdb-0.8.0.mosaune.zip
var ProviderIdentifiersExample []byte

//go:embed theme/default.json
var DefaultThemeJSON []byte

//go:embed theme/contract.json
var ThemeContractJSON []byte

//go:embed ui_navigation.json
var UiNavigationJSON []byte

//go:embed examples/reading-room/extension.json
var ExampleUIPages []byte

//go:embed extensions/ui-*/extension.json
var SuppliedScreens embed.FS

//go:embed examples/cinema-compact/extension.json
var CompactScreen []byte
