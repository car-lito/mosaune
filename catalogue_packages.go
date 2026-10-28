package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	catalogue "mosaune/extensionkit/capabilities/catalogue/v1"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/registry"
	runtime "mosaune/extensionkit/runtime"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"
)

var catalogueKinds = []string{"films", "series", "music"}

func migrateCataloguePackages(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 22 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx := context.Background()
	store := registry.SQLStore{DB: tx}
	for _, kind := range catalogueKinds {
		slot := "catalogue-" + kind
		s, e := store.Load(ctx)
		if e != nil {
			return e
		}
		s, e = installRecognition(ctx, tx, slot, s.Revision)
		if e != nil {
			return e
		}
		id, e := recognitionID(slot)
		if e != nil {
			return e
		}
		if e = registryWanted(ctx, store, s, id, true); e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT INTO recognition_selection(kind,id) VALUES(?,?)", slot, id); e != nil {
			return e
		}
	}
	if _, err = tx.Exec("PRAGMA user_version=22"); err != nil {
		return err
	}
	return tx.Commit()
}
func catalogueImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	d := m.Module
	if len(m.Contributions) != 1 || m.Contributions[0].Role != catalogue.Capability || !slices.Contains(catalogueKinds, m.Contributions[0].Target) || len(d.Capabilities) != 1 || d.Capabilities[0] != catalogue.Capability || len(d.Services) != 0 || len(d.Dependencies) != 0 {
		return "", errors.New("Module de catalogue incompatible : métadonnées autorisées uniquement, sans service ni dépendance.")
	}
	return "catalogue-" + m.Contributions[0].Target, nil
}

type catalogueProjection struct {
	catalogue.Item
	Owner string `json:"owner"`
}
type catalogueCacheEntry struct {
	fingerprint string
	projections []catalogueProjection
}

type cataloguer struct {
	engine     *runtime.Engine
	capability string
	id         string
}

func loadCataloguer(ctx context.Context, d protocol.Descriptor, code []byte) (*cataloguer, error) {
	return loadFactProjector(ctx, d, code, catalogue.Capability)
}
func loadFactProjector(ctx context.Context, d protocol.Descriptor, code []byte, capability string) (*cataloguer, error) {
	if len(d.Capabilities) != 1 || d.Capabilities[0] != capability || len(d.Services) != 0 || len(d.Dependencies) != 0 {
		return nil, errors.New("Contrat de catalogue incompatible.")
	}
	limits := runtime.DefaultLimits()
	limits.Modules = 1
	if capability == "media.detail-experience.v1" {
		limits.Timeout = 10 * time.Second
		limits.InputBytes = 1 << 20
		limits.OutputBytes = 1 << 20
	}
	limits.Interpreter = true
	e, err := runtime.New(ctx, limits)
	if err != nil {
		return nil, err
	}
	r := &cataloguer{engine: e, id: d.ID, capability: capability}
	ok := false
	defer func() {
		if !ok {
			e.Close()
		}
	}()
	if err = e.Load(ctx, d, code); err != nil {
		return nil, err
	}
	if err = e.Enable(d.ID); err != nil {
		return nil, err
	}
	var info struct {
		Version int `json:"version"`
	}
	if err = r.call(ctx, "describe", struct{}{}, &info); err != nil {
		return nil, err
	}
	if info.Version != 1 {
		return nil, errors.New("Version de projection incompatible.")
	}
	ok = true
	return r, nil
}
func (r *cataloguer) call(ctx context.Context, operation string, payload, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	v, err := r.engine.Call(ctx, r.id, protocol.Request{Protocol: protocol.Version, ID: "catalogue-" + operation, Capability: r.capability, Operation: operation, Payload: b})
	if err != nil {
		return err
	}
	if v.Response.Error != nil || protocol.DecodeOne(v.Response.Result, out) != nil {
		return errors.New("Projection de catalogue invalide.")
	}
	return ctx.Err()
}
func (r *cataloguer) project(ctx context.Context, q catalogue.Query) (catalogue.Result, error) {
	var out catalogue.Result
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := r.call(ctx, "project", q, &out); err != nil {
		return out, err
	}
	return out, out.Validate(q)
}

// Sanitize bounded copies only; never rewrite the stored catalogue.
func catalogueText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(s, "�"))
	if len(s) > 512 {
		s = strings.ToValidUTF8(s[:512], "")
	}
	return s
}
func catalogueFacts(m media) catalogue.Facts {
	f := catalogue.Facts{ID: m.ID, Name: m.Name, Filename: path.Base(strings.ReplaceAll(m.Filename, "\\", "/"))}
	if d := m.Metadata; d != nil {
		f.Title = d.Title
		f.Provider = d.Provider
		f.ProviderID = d.ProviderID
		f.Series = d.Series
		f.Artist = d.Artist
		f.AlbumArtist = d.AlbumArtist
		f.Album = d.Album
		f.Year = d.Year
		f.Season = d.Season
		f.Episode = d.Episode
		f.Disc = d.Disc
		f.Track = d.Track
	}
	for _, s := range []*string{&f.ID, &f.Name, &f.Filename, &f.Title, &f.Provider, &f.ProviderID, &f.Series, &f.Artist, &f.AlbumArtist, &f.Album} {
		*s = catalogueText(*s)
	}
	return f
}

// Invoked only on already permission-filtered entries. No VM remains resident.
// One projection request per catalogue at a time limits memory under parallel clients.
func (c *catalog) projectCatalogue(ctx context.Context, items []media) []string {
	c.catalogueMu.Lock()
	defer c.catalogueMu.Unlock()
	warnings := []string{}
	for _, kind := range catalogueKinds {
		indices := []int{}
		for i := range items {
			if items[i].Kind == kind {
				items[i].Catalogue = nil
				indices = append(indices, i)
			}
		}
		if len(indices) == 0 {
			continue
		}
		err := func() error {
			timeout, stop := context.WithTimeout(ctx, 15*time.Second)
			defer stop()
			call, done, d, code, err := c.beginRecognition(timeout, "catalogue-"+kind)
			if err != nil {
				return err
			}
			defer done()
			facts := make([]catalogue.Facts, len(indices))
			for j, i := range indices {
				facts[j] = catalogueFacts(items[i])
			}
			digest := sha256.New()
			digest.Write([]byte(d.ID + "@" + d.Version))
			digest.Write([]byte(recognitionHash(code)))
			if err := json.NewEncoder(digest).Encode(facts); err != nil {
				return err
			}
			fingerprint := hex.EncodeToString(digest.Sum(nil))
			if cached, ok := c.catalogueCache[kind]; ok && cached.fingerprint == fingerprint {
				unlock, err := c.recognitionCommit(call)
				if err != nil {
					return err
				}
				defer unlock()
				for j, i := range indices {
					v := cached.projections[j]
					items[i].Catalogue = &v
				}
				return nil
			}
			delete(c.catalogueCache, kind)
			r, err := loadCataloguer(call, d, code)
			if err != nil {
				return err
			}
			defer r.engine.Close()
			projections := make([]catalogueProjection, 0, len(indices))
			for start := 0; start < len(indices); {
				q := catalogue.Query{}
				for start < len(indices) && len(q.Items) < catalogue.BatchSize {
					q.Items = append(q.Items, facts[start])
					encoded, _ := json.Marshal(q)
					if len(encoded) > 48<<10 {
						q.Items = q.Items[:len(q.Items)-1]
						break
					}
					start++
				}
				if len(q.Items) == 0 {
					return errors.New("Métadonnées trop volumineuses.")
				}
				out, err := r.project(call, q)
				if err != nil {
					return err
				}
				for _, v := range out.Items {
					projections = append(projections, catalogueProjection{Item: v, Owner: d.ID})
				}
			}
			// A disable/update during computation must discard the entire late result.
			unlock, err := c.recognitionCommit(call)
			if err != nil {
				return err
			}
			defer unlock()
			for j, i := range indices {
				v := projections[j]
				items[i].Catalogue = &v
			}
			// One bounded projection per known slot, no persistent VM and no cache on disk.
			raw, _ := json.Marshal(projections)
			if len(raw) <= 2<<20 {
				if c.catalogueCache == nil {
					c.catalogueCache = map[string]catalogueCacheEntry{}
				}
				c.catalogueCache[kind] = catalogueCacheEntry{fingerprint, projections}
			}
			return nil
		}()
		if err != nil {
			delete(c.catalogueCache, kind)
			warnings = append(warnings, "Organisation du catalogue "+kind+" indisponible. Les fichiers restent affichés individuellement. Vérifiez son extension dans les réglages.")
		}
	}
	return warnings
}
