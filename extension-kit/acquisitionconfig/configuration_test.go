package acquisitionconfig

import "testing"

func TestConfigurationDestinationBinding(t *testing.T) {
	old := Configuration{ID: "radarr", URL: "http://nas/radarr", Key: "private-key", Revision: 7}
	input := Configuration{ID: "radarr", URL: "http://nas/radarr/api/v3/"}
	got, err := PrepareConfiguration(input, old)
	if err != nil || got.URL != old.URL || got.Key != old.Key || got.Revision != 7 {
		t.Fatal(got, err)
	}
	for _, url := range []string{"http://other", "file:///x", "http://user:pass@nas", "http://nas?key=private", "http://nas?", "http://nas#x"} {
		input.URL = url
		if _, err = PrepareConfiguration(input, old); err == nil {
			t.Fatal("unsafe or changed address accepted", url)
		}
	}
	input.URL = old.URL
	input.ID = "sonarr"
	if _, err = PrepareConfiguration(input, old); err == nil {
		t.Fatal("cross connector reuse")
	}
	input.ID = old.ID
	input.Key = "bad\r\nkey"
	if _, err = PrepareConfiguration(input, old); err == nil {
		t.Fatal("header injection")
	}
}
