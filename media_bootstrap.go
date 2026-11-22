package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mosaune/extensionkit/configuration"
	"strings"
)

// Internal first-install composition. An existing catalogue always keeps its
// recorded choices; this is not a request to enable or remove installed media.
type mediaBootstrapSelection struct {
	Families []string `json:"families"`
	Home     bool     `json:"home"`
}

func suppliedMediaSelection() mediaBootstrapSelection {
	return mediaBootstrapSelection{Families: []string{"films", "series", "music"}, Home: true}
}

func parseInitialMedia(value string, home bool) (mediaBootstrapSelection, error) {
	selection := mediaBootstrapSelection{Families: []string{}, Home: home}
	if strings.TrimSpace(value) != "none" {
		for _, part := range strings.Split(value, ",") {
			id := strings.TrimSpace(part)
			if id == "" {
				return selection, errors.New("Indiquez les familles initiales, ou none pour aucune.")
			}
			selection.Families = append(selection.Families, id)
		}
	}
	return selection, validateMediaSelection(selection)
}

func validateMediaSelection(selection mediaBootstrapSelection) error {
	definitions, err := mediaBundleDefinitions()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, d := range definitions {
		known[d.Kind] = true
	}
	seen := map[string]bool{}
	for _, id := range selection.Families {
		if !known[id] || seen[id] {
			return errors.New("Composition initiale des familles invalide.")
		}
		seen[id] = true
	}
	return nil
}

func prepareMediaBootstrap(ctx context.Context, db *sql.DB, oldVersion int, selection mediaBootstrapSelection) error {
	if err := validateMediaSelection(selection); err != nil {
		return err
	}
	if selection.Families == nil {
		selection.Families = []string{}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS media_bootstrap(id INTEGER PRIMARY KEY CHECK(id=1),selection TEXT NOT NULL,completed INTEGER NOT NULL CHECK(completed IN(0,1)))`); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM media_bootstrap WHERE id=1)").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		var used bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM libraries) OR EXISTS(SELECT 1 FROM media)").Scan(&used); err != nil {
			return err
		}
		raw, _ := json.Marshal(selection)
		if _, err = tx.ExecContext(ctx, "INSERT INTO media_bootstrap VALUES(1,?,?)", string(raw), oldVersion != 0 || used); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func pendingMediaBootstrap(ctx context.Context, db configuration.Connection) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='media_bootstrap')").Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	var completed bool
	err := db.QueryRowContext(ctx, "SELECT completed FROM media_bootstrap WHERE id=1").Scan(&completed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return !completed, err
}

func deferMediaSeed(ctx context.Context, tx *sql.Tx, version int) (bool, error) {
	pending, err := pendingMediaBootstrap(ctx, tx)
	if err != nil || !pending {
		return false, err
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", version))
	return true, err
}

// All families, shared components, screens and their completion marker commit
// together. Use the normal family planner and installer, not a second recipe.
func finishMediaBootstrap(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw string
	var completed bool
	if err = tx.QueryRowContext(ctx, "SELECT selection,completed FROM media_bootstrap WHERE id=1").Scan(&raw, &completed); err != nil {
		return err
	}
	if !completed {
		var selection mediaBootstrapSelection
		if len(raw) > 65536 || json.Unmarshal([]byte(raw), &selection) != nil || selection.Families == nil {
			return errors.New("Composition initiale des familles illisible.")
		}
		if err = validateMediaSelection(selection); err != nil {
			return err
		}
		definitions, err := mediaBundleDefinitions()
		if err != nil {
			return err
		}
		for _, kind := range selection.Families {
			var family mediaBundleManifest
			for _, d := range definitions {
				if d.Kind == kind {
					family = d
					break
				}
			}
			packages, chosen, revision, err := (&catalog{}).mediaSetInputs(ctx, tx)
			if err != nil {
				return err
			}
			screens, err := bundleScreens(ctx, tx)
			if err != nil {
				return err
			}
			plan, err := planBundle(ctx, tx, definitions, family, "prepare", packages, chosen, revision, false, screens)
			if err != nil {
				return err
			}
			if len(plan.Blockers) > 0 {
				return fmt.Errorf("Installation initiale de %s impossible : %s", kind, plan.Blockers[0])
			}
			if err = applyMediaBundlePlanTx(ctx, tx, plan, "prepare", revision); err != nil {
				return err
			}
		}
		if selection.Home {
			found := false
			for _, p := range screenPackages() {
				for _, page := range p.Manifest.Pages {
					if page.Screen != "home" {
						continue
					}
					found = true
					data, _ := json.Marshal(p.Manifest)
					if _, err = tx.ExecContext(ctx, "INSERT INTO extensions VALUES(?,?,?,1)", p.Manifest.ID, string(data), p.Checksum); err != nil {
						return err
					}
					if _, err = tx.ExecContext(ctx, "INSERT INTO extension_grants(user_id,extension_id) SELECT id,? FROM accounts", p.Manifest.ID); err != nil {
						return err
					}
				}
			}
			if !found {
				return errors.New("Écran d’accueil initial indisponible.")
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE media_bootstrap SET completed=1 WHERE id=1"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=52"); err != nil {
		return err
	}
	return tx.Commit()
}
