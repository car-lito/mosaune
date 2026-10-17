package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func themeV2Fixture(t *testing.T) extensionManifest {
	t.Helper()
	var m extensionManifest
	if e := json.Unmarshal(exampleThemeV2, &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func TestThemeDefaultsAndExample(t *testing.T) {
	if e := validateThemeV2(defaultTheme); e != nil {
		t.Fatal("default", e)
	}
	if len(themeContract.Tokens) < 900 {
		t.Fatal("incomplete component migration")
	}
	if len(defaultTheme.Tokens) != len(themeContract.Tokens) {
		t.Fatal("missing defaults")
	}
	built, e := os.ReadFile("web/src/theme-defaults.css")
	if e != nil {
		t.Fatal(e)
	}
	if string(built) != renderTheme(defaultTheme, "") {
		t.Fatal("frontend defaults differ from the server theme renderer; run scripts/build-theme.mjs")
	}
	m := themeV2Fixture(t)
	if e = validateExtension(m); e != nil {
		t.Fatal("example", e)
	}
	p, e := parseExtensionPackage(extensionZIP(t, exampleThemeV2, "extension.json"))
	if e != nil {
		t.Fatal(e)
	}
	css := extensionCSS(p, "teststamp")
	for _, fragment := range []string{"--m-sidebar-background:#352619;", "--m-button-primary-background:#884417;", "--m-tile-status-local-background:#563310f2;", "--m-button-primary-border-radius:10px;", "--m-app-font-family:\"Segoe UI\",\"Arial\",sans-serif;"} {
		if !strings.Contains(css, fragment) {
			t.Fatal("missing token", fragment)
		}
	}
	if strings.Contains(css, "data-ui-section") {
		t.Fatal("example unexpectedly changes theme between sections")
	}
	if strings.Contains(css, ".sidebar{") || strings.Contains(css, ".primary{") {
		t.Fatal("theme still overrides component CSS instead of supplying variables")
	}
}
func TestThemeSectionInheritanceAndReferences(t *testing.T) {
	m := themeV2Fixture(t)
	m.Theme.Palette["music-menu"] = themeModes{"#302343", "#171020"}
	m.Theme.Sections = map[string]map[string]string{"music": {"sidebar.background": "$palette.music-menu", "button.primary.border.radius": "$radii.round"}, "settings": {"sidebar.font.family": "$fonts.subtitle"}}
	if e := validateExtension(m); e != nil {
		t.Fatal(e)
	}
	css := extensionCSS(extensionPackage{Manifest: m}, "teststamp")
	marker := `.app-shell[data-extension-theme="teststamp"][data-ui-section="music"]{`
	if !strings.Contains(css, marker+"--m-button-primary-border-radius:50%;--m-sidebar-background:#302343;") {
		t.Fatal("section override missing")
	}
	if !strings.Contains(css, "--m-sidebar-background:#352619;") {
		t.Fatal("lost shared default")
	}
	if strings.Contains(css, `data-ui-section="films"`) {
		t.Fatal("unrequested section override")
	}
	m.Theme.Fonts["reading"] = []string{"Verdana", "sans-serif"}
	m.Theme.Tokens["app.font.family"] = "$fonts.reading"
	m.Theme.Radii["pill"] = "28px"
	m.Theme.Tokens["button.primary.border.radius"] = "$radii.pill"
	m.Theme.Tokens["button.shared.font.size"] = "18px"
	if e := validateExtension(m); e != nil {
		t.Fatal(e)
	}
	css = extensionCSS(extensionPackage{Manifest: m}, "fontstamp")
	if !strings.Contains(css, `--m-app-font-family:"Verdana",sans-serif;`) || !strings.Contains(css, "--m-button-primary-border-radius:28px;") || !strings.Contains(css, "--m-button-shared-font-size:18px;") {
		t.Fatal("primitive references ignored")
	}
}
func TestThemeV2RejectsUnsupportedOrUnreadableValues(t *testing.T) {
	cases := []struct {
		name   string
		change func(*extensionManifest)
	}{
		{"unknown token", func(m *extensionManifest) { m.Theme.Tokens["account.hide"] = "none" }},
		{"css injection", func(m *extensionManifest) { m.Theme.Tokens["sidebar.background"] = "#123456;display:none" }},
		{"unknown section", func(m *extensionManifest) {
			m.Theme.Sections = map[string]map[string]string{"login": {"sidebar.background": "$palette.ink"}}
		}},
		{"missing palette", func(m *extensionManifest) { m.Theme.Tokens["sidebar.background"] = "$palette.absent" }},
		{"wrong primitive type", func(m *extensionManifest) { m.Theme.Tokens["sidebar.background"] = "$fonts.body" }},
		{"font URL", func(m *extensionManifest) { m.Theme.Fonts["body"] = []string{"url(https://example.com/font.woff)"} }},
		{"font injection", func(m *extensionManifest) { m.Theme.Fonts["body"] = []string{"Arial\";display:none"} }},
		{"invisible text", func(m *extensionManifest) { m.Theme.Tokens["button.shared.font.size"] = "0px" }},
		{"excessive radius", func(m *extensionManifest) { m.Theme.Radii["poster"] = "9999px" }},
		{"unreadable section", func(m *extensionManifest) {
			m.Theme.Sections = map[string]map[string]string{"settings": {"sidebar.color": "$palette.sidebar"}}
		}},
		{"mixed v1", func(m *extensionManifest) { m.Theme.Light = map[string]string{"canvas": "#ffffff"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := themeV2Fixture(t)
			tc.change(&m)
			if e := validateExtension(m); e == nil {
				t.Fatal("accepted invalid theme")
			}
		})
	}
}
func TestThemeV2InstalledAndScopedToAccount(t *testing.T) {
	c, _, admin := accountFixture(t)
	admin.enter("setup", map[string]string{"code": c.auth.code, "name": "Owner", "password": accountTestPassword})
	data := extensionZIP(t, exampleThemeV2, "extension.json")
	p, e := parseExtensionPackage(data)
	if e != nil {
		t.Fatal(e)
	}
	if w := extensionUpload(admin, "install", data, p.Checksum, ""); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("PUT", "/api/v1/extensions/"+p.Manifest.ID+"/state", map[string]any{"enabled": true, "checksum": p.Checksum}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := admin.request("PUT", "/api/v1/account/extension-theme", map[string]string{"id": p.Manifest.ID}); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := admin.request("GET", "/api/v1/extensions/appearance", nil)
	var a extensionAppearance
	_ = json.Unmarshal(w.Body.Bytes(), &a)
	w = admin.request("GET", "/api/v1/extensions/theme.css?stamp="+a.Stamp, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "--m-tile-status-local-background:#563310f2;") {
		t.Fatal("installed v2 not rendered", w.Code)
	}
}
