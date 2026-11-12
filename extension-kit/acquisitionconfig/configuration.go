package acquisitionconfig

import (
	"context"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/configuration"
	"net/url"
	"strconv"
	"strings"
)

// Configuration and ConfigurationStore are host-only. A replacement shares the
// capability's connector settings, never an owner chosen by untrusted code.
type Configuration struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Key        string `json:"key,omitempty"`
	Configured bool   `json:"configured"`
	Enabled    bool   `json:"enabled"`
	Profile    int    `json:"profile"`
	Root       string `json:"root"`
	Library    string `json:"library"`
	Revision   int64  `json:"-"`
}
type ConfigurationStore struct{ DB configuration.Connection }

func ConfigurationOwner(id string) (string, error) {
	if id != "radarr" && id != "sonarr" {
		return "", errors.New("Connecteur inconnu.")
	}
	return "mosaune.acquisition." + id, nil
}
func connectorSchema() configuration.Schema {
	return configuration.Schema{Version: 1, Title: "Connexion de téléchargement", Fields: []configuration.Field{
		{ID: "url", Label: "Adresse", Type: "text", MaxLength: 2048},
		{ID: "key", Label: "Clé API", Type: "secret", MaxLength: 200},
		{ID: "enabled", Label: "Actif", Type: "boolean"},
		// Storage has text/boolean fields. The adapter preserves the public numeric API.
		{ID: "profile", Label: "Profil", Type: "text", MaxLength: 20},
		{ID: "root", Label: "Destination", Type: "text", MaxLength: 4096},
		{ID: "library", Label: "Bibliothèque", Type: "text", MaxLength: 256},
	}}
}
func (s ConfigurationStore) Register(ctx context.Context, id string) error {
	owner, err := ConfigurationOwner(id)
	if err != nil {
		return err
	}
	return (configuration.SQLStore{DB: s.DB}).Register(ctx, owner, connectorSchema())
}
func (s ConfigurationStore) Bound(id string) (configuration.Bound, error) {
	owner, err := ConfigurationOwner(id)
	if err != nil {
		return configuration.Bound{}, err
	}
	return (configuration.SQLStore{DB: s.DB}).For(owner)
}
func (s ConfigurationStore) Load(ctx context.Context, id string) (Configuration, error) {
	out := Configuration{ID: id}
	b, err := s.Bound(id)
	if err != nil {
		return out, err
	}
	state, secret, err := b.ReadWithSecret(ctx, "key")
	if err != nil {
		return out, err
	}
	out.Revision = state.Revision
	var profile string
	for name, target := range map[string]any{"url": &out.URL, "enabled": &out.Enabled, "root": &out.Root, "library": &out.Library, "profile": &profile} {
		if raw, ok := state.Values[name]; ok {
			if string(raw) == "null" || json.Unmarshal(raw, target) != nil {
				return out, configuration.ErrStored
			}
		}
	}
	if profile != "" {
		out.Profile, err = strconv.Atoi(profile)
		if err != nil || out.Profile < 0 {
			return out, configuration.ErrStored
		}
	}
	out.Key = secret
	out.Configured = out.Key != ""
	return out, err
}
func configurationPatch(v Configuration) (configuration.Patch, error) {
	if _, err := ConfigurationOwner(v.ID); err != nil {
		return configuration.Patch{}, err
	}
	if v.Profile < 0 {
		return configuration.Patch{}, errors.New("Profil invalide.")
	}
	patch := configuration.Patch{Revision: &v.Revision, Values: map[string]json.RawMessage{}}
	for key, value := range map[string]any{"url": v.URL, "enabled": v.Enabled, "profile": strconv.Itoa(v.Profile), "root": v.Root, "library": v.Library} {
		patch.Values[key], _ = json.Marshal(value)
	}
	if v.Key == "" {
		patch.ClearSecrets = []string{"key"}
	} else {
		patch.Values["key"], _ = json.Marshal(v.Key)
	}
	// Reject destructive normalization, including during migration of old paths.
	normalized, err := connectorSchema().NormalizePatch(patch)
	if err != nil {
		return patch, err
	}
	for k, raw := range patch.Values {
		if string(raw) != string(normalized.Values[k]) {
			return patch, configuration.ErrIncompatible
		}
	}
	return patch, nil
}
func (s ConfigurationStore) Save(ctx context.Context, v Configuration) error {
	patch, err := configurationPatch(v)
	if err != nil {
		return err
	}
	b, err := s.Bound(v.ID)
	if err != nil {
		return err
	}
	return b.Apply(ctx, patch)
}
func (s ConfigurationStore) Pause(ctx context.Context, id string) error {
	b, err := s.Bound(id)
	if err != nil {
		return err
	}
	state, err := b.Read(ctx)
	if err != nil {
		return err
	}
	return b.Apply(ctx, configuration.Patch{Revision: &state.Revision, Values: map[string]json.RawMessage{"enabled": json.RawMessage("false")}})
}

// NormalizeAddress preserves the legacy base-URL convention for existing arr
// connectors. It does not grant any network access or expose credentials.
func NormalizeAddress(raw string) (string, error) {
	u, e := url.Parse(strings.TrimSpace(raw))
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", errors.New("Indiquez l’URL HTTP ou HTTPS du service, sans clé, identifiants ni paramètres.")
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/api/v3")
	u.RawPath = ""
	return strings.TrimRight(u.String(), "/"), nil
}
func PrepareConfiguration(input, old Configuration) (Configuration, error) {
	if _, err := ConfigurationOwner(input.ID); err != nil {
		return input, err
	}
	if old.ID != input.ID {
		return input, errors.New("Configuration d’un autre connecteur.")
	}
	normalized, err := NormalizeAddress(input.URL)
	if err != nil {
		return input, err
	}
	input.URL = normalized
	input.Key = strings.TrimSpace(input.Key)
	if input.Key == "" && old.URL == input.URL {
		input.Key = old.Key
	}
	if input.Key == "" || len(input.Key) > 200 || strings.ContainsAny(input.Key, "\x00\r\n") {
		return input, errors.New("Saisissez la clé API du service. Une nouvelle adresse nécessite sa clé.")
	}
	input.Configured = true
	input.Revision = old.Revision
	return input, nil
}
