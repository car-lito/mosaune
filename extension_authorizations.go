package main

import (
	"errors"
	"mosaune/extensionkit/authorization"
	"net/http"
)

func (c *catalog) extensionAuthorizationRoutes(mux *secureMux) {
	// Only host adapters with a complete revocation/commit lifecycle may expose
	// mutable grants. A registry row alone must not enable an unenforced switch.
	provider := func(w http.ResponseWriter, r *http.Request) (providerServiceAccess, bool) {
		access, ok := c.authorizationFor(r.PathValue("id"))
		if !ok {
			problem(w, 404, "Accès de cette extension non disponibles.")
		}
		return access, ok
	}
	mux.HandleFunc("GET /api/v1/extensions/{id}/authorizations", func(w http.ResponseWriter, r *http.Request) {
		access, ok := provider(w, r)
		if !ok {
			return
		}
		state, err := access.read(r.Context())
		if err != nil {
			problem(w, 500, "Autorisations indisponibles.")
			return
		}
		sendJSON(w, 200, state)
	})
	mux.HandleFunc("PUT /api/v1/extensions/{id}/authorizations", func(w http.ResponseWriter, r *http.Request) {
		access, ok := provider(w, r)
		if !ok {
			return
		}
		var patch authorization.Patch
		if !readJSON(w, r, &patch) {
			return
		}
		if err := access.apply(r.Context(), patch); err != nil {
			switch {
			case errors.Is(err, authorization.ErrConflict):
				problem(w, 409, err.Error())
			case errors.Is(err, authorization.ErrInvalid):
				problem(w, 400, err.Error())
			default:
				problem(w, 500, "Autorisations non enregistrées.")
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
