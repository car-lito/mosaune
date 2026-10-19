package configuration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"mosaune/extensionkit/protocol"
	"reflect"
)

var ErrConflict = errors.New("Les réglages ont changé. Actualisez la fiche avant de réessayer.")
var ErrIncompatible = errors.New("Cette version nécessite une migration explicite des réglages existants.")
var ErrStored = errors.New("Réglages enregistrés invalides.")

const maxStoredBytes = 256 << 10

type Connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// SQLStore is host-only. Never pass it, an owner supplied by a guest, or Secret
// to a module. An authorized host adapter binds each invocation to its owner.
// Secrets are separated from public State, but are not encrypted at rest.
type SQLStore struct{ DB Connection }
type Bound struct {
	store SQLStore
	owner string
}
type stored struct {
	schema   Schema
	revision int64
	values   map[string]json.RawMessage
	secrets  map[string]json.RawMessage
}

func validOwner(id string) bool {
	return (protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{"configuration.owner.v1"}}).Validate() == nil
}
func (s SQLStore) Initialize(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS extension_configuration(owner TEXT PRIMARY KEY,revision INTEGER NOT NULL CHECK(revision>=0),schema_json TEXT NOT NULL,values_json TEXT NOT NULL,secrets_json TEXT NOT NULL)`)
	return err
}
func (s SQLStore) For(owner string) (Bound, error) {
	if !validOwner(owner) {
		return Bound{}, errors.New("invalid configuration owner")
	}
	return Bound{store: s, owner: owner}, nil
}
func (b Bound) load(ctx context.Context) (stored, error) {
	var v stored
	var schema, values, secrets string
	if !validOwner(b.owner) || b.store.DB == nil {
		return v, ErrStored
	}
	err := b.store.DB.QueryRowContext(ctx, `SELECT revision,schema_json,values_json,secrets_json FROM extension_configuration WHERE owner=?`, b.owner).Scan(&v.revision, &schema, &values, &secrets)
	if err != nil {
		return v, err
	}
	if v.revision < 0 || v.revision == math.MaxInt64 || len(schema)+len(values)+len(secrets) > maxStoredBytes || protocol.DecodeOne([]byte(schema), &v.schema) != nil || v.schema.Validate() != nil || json.Unmarshal([]byte(values), &v.values) != nil || json.Unmarshal([]byte(secrets), &v.secrets) != nil || v.values == nil || v.secrets == nil {
		return stored{}, ErrStored
	}
	all := map[string]json.RawMessage{}
	kinds := map[string]string{}
	for _, f := range v.schema.Fields {
		kinds[f.ID] = f.Type
	}
	for k, x := range v.values {
		if kinds[k] == "secret" {
			return stored{}, ErrStored
		}
		all[k] = x
	}
	for k, x := range v.secrets {
		if kinds[k] != "secret" {
			return stored{}, ErrStored
		}
		all[k] = x
	}
	normalized, e := v.schema.NormalizePatch(Patch{Values: all})
	if e != nil || !reflect.DeepEqual(normalized.Values, all) {
		return stored{}, ErrStored
	}
	return v, nil
}

// Register accepts only a schema authenticated by the host's installer. It is
// additive when values already exist: removing/reclassifying a populated field
// or invalidating its value requires an explicit host migration, never erasure.
// Call with the installer's transaction to commit schema and archive together.
func (s SQLStore) Register(ctx context.Context, owner string, schema Schema) error {
	b, err := s.For(owner)
	if err != nil {
		return err
	}
	if err = schema.Validate(); err != nil {
		return err
	}
	next, _ := json.Marshal(schema)
	if len(next)+4 > maxStoredBytes {
		return errors.New("configuration too large")
	}
	old, err := b.load(ctx)
	if err == sql.ErrNoRows {
		r, e := s.DB.ExecContext(ctx, `INSERT INTO extension_configuration(owner,revision,schema_json,values_json,secrets_json) SELECT ?,0,?,'{}','{}' WHERE (SELECT COUNT(*) FROM extension_configuration)<128`, owner, string(next))
		if e != nil {
			return e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			return errors.New("configuration capacity reached")
		}
		return nil
	}
	if err != nil {
		return err
	}
	previous, _ := json.Marshal(old.schema)
	if string(previous) == string(next) {
		return nil
	}
	kinds := map[string]string{}
	for _, f := range schema.Fields {
		kinds[f.ID] = f.Type
	}
	all := map[string]json.RawMessage{}
	for k, v := range old.values {
		all[k] = v
	}
	for k, v := range old.secrets {
		all[k] = v
	}
	for _, f := range old.schema.Fields {
		if _, exists := all[f.ID]; exists && kinds[f.ID] != f.Type {
			return ErrIncompatible
		}
	}
	normalized, e := schema.NormalizePatch(Patch{Values: all})
	if e != nil || !reflect.DeepEqual(normalized.Values, all) {
		return ErrIncompatible
	}
	return b.replace(ctx, old.revision, schema, old.values, old.secrets)
}
func (b Bound) replace(ctx context.Context, revision int64, schema Schema, values, secrets map[string]json.RawMessage) error {
	if revision >= math.MaxInt64-1 {
		return ErrStored
	}
	a, _ := json.Marshal(schema)
	v, _ := json.Marshal(values)
	s, _ := json.Marshal(secrets)
	if len(a)+len(v)+len(s) > maxStoredBytes {
		return errors.New("configuration too large")
	}
	r, err := b.store.DB.ExecContext(ctx, `UPDATE extension_configuration SET revision=?,schema_json=?,values_json=?,secrets_json=? WHERE owner=? AND revision=?`, revision+1, string(a), string(v), string(s), b.owner, revision)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (b Bound) Read(ctx context.Context) (State, error) {
	v, err := b.load(ctx)
	if err != nil {
		return State{}, err
	}
	state := State{Schema: v.schema, Revision: v.revision, Values: v.values, SecretsConfigured: map[string]bool{}}
	for _, f := range v.schema.Fields {
		if f.Type == "secret" {
			_, state.SecretsConfigured[f.ID] = v.secrets[f.ID]
		}
	}
	return state, nil
}
func (b Bound) Apply(ctx context.Context, patch Patch) error {
	v, err := b.load(ctx)
	if err != nil {
		return err
	}
	if patch.Revision != nil && *patch.Revision != v.revision {
		return ErrConflict
	}
	patch, err = v.schema.NormalizePatch(patch)
	if err != nil {
		return err
	}
	fields := map[string]string{}
	for _, f := range v.schema.Fields {
		fields[f.ID] = f.Type
	}
	for k, x := range patch.Values {
		if fields[k] == "secret" {
			v.secrets[k] = x
		} else {
			v.values[k] = x
		}
	}
	for _, k := range patch.ClearSecrets {
		delete(v.secrets, k)
	}
	return b.replace(ctx, v.revision, v.schema, v.values, v.secrets)
}

// Secret is for trusted host services only. Modules receive a narrow operation
// (for example a TMDB request), never the credential or this storage handle.
func (b Bound) Secret(ctx context.Context, field string) (string, error) {
	v, err := b.load(ctx)
	if err != nil {
		return "", err
	}
	for _, f := range v.schema.Fields {
		if f.ID == field && f.Type == "secret" {
			var value string
			if raw, ok := v.secrets[field]; ok {
				if json.Unmarshal(raw, &value) != nil {
					return "", ErrStored
				}
			}
			return value, nil
		}
	}
	return "", errors.New("unknown secret field")
}

// Remove deletes only this owner's private settings. The host must check that
// no remaining implementation uses them, and include it in its uninstall tx.
func (b Bound) Remove(ctx context.Context, revision int64) error {
	if _, err := b.load(ctx); err != nil {
		return err
	}
	r, err := b.store.DB.ExecContext(ctx, `DELETE FROM extension_configuration WHERE owner=? AND revision=?`, b.owner, revision)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}
