package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

//go:embed settings.json
var settingsJSON []byte

func settings(connector string) (a.Settings, error) {
	var forms map[string]a.Settings
	if err := json.Unmarshal(settingsJSON, &forms); err != nil {
		return a.Settings{}, err
	}
	form, ok := forms[connector]
	if !ok {
		return form, errors.New("unknown connector")
	}
	return form, form.Validate()
}
