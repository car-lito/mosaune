package main

import "testing"

func TestSettingsGuards(t *testing.T) {
	for _, id := range []string{"radarr", "sonarr"} {
		form, err := settings(id)
		if err != nil {
			t.Fatal(err)
		}
		if err = form.ValidateConnection("http://nas", "secret"); err != nil {
			t.Fatal(err)
		}
		form.Fields[0].MaxLength = 4
		if form.ValidateConnection("http://nas", "secret") == nil {
			t.Fatal("length not enforced")
		}
		form.Fields[0].Binding = "key"
		if form.Validate() == nil {
			t.Fatal("duplicate binding")
		}
	}
	form, _ := settings("radarr")
	form.Fields[0].Binding = "script"
	if form.Validate() == nil {
		t.Fatal("unknown control")
	}
	form, _ = settings("radarr")
	form.Fields[1].MaxLength = 4096
	if form.Validate() == nil {
		t.Fatal("secret limit expanded")
	}
	form, _ = settings("radarr")
	form.Fields = form.Fields[:4]
	if form.Validate() == nil {
		t.Fatal("missing field")
	}
	if _, err := settings("arbitrary"); err == nil {
		t.Fatal("unknown connector")
	}
}
