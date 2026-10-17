// Build: GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags="-s -w" -o ../provider.wasm .
package main

import (
	"encoding/json"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
	"strings"
)

// This deliberately fictional provider works offline. It cannot see Mosaune's
// libraries. IDs are opaque strings, so hosts cannot assume TMDB numeric IDs.
type Film = metadata.Candidate

var films = []Film{{ID: "atelier:etoiles", Title: "Voyage sous les étoiles", Year: 2024, Overview: "Une astronome prépare un voyage imaginaire. Fiche de démonstration, pas un vrai film."}, {ID: "atelier:jardin", Title: "Le Jardin des possibles", Year: 2025, Overview: "Des voisins transforment un jardin oublié. Fiche de démonstration, pas un vrai film."}}

func handle(request protocol.Request) (any, *protocol.Failure) {
	switch request.Capability {
	case metadata.SearchCapability:
		if request.Operation != "search" {
			break
		}
		var query metadata.Query
		if json.Unmarshal(request.Payload, &query) != nil || query.Validate() != nil || query.Kind != "films" {
			return nil, &protocol.Failure{Code: "invalid_query", Message: "Ce fournisseur de démonstration recherche uniquement des films."}
		}
		items := []Film{}
		for _, f := range films {
			if strings.Contains(strings.ToLower(f.Title), strings.ToLower(query.Title)) {
				items = append(items, f)
			}
		}
		return metadata.SearchResult{Items: items}, nil
	case metadata.DetailsCapability:
		if request.Operation != "details" {
			break
		}
		var selection metadata.Selection
		_ = json.Unmarshal(request.Payload, &selection)
		for _, f := range films {
			if f.ID == selection.ID {
				return f, nil
			}
		}
		return nil, &protocol.Failure{Code: "not_found", Message: "Aucune fiche de démonstration pour cet identifiant."}
	}
	return nil, &protocol.Failure{Code: "unsupported", Message: "Cette capacité ou opération n’est pas prise en charge."}
}
func main() {
	if sdk.Serve(os.Stdin, os.Stdout, handle) != nil {
		os.Exit(1)
	}
}
