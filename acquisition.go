package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	a "mosaune/extensionkit/acquisitionconfig"
	policy "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/configuration"
	"net/http"
	"strconv"
	"time"
)

func migrateRequests(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 6 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS acquisition_settings(id TEXT PRIMARY KEY,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS acquisition_requests(id TEXT PRIMARY KEY,work_id TEXT NOT NULL REFERENCES works(id),season INTEGER NOT NULL,data TEXT NOT NULL,UNIQUE(work_id,season));
 PRAGMA user_version=6;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func validChoices(cfg arrConfig, opt arrOptions) bool {
	p, r := false, false
	for _, x := range opt.Profiles {
		p = p || x.ID == cfg.Profile
	}
	for _, x := range opt.Roots {
		r = r || x.Path == cfg.Root
	}
	return p && r
}
func (c *catalog) requestLibrary(ctx context.Context, cfg arrConfig) (library, error) {
	var l library
	err := c.db.QueryRowContext(ctx, `SELECT id,name,path,kind FROM libraries WHERE id=?`, cfg.Library).Scan(&l.ID, &l.Name, &l.Path, &l.Kind)
	if err != nil || l.Kind != arrKind(cfg.ID) {
		return l, errors.New("Choisissez un dossier Mosaune correspondant à ce type de média.")
	}
	return l, nil
}

type acquisitionRequest struct {
	Root       string  `json:"root,omitempty"`
	ID         string  `json:"id"`
	WorkID     string  `json:"workId"`
	Title      string  `json:"title"`
	Kind       string  `json:"kind"`
	Connector  string  `json:"connector"`
	Endpoint   string  `json:"endpoint"`
	Library    string  `json:"library"`
	Season     int     `json:"season"`
	ExternalID int     `json:"externalId"`
	RemoteID   int     `json:"remoteId"`
	State      string  `json:"state"`
	Message    string  `json:"message"`
	Created    int64   `json:"created"`
	Updated    int64   `json:"updated"`
	Imported   int     `json:"imported"`
	Total      int     `json:"total"`
	Local      int     `json:"local"`
	Progress   float64 `json:"progress"`
	Uncertain  bool    `json:"uncertain"`
}

func (c *catalog) requests(ctx context.Context) ([]acquisitionRequest, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT data FROM acquisition_requests ORDER BY rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []acquisitionRequest{}
	for rows.Next() {
		var b string
		var v acquisitionRequest
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(b), &v); err != nil {
			return nil, err
		}
		if canLibrary(ctx, v.Library, v.Kind, "view") {
			if u := viewer(ctx); u != nil && !u.Admin {
				v.Endpoint = ""
				v.Root = ""
			}
			result = append(result, v)
		}
	}
	return result, rows.Err()
}
func (c *catalog) saveRequest(ctx context.Context, v acquisitionRequest) error {
	v.Updated = time.Now().Unix()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, `INSERT INTO acquisition_requests(id,work_id,season,data) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data`, v.ID, v.WorkID, v.Season, b)
	return err
}

type requestInput struct {
	WorkID       string `json:"workId"`
	Season       int    `json:"season"`
	Confirmation string `json:"confirmation"`
}
type requestPreview struct {
	Title        string `json:"title"`
	Connector    string `json:"connector"`
	Root         string `json:"root"`
	Profile      string `json:"profile"`
	Library      string `json:"library"`
	Existing     bool   `json:"existing"`
	Seasons      []int  `json:"seasons"`
	Confirmation string `json:"confirmation"`
}

func (c *catalog) previewRequest(ctx context.Context, in requestInput) (requestPreview, arrConfig, arrObject, discoveryWork, int, error) {
	var p requestPreview
	var cfg arrConfig
	var obj arrObject
	var external int
	w, err := c.work(ctx, in.WorkID)
	if err != nil || !videoKind(w.Kind) || in.Season < -1 || in.Season > 1000 || (w.Kind == "films" && in.Season != -1) {
		return p, cfg, obj, w, 0, errors.New("Œuvre ou saison invalide.")
	}
	query := policy.PreviewQuery{Kind: w.Kind, Season: in.Season}
	probe := arrClient{arrConfig{owner: c}}
	ctx, done, err := probe.openSession(ctx)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	defer done()
	plan, err := probe.previewPlan(ctx, query)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	cfg, err = c.arrConfig(ctx, plan.Connector)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	if !cfg.Enabled || !cfg.Configured {
		return p, cfg, obj, w, 0, errors.New("Activez ce service dans Réglages → Téléchargements.")
	}
	l, err := c.requestLibrary(ctx, cfg)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	resolver, ok := c.provider(w.Metadata.Provider).(AcquisitionIdentifierProvider)
	if !ok {
		return p, cfg, obj, w, 0, errors.New("Ce fournisseur ne fournit pas encore les identifiants nécessaires aux demandes.")
	}
	external, err = resolver.AcquisitionID(ctx, c.credential(ctx, w.Metadata.Provider), w.Kind, w.Metadata.ProviderID)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	client := arrClient{cfg}
	cfg.planChecksum = ctx.Value(acquisitionSessionKey{}).(*acquisitionSession).checksum
	obj, opt, existing, err := client.previewResource(ctx, query, plan, external)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	destination, err := plan.Project(query, existing, policy.Bindings{Root: cfg.Root, Profile: cfg.Profile}, obj, opt.Profiles)
	if err != nil {
		return p, cfg, obj, w, 0, err
	}
	p = requestPreview{Title: w.Metadata.Title, Connector: cfg.ID, Root: destination.Root, Profile: destination.Profile, Library: l.Name, Existing: existing, Seasons: destination.Seasons}
	// The confirmation describes the exact destination and scope. Re-read before
	// submitting, rejecting a changed configuration instead of using stale UI data.
	b, _ := json.Marshal([]any{cfg, cfg.revision, cfg.planChecksum, in.WorkID, in.Season, external, number(obj["id"]), p})
	sum := sha256.Sum256(b)
	p.Confirmation = hex.EncodeToString(sum[:])
	return p, cfg, obj, w, external, nil
}
func acquisitionRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	guard := func(w http.ResponseWriter, r *http.Request) bool {
		if !allowed("films") && !allowed("series") {
			http.NotFound(w, r)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/v1/acquisition/settings", func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		result := []acquisitionSettingsView{}
		for _, id := range []string{"radarr", "sonarr"} {
			if allowed(arrKind(id)) {
				cfg, err := c.arrConfig(r.Context(), id)
				if err != nil {
					problem(w, 500, "Réglages indisponibles.")
					return
				}
				cfg.Key = ""
				result = append(result, c.acquisitionSettings(r.Context(), cfg))
			}
		}
		sendJSON(w, 200, result)
	})
	for _, route := range []string{"POST /api/v1/acquisition/test", "PUT /api/v1/acquisition/settings"} {
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			if !guard(w, r) {
				return
			}
			var input struct {
				arrConfig
				FormRevision string `json:"formRevision"`
			}
			if !readJSON(w, r, &input) {
				return
			}
			if !allowed(arrKind(input.ID)) {
				problem(w, 400, "Connecteur inactif.")
				return
			}
			if !c.requestMu.TryLock() {
				problem(w, 409, "Une vérification est en cours. Réessayez dans un instant.")
				return
			}
			defer c.requestMu.Unlock()
			cfg, err := c.configInput(r.Context(), input.arrConfig)
			if err != nil {
				problem(w, 400, err.Error())
				return
			}
			client := arrClient{cfg}
			ctx, done, err := client.openSession(r.Context())
			defer done()
			if err != nil {
				problem(w, 400, err.Error())
				return
			}
			form, revision, err := client.settingsForm(ctx)
			if err != nil {
				problem(w, 400, err.Error())
				return
			}
			if input.FormRevision == "" || input.FormRevision != revision {
				problem(w, 409, "Le formulaire a changé. Rechargez les réglages du connecteur.")
				return
			}
			if err = form.ValidateConnection(cfg.URL, cfg.Key); err != nil {
				problem(w, 400, err.Error())
				return
			}
			opt, err := client.options(ctx)
			if err != nil {
				problem(w, 400, err.Error())
				return
			}
			if r.Method == "PUT" {
				if !validChoices(cfg, opt) {
					problem(w, 400, "Choisissez un profil et un dossier proposés par le service.")
					return
				}
				if _, err = c.requestLibrary(r.Context(), cfg); err != nil {
					problem(w, 400, err.Error())
					return
				}
				if ctx.Err() != nil {
					problem(w, 409, "Le composant a été désactivé. Rechargez les réglages.")
					return
				}
				err = c.saveAcquisitionConfiguration(ctx, &cfg)
				if err != nil {
					if errors.Is(err, configuration.ErrConflict) {
						problem(w, 409, "Les réglages ont changé. Rechargez le formulaire.")
					} else {
						problem(w, 500, "Réglages non enregistrés.")
					}
					return
				}
			}
			revision = acquisitionFormRevision(ctx, cfg.ID, form, cfg.revision)
			cfg.Key = ""
			sendJSON(w, 200, map[string]any{"config": acquisitionSettingsView{arrConfig: cfg, Form: &form, FormRevision: revision}, "options": opt})
		})
	}
	mux.HandleFunc("POST /api/v1/acquisition/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		id := r.PathValue("id")
		if !allowed(arrKind(id)) {
			http.NotFound(w, r)
			return
		}
		if !c.requestMu.TryLock() {
			problem(w, 409, "Une opération est en cours.")
			return
		}
		defer c.requestMu.Unlock()
		if err := (a.ConfigurationStore{DB: c.db}).Pause(r.Context(), id); err != nil {
			problem(w, 500, "Pause non enregistrée.")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/v1/acquisition/requests", func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		items, err := c.requests(r.Context())
		if err != nil {
			problem(w, 500, "Demandes indisponibles.")
			return
		}
		filtered := []acquisitionRequest{}
		for _, v := range items {
			if allowed(v.Kind) {
				filtered = append(filtered, v)
			}
		}
		sendJSON(w, 200, filtered)
	})
	for _, route := range []string{"POST /api/v1/acquisition/preview", "POST /api/v1/acquisition/requests"} {
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			if !guard(w, r) {
				return
			}
			var in requestInput
			if !readJSON(w, r, &in) {
				return
			}
			work, err := c.work(r.Context(), in.WorkID)
			if err != nil || !allowed(work.Kind) {
				problem(w, 404, "Œuvre inconnue.")
				return
			}
			cfgID := "radarr"
			if work.Kind == "series" {
				cfgID = "sonarr"
			}
			dest, e := c.arrConfig(r.Context(), cfgID)
			if e != nil || !canLibrary(r.Context(), dest.Library, work.Kind, "request") {
				problem(w, 403, "Votre compte ne peut pas demander un téléchargement dans cette bibliothèque.")
				return
			}
			if !c.requestMu.TryLock() {
				problem(w, 409, "Une demande ou une vérification est en cours. Réessayez dans un instant.")
				return
			}
			defer c.requestMu.Unlock()
			all, err := c.requests(r.Context())
			if err != nil {
				problem(w, 500, "Demandes indisponibles.")
				return
			}
			for _, v := range all {
				if v.WorkID == in.WorkID && (v.Season == in.Season || v.Season == -1) {
					if r.URL.Path == "/api/v1/acquisition/requests" {
						sendJSON(w, 200, v)
					} else {
						problem(w, 409, "Cette demande existe déjà. Retrouvez son état dans Demandes.")
					}
					return
				}
			}
			if len(all) >= 1000 {
				problem(w, 409, "La limite de 1 000 demandes de ce prototype est atteinte.")
				return
			}
			p, cfg, obj, work, external, err := c.previewRequest(r.Context(), in)
			if err != nil {
				problem(w, 400, err.Error())
				return
			}
			if r.URL.Path == "/api/v1/acquisition/preview" {
				sendJSON(w, 200, p)
				return
			}
			if in.Confirmation == "" || in.Confirmation != p.Confirmation {
				problem(w, 409, "La destination a changé. Fermez puis rouvrez la demande pour la vérifier.")
				return
			}
			v := acquisitionRequest{Root: cfg.Root, ID: rand.Text(), WorkID: in.WorkID, Title: work.Metadata.Title, Kind: work.Kind, Connector: cfg.ID, Endpoint: cfg.URL, Library: cfg.Library, Season: in.Season, ExternalID: external, State: "uncertain", Message: "Envoi en cours ; confirmation du service attendue.", Uncertain: true, Created: time.Now().Unix()}
			if err = c.saveRequest(r.Context(), v); err != nil {
				problem(w, 500, "Demande non enregistrée. Aucun envoi effectué.")
				return
			}
			// Persist the intent before any remote write. A crash or ambiguous timeout
			// must never trigger an automatic replay of a search command.
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 90*time.Second)
			defer cancel()
			v.RemoteID, err = (arrClient{cfg}).submit(ctx, obj, in.Season)
			if err != nil {
				v.Message = "Envoi non confirmé. " + err.Error() + " Vérifiez dans " + cfg.ID + " avant toute nouvelle recherche."
			} else {
				v.State = "waiting"
				v.Message = "Demande acceptée. Le service surveille les disponibilités."
				v.Uncertain = false
			}
			if err = c.saveRequest(ctx, v); err != nil {
				problem(w, 500, "Envoi effectué, suivi non enregistré. Consultez le service avant toute nouvelle recherche.")
				return
			}
			sendJSON(w, 201, v)
		})
	}
	mux.HandleFunc("POST /api/v1/acquisition/sync", func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		if !c.requestMu.TryLock() {
			problem(w, 409, "Une vérification est déjà en cours.")
			return
		}
		defer c.requestMu.Unlock()
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		c.syncRequests(ctx, allowed, true)
		sendJSON(w, 200, map[string]bool{"ok": true})
	})
}

func (c *catalog) acquisitionWorker(ctx context.Context, allowed func(string) bool) {
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		if c.requestMu.TryLock() {
			run, cancel := context.WithTimeout(ctx, 50*time.Second)
			c.syncRequests(run, allowed, false)
			cancel()
			c.requestMu.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}
func (c *catalog) syncRequests(ctx context.Context, allowed func(string) bool, force bool) {
	all, err := c.requests(ctx)
	if err != nil {
		return
	}
	// Old requests rotate with new ones. A per-cycle deadline bounds remote work.
	for i := len(all) - 1; i >= 0; i-- {
		v := all[i]
		if !allowed(v.Kind) || ctx.Err() != nil {
			continue
		}
		interval := 45 * time.Second
		if v.State == "available" {
			interval = 15 * time.Minute
		}
		if !force && time.Since(time.Unix(v.Updated, 0)) < interval {
			continue
		}
		cfg, err := c.arrConfig(ctx, v.Connector)
		if err != nil {
			continue
		}
		if !cfg.Enabled || cfg.URL != v.Endpoint {
			v.State = "paused"
			v.Message = "Suivi en pause : connecteur désactivé ou adresse modifiée. Les téléchargements déjà envoyés continuent dans le service."
			_ = c.saveRequest(ctx, v)
			continue
		}
		err = c.syncRequest(ctx, cfg, &v, allowed)
		if err != nil {
			v.Message = err.Error()
			if v.Uncertain {
				v.State = "uncertain"
			} else {
				v.State = "unreachable"
			}
		}
		_ = c.saveRequest(ctx, v)
	}
}
func (c *catalog) syncRequest(ctx context.Context, cfg arrConfig, v *acquisitionRequest, allowed func(string) bool) error {
	cfg.owner = c
	a := arrClient{cfg}
	ctx, done, err := a.openSession(ctx)
	if err != nil {
		return err
	}
	defer done()
	reader, err := a.trackingReader(ctx)
	if err != nil {
		return err
	}
	defer reader.engine.Close()
	obj, err := a.find(ctx, v.ExternalID)
	if err != nil {
		return err
	}
	observations, episodes, err := a.trackingObservations(ctx, reader, *v, obj)
	if err != nil {
		return err
	}
	if err = applyAcquisitionTracking(ctx, reader, observations, v); err != nil {
		return err
	}
	importedEpisodes := map[string]bool{}
	for _, ep := range observations.Episodes {
		if ep.HasFile && (v.Season < 0 || ep.Season == v.Season) {
			importedEpisodes[strconv.Itoa(ep.Season)+":"+strconv.Itoa(ep.Number)] = true
		}
	}
	if v.Imported > 0 {
		c.metadataMu.Lock()
		defer c.metadataMu.Unlock()
		// One scan per destination every five minutes, regardless of request count.
		if c.requestScans == nil {
			c.requestScans = map[string]time.Time{}
		}
		target := cfg
		target.Library = v.Library
		if time.Since(c.requestScans[v.Library]) >= 5*time.Minute {
			l, e := c.requestLibrary(ctx, target)
			if e == nil {
				_, e = c.scan(ctx, l, false)
				if e == nil {
					c.requestScans[v.Library] = time.Now()
				}
			}
		}
		// Identity comes from a confirmed import and its exact mapped path.
		// Failed path mapping leaves ordinary conservative identification in charge.
		if err := c.prioritizeImport(ctx, a, *v, obj, episodes); err != nil {
			return err
		}
		items, e := c.items(ctx)
		if e != nil {
			return e
		}
		work, e := c.work(ctx, v.WorkID)
		if e != nil {
			return e
		}
		localEpisodes := map[string]bool{}
		for _, m := range items {
			md := m.Metadata
			if m.LibraryID != v.Library || md == nil || md.Provider != work.Metadata.Provider || md.ProviderID != work.Metadata.ProviderID {
				continue
			}
			f, _, openErr := c.openMedia(ctx, m.ID)
			if openErr != nil {
				continue
			}
			f.Close()
			if v.Kind == "films" {
				v.Local = 1
				break
			}
			if v.Season >= 0 && md.Season != v.Season {
				continue
			}
			end := md.EpisodeEnd
			if end < md.Episode {
				end = md.Episode
			}
			for n := md.Episode; n <= end; n++ {
				key := strconv.Itoa(md.Season) + ":" + strconv.Itoa(n)
				if importedEpisodes[key] {
					localEpisodes[key] = true
				}
			}
		}
		if v.Kind == "series" {
			v.Local = len(localEpisodes)
		}
		observations.Local = v.Local
		return applyAcquisitionTracking(ctx, reader, observations, v)
	}
	return nil
}
