package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The default theme and installed themes use the same renderer and contract.
// Components own layout; these data files own their visual tokens.
//
//go:embed theme/default.json
var defaultThemeJSON []byte

//go:embed theme/contract.json
var themeContractJSON []byte

type themeModes struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}
type themeToken struct {
	Type string `json:"type"`
	Uses []struct {
		Selector string `json:"selector"`
		Property string `json:"property"`
		Source   string `json:"source"`
	} `json:"uses"`
}

var themeContract struct {
	Version int                   `json:"version"`
	Tokens  map[string]themeToken `json:"tokens"`
}
var defaultTheme extensionTheme

func init() {
	if err := json.Unmarshal(defaultThemeJSON, &defaultTheme); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(themeContractJSON, &themeContract); err != nil {
		panic(err)
	}
}

var themeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var themeHex = regexp.MustCompile(`^#[a-fA-F0-9]{6}([a-fA-F0-9]{2})?$`)
var themeFont = regexp.MustCompile(`^[\pL\pN -]{1,60}$`)
var themeMeasure = regexp.MustCompile(`^(-?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+))(px|rem|vw|%)$`)
var themeSections = map[string]bool{"home": true, "films": true, "series": true, "music": true, "settings": true, "watchlist": true, "favorites": true, "playlists": true, "requests": true}

func mergeTheme(t extensionTheme) extensionTheme {
	result := extensionTheme{Palette: maps.Clone(defaultTheme.Palette), Fonts: maps.Clone(defaultTheme.Fonts), Radii: maps.Clone(defaultTheme.Radii), Tokens: maps.Clone(defaultTheme.Tokens), Sections: t.Sections}
	maps.Copy(result.Palette, t.Palette)
	maps.Copy(result.Fonts, t.Fonts)
	maps.Copy(result.Radii, t.Radii)
	maps.Copy(result.Tokens, t.Tokens)
	return result
}
func validThemeLength(v string, kind string) bool {
	m := themeMeasure.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	n, _ := strconv.ParseFloat(m[1], 64)
	switch kind {
	case "radius":
		return (m[2] == "px" && n >= 0 && n <= 100) || (m[2] == "rem" && n >= 0 && n <= 6) || (m[2] == "%" && n >= 0 && n <= 50)
	case "spacing":
		return (m[2] == "px" && n >= -4 && n <= 20) || (m[2] == "rem" && n >= -.25 && n <= 1.25)
	case "decorative-spacing":
		return (m[2] == "px" && n >= -12 && n <= 20) || (m[2] == "rem" && n >= -.75 && n <= 1.25)
	case "size":
		return (m[2] == "px" && n >= 4 && n <= 128) || (m[2] == "rem" && n >= .25 && n <= 8) || (m[2] == "vw" && n >= 0 && n <= 10)
	}
	return false
}
func resolveThemeToken(t extensionTheme, key, value, mode string) (string, error) {
	spec, ok := themeContract.Tokens[key]
	if !ok {
		return "", fmt.Errorf("Réglage de thème inconnu : %s.", key)
	}
	fail := func() (string, error) {
		return "", fmt.Errorf("Valeur ou référence invalide pour %s (%s).", key, spec.Type)
	}
	if strings.HasPrefix(value, "$") {
		parts := strings.SplitN(value[1:], ".", 2)
		if len(parts) != 2 {
			return fail()
		}
		switch parts[0] {
		case "palette":
			if spec.Type != "color" {
				return fail()
			}
			color, ok := t.Palette[parts[1]]
			if !ok {
				return fail()
			}
			value = color.Light
			if mode == "dark" {
				value = color.Dark
			}
		case "fonts":
			if spec.Type != "font" {
				return fail()
			}
			families, ok := t.Fonts[parts[1]]
			if !ok || len(families) == 0 || len(families) > 6 {
				return fail()
			}
			escaped := []string{}
			for _, family := range families {
				if !themeFont.MatchString(family) {
					return fail()
				}
				switch family {
				case "serif", "sans-serif", "monospace", "system-ui", "cursive", "fantasy":
					escaped = append(escaped, family)
				default:
					escaped = append(escaped, strconv.Quote(family))
				}
			}
			return strings.Join(escaped, ","), nil
		case "radii":
			if spec.Type != "radius" {
				return fail()
			}
			var ok bool
			value, ok = t.Radii[parts[1]]
			if !ok {
				return fail()
			}
		default:
			return fail()
		}
	}
	switch spec.Type {
	case "color":
		if themeHex.MatchString(value) {
			return strings.ToLower(value), nil
		}
	case "radius":
		if validThemeLength(value, "radius") {
			return value, nil
		}
	case "spacing", "decorative-spacing":
		if value == "normal" || validThemeLength(value, spec.Type) {
			return value, nil
		}
	case "size":
		if validThemeLength(value, "size") && !strings.HasSuffix(value, "vw") {
			return value, nil
		}
		if strings.HasPrefix(value, "clamp(") && strings.HasSuffix(value, ")") {
			parts := strings.Split(value[6:len(value)-1], ",")
			if len(parts) == 3 {
				valid := true
				for _, p := range parts {
					valid = valid && validThemeLength(strings.TrimSpace(p), "size")
				}
				if valid {
					return value, nil
				}
			}
		}
	case "weight":
		if value == "normal" || value == "bold" {
			return value, nil
		}
		if n, e := strconv.Atoi(value); e == nil && n >= 100 && n <= 900 {
			return value, nil
		}
	case "style":
		if value == "normal" || value == "italic" {
			return value, nil
		}
	case "line":
		if n, e := strconv.ParseFloat(value, 64); e == nil && n >= .8 && n <= 3 {
			return value, nil
		}
	case "opacity":
		if n, e := strconv.ParseFloat(value, 64); e == nil && n >= .25 && n <= 1 {
			return value, nil
		}
	case "decorative-opacity":
		if n, e := strconv.ParseFloat(value, 64); e == nil && n >= .01 && n <= 1 {
			return value, nil
		}
	}
	return fail()
}
func validateThemeV2(input extensionTheme) error {
	if len(input.Light) > 0 || len(input.Dark) > 0 {
		return fmt.Errorf("Un thème v2 utilise palette, fonts, radii, tokens et sections ; les palettes v1 light/dark ne s’y mélangent pas.")
	}
	if len(input.Palette) > 256 || len(input.Fonts) > 32 || len(input.Radii) > 64 || len(input.Sections) > len(themeSections) {
		return fmt.Errorf("Trop de valeurs dans le thème.")
	}
	for k, c := range input.Palette {
		if !themeName.MatchString(k) || !themeHex.MatchString(c.Light) || !themeHex.MatchString(c.Dark) {
			return fmt.Errorf("Palette %s invalide : deux couleurs #RRGGBB ou #RRGGBBAA sont requises.", k)
		}
	}
	for k, f := range input.Fonts {
		if !themeName.MatchString(k) || len(f) < 1 || len(f) > 6 {
			return fmt.Errorf("Police %s invalide.", k)
		}
		for _, name := range f {
			if !themeFont.MatchString(name) {
				return fmt.Errorf("Nom de police invalide : %s.", k)
			}
		}
	}
	for k, r := range input.Radii {
		if !themeName.MatchString(k) || !validThemeLength(r, "radius") {
			return fmt.Errorf("Arrondi %s invalide.", k)
		}
	}
	merged := mergeTheme(input)
	for _, mode := range []string{"light", "dark"} {
		for k, v := range merged.Tokens {
			if _, e := resolveThemeToken(merged, k, v, mode); e != nil {
				return e
			}
		}
		if e := validateThemeContrast(merged, nil, mode); e != nil {
			return e
		}
		for section, overrides := range input.Sections {
			if !themeSections[section] {
				return fmt.Errorf("Rubrique inconnue : %s.", section)
			}
			for k, v := range overrides {
				if _, e := resolveThemeToken(merged, k, v, mode); e != nil {
					return e
				}
			}
			if e := validateThemeContrast(merged, overrides, mode); e != nil {
				return fmt.Errorf("Rubrique %s : %w", section, e)
			}
		}
	}
	return nil
}
func validateThemeContrast(t extensionTheme, overrides map[string]string, mode string) error {
	// Guard the main controls, also in settings; all themes remain removable.
	for _, pair := range [][2]string{{"app.color", "app.background"}, {"sidebar.color", "sidebar.background"}, {"button.primary.color", "button.primary.background"}, {"button.secondary.color", "button.secondary.background"}, {"navigation.item.selected.color", "navigation.item.selected.background"}, {"navigation.item.hover.color", "navigation.item.hover.background"}} {
		colors := [2]string{}
		for i, k := range pair {
			v := t.Tokens[k]
			if override, ok := overrides[k]; ok {
				v = override
			}
			resolved, e := resolveThemeToken(t, k, v, mode)
			if e != nil {
				return e
			}
			colors[i] = resolved
		}
		if len(colors[0]) != 7 || len(colors[1]) != 7 || colorContrast(colors[0], colors[1]) < 4.5 {
			return fmt.Errorf("Contraste insuffisant (%s) entre %s et %s : utilisez des couleurs opaques et lisibles.", mode, pair[0], pair[1])
		}
	}
	return nil
}
func themeDeclarations(t extensionTheme, tokens map[string]string, mode string) string {
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v, e := resolveThemeToken(t, k, tokens[k], mode)
		if e != nil {
			continue
		}
		fmt.Fprintf(&b, "--m-%s:%s;", strings.ReplaceAll(k, ".", "-"), v)
	}
	return b.String()
}
func renderTheme(t extensionTheme, scope string) string {
	t = mergeTheme(t)
	var b strings.Builder
	for _, mode := range []string{"light", "dark"} {
		selector := `:root[data-theme="` + mode + `"]`
		if scope != "" {
			selector += " " + scope
		} else if mode == "light" {
			selector = ":root"
		}
		b.WriteString(selector + "{" + themeDeclarations(t, t.Tokens, mode) + "}\n")
		if scope != "" {
			sections := make([]string, 0, len(t.Sections))
			for section := range t.Sections {
				sections = append(sections, section)
			}
			sort.Strings(sections)
			for _, section := range sections {
				b.WriteString(selector + `[data-ui-section="` + section + `"]{` + themeDeclarations(t, t.Sections[section], mode) + "}\n")
			}
		}
	}
	return b.String()
}
func legacyTheme(t extensionTheme) extensionTheme {
	result := extensionTheme{Palette: map[string]themeModes{}}
	mapping := map[string]string{"canvas": "canvas", "surface": "surface", "surface-muted": "surface", "surface-hover": "surface", "ink": "ink", "muted": "muted", "line": "line", "primary": "accent", "primary-hover": "accent", "on-primary": "accentInk", "secondary": "surface", "on-secondary": "ink", "sidebar": "sidebar", "sidebar-text": "sidebarInk", "sidebar-muted": "sidebarInk", "sidebar-hover": "sidebar", "sidebar-active": "sidebar", "placeholder": "surface", "accent": "accent", "focus": "accent", "success": "surface", "on-success": "ink", "warning": "surface", "on-warning": "ink", "badge-local": "accent", "on-badge-local": "accentInk", "badge-discover": "sidebar", "on-badge-discover": "sidebarInk"}
	for dst, src := range mapping {
		result.Palette[dst] = themeModes{Light: t.Light[src], Dark: t.Dark[src]}
	}
	return result
}
func themeForManifest(m extensionManifest) extensionTheme {
	if m.API == 1 {
		return legacyTheme(m.Theme)
	}
	return m.Theme
}
func themePreviewFor(m extensionManifest) map[string]map[string]string {
	t := mergeTheme(themeForManifest(m))
	out := map[string]map[string]string{"light": {}, "dark": {}}
	for k, token := range map[string]string{"canvas": "app.background", "ink": "app.color", "muted": "text.action.color", "accent": "button.primary.background"} {
		for _, mode := range []string{"light", "dark"} {
			v, e := resolveThemeToken(t, token, t.Tokens[token], mode)
			if e != nil {
				v = "#777777"
			}
			out[mode][k] = v
		}
	}
	return out
}
