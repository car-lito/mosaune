package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

// Form data is transient API output. It is not persisted alongside credentials.
type acquisitionSettingsView struct {
	arrConfig
	Form         *a.Settings `json:"form,omitempty"`
	FormRevision string      `json:"formRevision,omitempty"`
	FormError    string      `json:"formError,omitempty"`
}

func readAcquisitionSettings(ctx context.Context, r *cataloguer, id string) (a.Settings, error) {
	var form a.Settings
	if err := r.call(ctx, "describe-settings", a.TransportQuery{Connector: id}, &form); err != nil {
		return form, errors.New("Mettez à jour le composant de demandes pour configurer ce connecteur.")
	}
	if err := form.Validate(); err != nil {
		return form, errors.New("Le formulaire du connecteur est incompatible.")
	}
	if form.Instructions == nil {
		form.Instructions = []string{}
	}
	return form, nil
}
func (client arrClient) settingsForm(ctx context.Context) (a.Settings, string, error) {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || s.owner != client.config.owner {
		return a.Settings{}, "", errors.New("Session de demandes absente.")
	}
	r, err := loadFactProjector(ctx, s.descriptor, s.code, a.Capability)
	if err != nil {
		return a.Settings{}, "", err
	}
	defer r.engine.Close()
	form, err := readAcquisitionSettings(ctx, r, client.config.ID)
	if err != nil {
		return form, "", err
	}
	return form, acquisitionFormRevision(ctx, client.config.ID, form, client.config.revision), nil
}
func (c *catalog) acquisitionSettings(ctx context.Context, cfg arrConfig) acquisitionSettingsView {
	// Bind displayed values and their stamp to the same stored snapshot.
	current, err := c.arrConfig(ctx, cfg.ID)
	if err != nil {
		cfg.Key = ""
		return acquisitionSettingsView{arrConfig: cfg, FormError: "Réglages indisponibles."}
	}
	cfg = current
	cfg.Key = ""
	view := acquisitionSettingsView{arrConfig: cfg}
	client := arrClient{cfg}
	ctx, done, err := client.openSession(ctx)
	defer done()
	if err == nil {
		var form a.Settings
		form, view.FormRevision, err = client.settingsForm(ctx)
		if err == nil {
			view.Form = &form
		}
	}
	if err != nil {
		view.FormError = err.Error()
	}
	return view
}

func acquisitionFormRevision(ctx context.Context, id string, form a.Settings, revision int64) string {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok {
		return ""
	}
	b, _ := json.Marshal([]any{s.checksum, id, form, revision})
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
