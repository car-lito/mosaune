package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"
)

// Pending work is derived from persisted metadata/revisions, not an in-memory
// queue. Only failures and trusted acquisition priorities need extra rows.
func migrateMetadataQueue(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= 7 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS metadata_queue_state(id INTEGER PRIMARY KEY CHECK(id=1),paused INTEGER NOT NULL DEFAULT 0,retry_at INTEGER NOT NULL DEFAULT 0,message TEXT NOT NULL DEFAULT '',updated INTEGER NOT NULL DEFAULT 0);
 INSERT OR IGNORE INTO metadata_queue_state(id) VALUES(1);
 CREATE TABLE IF NOT EXISTS metadata_failures(media_id TEXT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,revision TEXT NOT NULL,attempts INTEGER NOT NULL,retry_at INTEGER NOT NULL,message TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS metadata_priorities(media_id TEXT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,revision TEXT NOT NULL,work_id TEXT NOT NULL,season INTEGER NOT NULL,episode INTEGER NOT NULL);
 PRAGMA user_version=7;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

type metadataCounts struct {
	LibraryID string `json:"libraryId"`
	Name      string `json:"name"`
	Total     int    `json:"total"`
	Matched   int    `json:"matched"`
	Review    int    `json:"review"`
	Pending   int    `json:"pending"`
	Failed    int    `json:"failed"`
}
type metadataQueueStatus struct {
	State     string               `json:"state"`
	Paused    bool                 `json:"paused"`
	Current   string               `json:"current"`
	Message   string               `json:"message"`
	RetryAt   int64                `json:"retryAt"`
	Updated   int64                `json:"updated"`
	Libraries []metadataCounts     `json:"libraries"`
	Errors    []metadataQueueError `json:"errors"`
}
type metadataQueueError struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Attempts int    `json:"attempts"`
}

const metadataQueueJoin = ` FROM media m JOIN libraries l ON l.id=m.library_id LEFT JOIN metadata d ON d.media_id=m.id AND d.revision=m.revision LEFT JOIN metadata_failures f ON f.media_id=m.id AND f.revision=m.revision `
const metadataPending = `m.available=1 AND COALESCE(json_extract(d.data,'$.providerId'),'')='' AND COALESCE(json_extract(d.data,'$.matchState'),'')<>'review'`

func (c *catalog) queueProviders(ctx context.Context, allowed func(string) bool) map[string]MetadataProvider {
	providers := map[string]MetadataProvider{}
	for _, p := range c.providers {
		info := p.Info()
		if !providerAvailable(p) {
			continue
		}
		if info.RequiresCredential && c.credential(ctx, info.ID) == "" {
			continue
		}
		for _, kind := range info.Kinds {
			if videoKind(kind) && allowed(kind) && providers[kind] == nil {
				providers[kind] = p
			}
		}
	}
	return providers
}
func (c *catalog) metadataStatus(ctx context.Context, allowed func(string) bool) (metadataQueueStatus, error) {
	s := metadataQueueStatus{State: "idle", Libraries: []metadataCounts{}, Errors: []metadataQueueError{}}
	if err := c.db.QueryRowContext(ctx, `SELECT paused,retry_at,message,updated FROM metadata_queue_state WHERE id=1`).Scan(&s.Paused, &s.RetryAt, &s.Message, &s.Updated); err != nil {
		return s, err
	}
	rows, err := c.db.QueryContext(ctx, `SELECT l.id,l.name,l.kind,COUNT(*),SUM(CASE WHEN COALESCE(json_extract(d.data,'$.providerId'),'')<>'' THEN 1 ELSE 0 END),SUM(CASE WHEN COALESCE(json_extract(d.data,'$.providerId'),'')='' AND json_extract(d.data,'$.matchState')='review' THEN 1 ELSE 0 END),SUM(CASE WHEN COALESCE(json_extract(d.data,'$.providerId'),'')='' AND COALESCE(json_extract(d.data,'$.matchState'),'')<>'review' AND COALESCE(f.attempts,0)>0 THEN 1 ELSE 0 END)`+metadataQueueJoin+`WHERE m.available=1 AND l.kind IN ('films','series') GROUP BY l.id ORDER BY l.name`)
	if err != nil {
		return s, err
	}
	pending, failed := 0, 0
	for rows.Next() {
		var v metadataCounts
		var kind string
		if err = rows.Scan(&v.LibraryID, &v.Name, &kind, &v.Total, &v.Matched, &v.Review, &v.Failed); err != nil {
			break
		}
		if allowed(kind) {
			v.Pending = v.Total - v.Matched - v.Review - v.Failed
			pending += v.Pending
			failed += v.Failed
			s.Libraries = append(s.Libraries, v)
		}
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	if rowErr != nil {
		return s, rowErr
	}
	rows, err = c.db.QueryContext(ctx, `SELECT m.id,COALESCE(json_extract(d.data,'$.title'),m.name),f.message,f.attempts,l.kind`+metadataQueueJoin+`WHERE `+metadataPending+` AND f.attempts>0 ORDER BY f.attempts DESC,m.name LIMIT 50`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var e metadataQueueError
		var kind string
		if err = rows.Scan(&e.ID, &e.Title, &e.Message, &e.Attempts, &kind); err != nil {
			break
		}
		if allowed(kind) {
			s.Errors = append(s.Errors, e)
		}
	}
	rowErr = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	if rowErr != nil {
		return s, rowErr
	}
	c.metadataActivityMu.Lock()
	s.Current = c.metadataCurrent
	c.metadataActivityMu.Unlock()
	switch {
	case s.Paused:
		s.State = "paused"
		s.Message = "Identification en pause. La fiche déjà commencée peut encore se terminer."
	case s.Current != "":
		s.State = "running"
	case pending+failed == 0:
		s.State = "complete"
		s.Message = "Identification terminée. Les correspondances incertaines restent à vérifier."
	case len(c.queueProviders(ctx, allowed)) == 0:
		s.State = "unconfigured"
		s.Message = "Activez et configurez un fournisseur vidéo, puis vérifiez ses accès dans les paramètres de l’extension pour identifier les fiches en attente."
	case s.RetryAt > time.Now().Unix():
		s.State = "retry"
	case pending == 0 && failed > 0:
		s.State = "errors"
		s.Message = "Des fiches restent en erreur. Les essais automatiques sont limités à trois par fichier ; vous pouvez relancer les erreurs après avoir vérifié le fournisseur."
	default:
		s.State = "waiting"
		s.Message = "Les fiches en attente seront traitées automatiquement. Les erreurs persistantes peuvent être relancées séparément."
	}
	return s, nil
}
func (c *catalog) metadataWorker(ctx context.Context, allowed func(string) bool) {
	// Sequential work; the provider retains its own request pacing and cache.
	for {
		delay := c.metadataStep(ctx, allowed)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (c *catalog) metadataStep(ctx context.Context, allowed func(string) bool) time.Duration {
	if ctx.Err() != nil {
		return time.Second
	}
	if !c.metadataMu.TryLock() {
		return time.Second
	}
	defer c.metadataMu.Unlock()
	var paused bool
	var retryAt int64
	if c.db.QueryRowContext(ctx, `SELECT paused,retry_at FROM metadata_queue_state WHERE id=1`).Scan(&paused, &retryAt) != nil || paused {
		return 2 * time.Second
	}
	if retryAt > time.Now().Unix() {
		return 2 * time.Second
	}
	providers := c.queueProviders(ctx, allowed)
	if len(providers) == 0 {
		return 5 * time.Second
	}
	film, series := providers["films"] != nil, providers["series"] != nil
	var id string
	err := c.db.QueryRowContext(ctx, `SELECT m.id`+metadataQueueJoin+`LEFT JOIN metadata_priorities p ON p.media_id=m.id AND p.revision=m.revision WHERE `+metadataPending+` AND ((l.kind='films' AND ?) OR (l.kind='series' AND ?)) AND COALESCE(f.attempts,0)<3 AND COALESCE(f.retry_at,0)<=? ORDER BY CASE WHEN p.media_id IS NULL THEN 1 ELSE 0 END,m.added_at DESC,m.id LIMIT 1`, film, series, time.Now().Unix()).Scan(&id)
	if err != nil {
		return 3 * time.Second
	}
	m, err := c.mediaMetadata(ctx, id)
	if err != nil {
		return time.Second
	}
	c.metadataActivityMu.Lock()
	c.metadataCurrent = m.Metadata.Title
	c.metadataActivityMu.Unlock()
	defer func() { c.metadataActivityMu.Lock(); c.metadataCurrent = ""; c.metadataActivityMu.Unlock() }()
	run, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	err = c.identifyQueued(run, m, providers[m.Kind])
	if ctx.Err() != nil || (errors.Is(err, errProviderDisabled) || errors.Is(err, errProviderModuleUnavailable)) || errors.Is(err, errProviderAccessDenied) {
		return time.Second
	} // Interrupted work stays pending, never a permanent error.
	if err != nil {
		_, _ = c.db.ExecContext(ctx, `INSERT INTO metadata_failures(media_id,revision,attempts,retry_at,message) VALUES(?,?,1,?,?) ON CONFLICT(media_id) DO UPDATE SET attempts=CASE WHEN metadata_failures.revision=excluded.revision THEN metadata_failures.attempts+1 ELSE 1 END,revision=excluded.revision,retry_at=excluded.retry_at,message=excluded.message`, m.ID, m.Revision, time.Now().Add(time.Minute).Unix(), truncate(err.Error(), 500))
		// Avoid hammering an unavailable or throttled provider with other titles.
		_, _ = c.db.ExecContext(ctx, `UPDATE metadata_queue_state SET retry_at=?,message=?,updated=unixepoch() WHERE id=1`, time.Now().Add(30*time.Second).Unix(), truncate(err.Error(), 500))
		return 2 * time.Second
	}
	_, _ = c.db.ExecContext(ctx, `DELETE FROM metadata_failures WHERE media_id=?; DELETE FROM metadata_priorities WHERE media_id=?; UPDATE metadata_queue_state SET retry_at=0,message='',updated=unixepoch() WHERE id=1`, m.ID, m.ID)
	return 500 * time.Millisecond
}
func (c *catalog) identifyQueued(ctx context.Context, m media, p MetadataProvider) error {
	ctx, done, e := beginProvider(ctx, p)
	if e != nil {
		return e
	}
	defer done()
	// A path confirmed by an acquisition service supplies an identity directly.
	// Never replace an existing association or a choice marked for review here.
	var workID string
	var season, episode int
	if c.db.QueryRowContext(ctx, `SELECT work_id,season,episode FROM metadata_priorities WHERE media_id=? AND revision=?`, m.ID, m.Revision).Scan(&workID, &season, &episode) == nil {
		w, err := c.work(ctx, workID)
		if err == nil && w.Kind == m.Kind {
			candidate := c.provider(w.Metadata.Provider)
			if candidate != nil && providerAvailable(candidate) && supportsProvider(candidate, m.Kind) {
				m.Metadata.Season = season
				m.Metadata.Episode = episode
				return c.applyProvider(ctx, candidate, c.credential(ctx, w.Metadata.Provider), m, w.Metadata.ProviderID, false)
			}
		}
	}
	title := m.Metadata.Title
	if m.Kind == "series" {
		title = m.Metadata.Series
	}
	review := func() error { m.Metadata.MatchState = "review"; return c.saveMetadata(ctx, m, *m.Metadata) }
	if title == "" || m.Metadata.EpisodeEnd > m.Metadata.Episode {
		return review()
	}
	q := MetadataQuery{Kind: m.Kind, Title: title, Year: m.Metadata.Year}
	candidates, err := p.Search(ctx, c.credential(ctx, p.Info().ID), q)
	if err != nil {
		return err
	}
	candidate, exact := exactCandidate(q, candidates)
	if !exact || (q.Year == 0 && len(candidates) != 1) {
		return review()
	}
	return c.applyProvider(ctx, p, c.credential(ctx, p.Info().ID), m, candidate.ID, false)
}
func (c *catalog) metadataQueuedMessage(ctx context.Context) string {
	var paused bool
	_ = c.db.QueryRowContext(ctx, `SELECT paused FROM metadata_queue_state WHERE id=1`).Scan(&paused)
	if paused {
		return "Analyse des fichiers terminée. L’identification des fiches est en pause ; reprenez-la dans le suivi des fiches."
	}
	return "Analyse des fichiers terminée. Les fiches vidéo en attente seront identifiées en arrière-plan si un fournisseur est configuré."
}
func metadataQueueRoutes(mux *secureMux, c *catalog, allowed func(string) bool) {
	guard := func(w http.ResponseWriter, r *http.Request) bool {
		if !allowed("films") && !allowed("series") {
			http.NotFound(w, r)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/v1/metadata/queue", func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		s, err := c.metadataStatus(r.Context(), allowed)
		if err != nil {
			problem(w, 500, "Suivi des fiches indisponible.")
			return
		}
		sendJSON(w, 200, s)
	})
	control := func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		action := r.PathValue("action")
		if r.URL.Path == "/api/v1/metadata/enrich" {
			action = "resume"
		}
		ctx := r.Context()
		var err error
		switch action {
		case "pause":
			_, err = c.db.ExecContext(ctx, `UPDATE metadata_queue_state SET paused=1 WHERE id=1`)
		case "resume":
			_, err = c.db.ExecContext(ctx, `UPDATE metadata_queue_state SET paused=0 WHERE id=1`)
		case "retry":
			// Serialize with the current identification so that a late failure cannot
			// recreate a failure that the user has just reset.
			if !c.metadataMu.TryLock() {
				problem(w, 409, "Une fiche se termine. Réessayez dans un instant.")
				return
			}
			defer c.metadataMu.Unlock()
			_, err = c.db.ExecContext(ctx, `DELETE FROM metadata_failures WHERE media_id IN (SELECT m.id FROM media m JOIN libraries l ON l.id=m.library_id WHERE (l.kind='films' AND ?) OR (l.kind='series' AND ?)); UPDATE metadata_queue_state SET retry_at=0,message='' WHERE id=1`, allowed("films"), allowed("series"))
		default:
			problem(w, 400, "Action inconnue.")
			return
		}
		if err != nil {
			problem(w, 500, "Action non enregistrée.")
			return
		}
		s, err := c.metadataStatus(ctx, allowed)
		if err != nil {
			problem(w, 500, "Suivi indisponible.")
			return
		}
		sendJSON(w, 202, s)
	}
	mux.HandleFunc("POST /api/v1/metadata/queue/{action}", control)
	mux.HandleFunc("POST /api/v1/metadata/enrich", control)
}

// Stable relative path comparison only: never infer an identity from a basename
// that could occur in several movie/series folders.
func (c *catalog) prioritizeMedia(ctx context.Context, libraryID, relative, workID string, season, episode int) error {
	var id, revision string
	err := c.db.QueryRowContext(ctx, `SELECT m.id,m.revision`+metadataQueueJoin+`WHERE m.library_id=? AND m.relative_path=? AND `+metadataPending, libraryID, relative).Scan(&id, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, `INSERT INTO metadata_priorities(media_id,revision,work_id,season,episode) VALUES(?,?,?,?,?) ON CONFLICT(media_id) DO UPDATE SET revision=excluded.revision,work_id=excluded.work_id,season=excluded.season,episode=excluded.episode`, id, revision, workID, season, episode)
	return err
}
