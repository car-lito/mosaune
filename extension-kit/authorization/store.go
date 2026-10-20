// Package authorization stores host-approved service operations, separately
// from the operations requested by an extension manifest. Never expose this
// store to a guest: a declaration, signature or update is not consent.
package authorization

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"mosaune/extensionkit/protocol"
)

var ErrConflict = errors.New("Les autorisations ont changé. Rechargez les accès avant de réessayer.")
var ErrInvalid = errors.New("Autorisations invalides ou non demandées par cette extension.")
var ErrStored = errors.New("Autorisations enregistrées illisibles. Aucun accès accordé.")

type Connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type SQLStore struct{ DB Connection }
type State struct {
	Revision  int64                         `json:"revision"`
	Requested []protocol.ServiceRequirement `json:"requested"`
	Granted   []protocol.ServiceRequirement `json:"granted"`
}
type Patch struct {
	Revision *int64                        `json:"revision"`
	Granted  []protocol.ServiceRequirement `json:"granted"`
}

func canonical(owner string, services []protocol.ServiceRequirement) ([]protocol.ServiceRequirement, error) {
	d := protocol.Descriptor{ID: owner, Version: "1.0.0", Capabilities: []string{"host.authorization.v1"}, Services: services}
	if d.Validate() != nil {
		return nil, ErrInvalid
	}
	out := make([]protocol.ServiceRequirement, 0, len(services))
	for _, s := range services {
		ops := append([]string{}, s.Operations...)
		sort.Strings(ops)
		out = append(out, protocol.ServiceRequirement{Name: s.Name, Operations: ops})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func contains(services []protocol.ServiceRequirement, name, op string) bool {
	for _, s := range services {
		if s.Name == name {
			for _, o := range s.Operations {
				if o == op {
					return true
				}
			}
		}
	}
	return false
}
func (s State) Allows(name, op string) bool {
	return contains(s.Requested, name, op) && contains(s.Granted, name, op)
}
func intersection(a, b []protocol.ServiceRequirement) []protocol.ServiceRequirement {
	out := []protocol.ServiceRequirement{}
	for _, s := range a {
		ops := []string{}
		for _, op := range s.Operations {
			if contains(b, s.Name, op) {
				ops = append(ops, op)
			}
		}
		if len(ops) > 0 {
			out = append(out, protocol.ServiceRequirement{Name: s.Name, Operations: ops})
		}
	}
	return out
}
func (s SQLStore) Initialize(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS extension_authorizations(owner TEXT PRIMARY KEY,revision INTEGER NOT NULL CHECK(revision>=0),requested BLOB NOT NULL,granted BLOB NOT NULL)`)
	return err
}
func (s SQLStore) Read(ctx context.Context, owner string) (State, error) {
	if _, err := canonical(owner, nil); err != nil {
		return State{}, err
	}
	var state State
	var requested, granted []byte
	err := s.DB.QueryRowContext(ctx, `SELECT revision,requested,granted FROM extension_authorizations WHERE owner=?`, owner).Scan(&state.Revision, &requested, &granted)
	if err != nil {
		return State{}, err
	}
	if state.Revision < 0 || len(requested) > 65536 || len(granted) > 65536 || protocol.DecodeOne(requested, &state.Requested) != nil || protocol.DecodeOne(granted, &state.Granted) != nil {
		return State{}, ErrStored
	}
	r, err := canonical(owner, state.Requested)
	if err != nil {
		return State{}, ErrStored
	}
	g, err := canonical(owner, state.Granted)
	if err != nil {
		return State{}, ErrStored
	}
	if !reflect.DeepEqual(g, intersection(g, r)) {
		return State{}, ErrStored
	}
	state.Requested = r
	state.Granted = g
	return state, nil
}

// Register participates in the installer's transaction. New operations remain
// unapproved, and removed operations lose consent even if later reintroduced.
func (s SQLStore) Register(ctx context.Context, d protocol.Descriptor) error {
	if d.Validate() != nil {
		return ErrInvalid
	}
	requested, err := canonical(d.ID, d.Services)
	if err != nil {
		return err
	}
	old, err := s.Read(ctx, d.ID)
	if errors.Is(err, sql.ErrNoRows) {
		raw, _ := json.Marshal(requested)
		res, e := s.DB.ExecContext(ctx, `INSERT INTO extension_authorizations(owner,revision,requested,granted) SELECT ?,0,?,'[]' WHERE (SELECT COUNT(*) FROM extension_authorizations)<128`, d.ID, raw)
		if e != nil {
			return e
		}
		n, e := res.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			return ErrInvalid
		}
		return nil
	}
	if err != nil {
		return err
	}
	if reflect.DeepEqual(old.Requested, requested) {
		return nil
	}
	return s.write(ctx, d.ID, old.Revision, requested, intersection(old.Granted, requested))
}

// Apply replaces the complete approved set with explicit administrator input.
// The host must also reject operations for which it has no service adapter.
func (s SQLStore) Apply(ctx context.Context, owner string, patch Patch) error {
	if patch.Revision == nil || *patch.Revision < 0 {
		return ErrInvalid
	}
	granted, err := canonical(owner, patch.Granted)
	if err != nil {
		return err
	}
	old, err := s.Read(ctx, owner)
	if err != nil {
		return err
	}
	if old.Revision != *patch.Revision {
		return ErrConflict
	}
	if !reflect.DeepEqual(granted, intersection(granted, old.Requested)) {
		return ErrInvalid
	}
	return s.write(ctx, owner, old.Revision, old.Requested, granted)
}
func (s SQLStore) write(ctx context.Context, owner string, revision int64, requested, granted []protocol.ServiceRequirement) error {
	r, _ := json.Marshal(requested)
	g, _ := json.Marshal(granted)
	res, err := s.DB.ExecContext(ctx, `UPDATE extension_authorizations SET revision=revision+1,requested=?,granted=? WHERE owner=? AND revision=? AND revision<9223372036854775807`, r, g, owner, revision)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}
