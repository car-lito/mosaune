// Package musicbrainz contains the transport declarations shipped by the music
// module. The server also uses this frozen profile for older installed modules.
package musicbrainz

import (
	_ "embed"
	"encoding/json"
	"mosaune/extensionkit/transport"
)

//go:embed transport.json
var profileJSON []byte

func Transport() transport.Profile {
	var profile transport.Profile
	if err := json.Unmarshal(profileJSON, &profile); err != nil {
		panic(err)
	}
	return profile
}
