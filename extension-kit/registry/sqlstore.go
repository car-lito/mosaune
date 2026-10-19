package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/protocol"
)

// SQLStore uses the host database and a single atomic compare-and-swap row.
// It owns neither the database connection nor the host's schema version. The
// host migration must call Initialize before first use (and handle backups).
type Connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type SQLStore struct{ DB Connection }

func (s SQLStore) Initialize(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS extension_registry_state(id INTEGER PRIMARY KEY CHECK(id=1),revision INTEGER NOT NULL CHECK(revision>=0),data TEXT NOT NULL)`); err != nil {
		return err
	}
	raw, _ := json.Marshal(Empty())
	_, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO extension_registry_state(id,revision,data) VALUES(1,0,?)`, string(raw))
	return err
}
func (s SQLStore) Load(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	var revision int64
	var raw string
	if err := s.DB.QueryRowContext(ctx, `SELECT revision,data FROM extension_registry_state WHERE id=1`).Scan(&revision, &raw); err != nil {
		return snapshot, err
	}
	if len(raw) > MaxBytes || protocol.DecodeOne([]byte(raw), &snapshot) != nil || snapshot.Revision != revision {
		return Snapshot{}, errors.New("invalid stored registry")
	}
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Commit recomputes the plan from current storage. A caller cannot submit a
// forged After snapshot, and concurrent/stale previews cannot overwrite data.
// This changes intent only; the host must reconcile its runtime afterwards.
func (s SQLStore) Commit(ctx context.Context, expected int64, change Change) (Plan, error) {
	current, err := s.Load(ctx)
	if err != nil {
		return Plan{}, err
	}
	if current.Revision != expected {
		return Plan{}, ErrConflict
	}
	plan, err := Preview(current, change)
	if err != nil {
		return Plan{}, err
	}
	raw, err := json.Marshal(plan.After)
	if err != nil {
		return Plan{}, err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE extension_registry_state SET revision=?,data=? WHERE id=1 AND revision=?`, plan.After.Revision, string(raw), expected)
	if err != nil {
		return Plan{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Plan{}, err
	}
	if count != 1 {
		return Plan{}, ErrConflict
	}
	return plan, nil
}
