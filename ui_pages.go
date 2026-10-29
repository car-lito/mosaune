package main

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

// UI documents are data, never executable markup. The host owns routes,
// authentication, rendering and all media operations.
type uiBlock struct {
	Type   string   `json:"type"`
	Title  string   `json:"title,omitempty"`
	Text   string   `json:"text,omitempty"`
	Target string   `json:"target,omitempty"`
	View   string   `json:"view,omitempty"`
	Facts  []uiFact `json:"facts,omitempty"`
}
type uiFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type uiPage struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Placement   string    `json:"placement"`
	Admin       bool      `json:"admin,omitempty"`
	Kinds       []string  `json:"kinds,omitempty"`
	Blocks      []uiBlock `json:"blocks,omitempty"`
	Owner       string    `json:"owner,omitempty"`
}

//go:embed ui_navigation.json
var uiNavigationJSON []byte

//go:embed examples/reading-room/extension.json
var exampleUIPages []byte

var uiLocalID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)

func uiViewKnown(v string) bool {
	switch v {
	case "home", "films", "series", "music", "watchlist", "favorites":
		return true
	}
	return false
}
func uiKindKnown(k string) bool { return k == "films" || k == "series" || k == "music" }
func uiPackageKind(m extensionManifest) string {
	if m.API == 3 {
		return "pages"
	}
	return "theme"
}
func validateUIPages(m extensionManifest) error {
	fail := errors.New("Pages invalides : utilisez le contrat ui.pages.v1 (identifiants, blocs et destinations déclarés).")
	if len(m.Requires) != 1 || m.Requires[0] != "ui.pages.v1" || len(m.Permissions) != 0 || len(m.Pages) == 0 || len(m.Pages) > 16 {
		return fail
	}
	theme, _ := json.Marshal(m.Theme)
	if string(theme) != "{}" {
		return fail
	}
	seen := map[string]bool{}
	for _, p := range m.Pages {
		if !uiLocalID.MatchString(p.ID) || seen[p.ID] || p.Owner != "" || strings.TrimSpace(p.Label) == "" || len(p.Label) > 80 || len(p.Description) > 600 || len(p.Blocks) > 32 {
			return fail
		}
		seen[p.ID] = true
		if p.Placement != "main" && p.Placement != "personal" && p.Placement != "settings" && p.Placement != "hidden" {
			return fail
		}
		switch p.Icon {
		case "", "home", "films", "series", "music", "bookmark", "heart", "queue", "settings", "server", "search":
		default:
			return fail
		}
		kinds := map[string]bool{}
		for _, k := range p.Kinds {
			if !uiKindKnown(k) || kinds[k] {
				return fail
			}
			kinds[k] = true
		}
		for _, b := range p.Blocks {
			if len(b.Title) > 120 || len(b.Text) > 4000 || len(b.Facts) > 24 {
				return fail
			}
			switch b.Type {
			case "text":
				if b.Target != "" || b.View != "" || len(b.Facts) > 0 {
					return fail
				}
			case "facts":
				if b.Target != "" || b.View != "" || b.Text != "" {
					return fail
				}
				for _, f := range b.Facts {
					if strings.TrimSpace(f.Label) == "" || len(f.Label) > 120 || len(f.Value) > 800 {
						return fail
					}
				}
			case "link":
				if strings.TrimSpace(b.Title) == "" || b.View != "" || len(b.Facts) > 0 {
					return fail
				}
			case "collection":
				if !uiViewKnown(b.View) || b.Target != "" || b.Text != "" || len(b.Facts) > 0 {
					return fail
				}
			default:
				return fail
			}
		}
	}
	for _, p := range m.Pages {
		for _, b := range p.Blocks {
			if b.Type == "link" {
				if strings.HasPrefix(b.Target, "page:") {
					if !seen[strings.TrimPrefix(b.Target, "page:")] {
						return fail
					}
				} else if !uiBuiltinTarget(b.Target) {
					return fail
				}
			}
		}
	}
	return nil
}
func uiBuiltins() []uiPage { var p []uiPage; _ = json.Unmarshal(uiNavigationJSON, &p); return p }
func uiBuiltinTarget(target string) bool {
	for _, p := range uiBuiltins() {
		if p.Placement == "settings" {
			if target == "settings:"+p.ID {
				return true
			}
		} else if target == p.ID {
			return true
		}
	}
	return false
}
func uiAllowed(ctx context.Context, p uiPage, allowed func(string) bool) bool {
	u := viewer(ctx)
	if u == nil || p.Admin && !u.Admin {
		return false
	}
	if len(p.Kinds) == 0 {
		return true
	}
	for _, k := range p.Kinds {
		if allowed(k) && canKind(ctx, k) {
			return true
		}
	}
	return false
}
func uiTargetAllowed(ctx context.Context, target string, pages []uiPage, allowed func(string) bool) bool {
	if strings.HasPrefix(target, "page:") {
		for _, p := range pages {
			if p.ID == strings.TrimPrefix(target, "page:") {
				return uiAllowed(ctx, p, allowed)
			}
		}
		return false
	}
	for _, p := range uiBuiltins() {
		id := p.ID
		if p.Placement == "settings" {
			id = "settings:" + id
		}
		if target == id {
			return uiAllowed(ctx, p, allowed)
		}
	}
	return false
}
func (c *catalog) activeUIPages(ctx context.Context, allowed func(string) bool) ([]uiPage, error) {
	out := []uiPage{}
	if c.extensionSafeMode {
		return out, nil
	}
	rows, e := c.db.QueryContext(ctx, `SELECT manifest FROM extensions WHERE enabled=1 AND (? OR id IN (SELECT extension_id FROM extension_grants WHERE user_id=?)) ORDER BY id`, viewer(ctx).Admin, userID(ctx))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var m extensionManifest
		if json.Unmarshal([]byte(raw), &m) != nil || m.API != 3 || validateExtension(m) != nil {
			continue
		}
		for _, p := range m.Pages {
			if !uiAllowed(ctx, p, allowed) {
				continue
			}
			p.Owner = m.ID
			blocks := []uiBlock{}
			for _, b := range p.Blocks {
				if b.Type == "link" && !uiTargetAllowed(ctx, b.Target, m.Pages, allowed) {
					continue
				}
				if b.Type == "collection" && uiKindKnown(b.View) && (!allowed(b.View) || !canKind(ctx, b.View)) {
					continue
				}
				blocks = append(blocks, b)
			}
			p.Blocks = blocks
			out = append(out, p)
		}
	}
	return out, rows.Err()
}
func (c *catalog) uiRoutes(mux *secureMux, allowed func(string) bool) {
	mux.HandleFunc("GET /api/v1/ui/navigation", func(w http.ResponseWriter, r *http.Request) {
		pages, e := c.activeUIPages(r.Context(), allowed)
		if e != nil {
			problem(w, 500, "Navigation indisponible.")
			return
		}
		nav := []uiPage{}
		for _, p := range uiBuiltins() {
			if uiAllowed(r.Context(), p, allowed) {
				nav = append(nav, p)
			}
		}
		for _, p := range pages {
			if p.Placement != "hidden" {
				p.Blocks = nil
				nav = append(nav, p)
			}
		}
		sendJSON(w, 200, nav)
	})
	mux.HandleFunc("GET /api/v1/ui/pages/{owner}/{page}", func(w http.ResponseWriter, r *http.Request) {
		pages, e := c.activeUIPages(r.Context(), allowed)
		if e != nil {
			problem(w, 500, "Page indisponible.")
			return
		}
		for _, p := range pages {
			if p.Owner == r.PathValue("owner") && p.ID == r.PathValue("page") {
				sendJSON(w, 200, p)
				return
			}
		}
		problem(w, 404, "Cette page n’est plus disponible pour votre compte.")
	})
	mux.HandleFunc("GET /api/v1/extensions/pages-example", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="salon.mosaune.zip"`)
		_, _ = w.Write(uiExampleArchive())
	})
}
func uiExampleArchive() []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, _ := z.Create("extension.json")
	_, _ = f.Write(exampleUIPages)
	_ = z.Close()
	return b.Bytes()
}
