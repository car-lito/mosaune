package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mosaune/extensionkit/protocol"
	ext "mosaune/extensionkit/runtime"
)

// Artifact must come from a host source that verifies the package, signer,
// compatibility and permissions independently of this registry. A stored
// checksum is an identity check, never proof that code is trusted.
type Artifact struct {
	Descriptor protocol.Descriptor
	Code       []byte
}
type Source func(context.Context, Entry) (Artifact, error)
type Status struct {
	ID       string `json:"id"`
	Wanted   bool   `json:"wanted"`
	State    string `json:"state"` // active, disabled, suspended, blocked, failed
	Code     string `json:"code,omitempty"`
	Provider string `json:"provider,omitempty"`
}
type Report struct {
	Revision int64    `json:"revision"`
	Modules  []Status `json:"modules"`
}

// Restore creates a fresh engine exclusively owned by the caller. Providers
// start before their consumers; a failure blocks only its consumers. Desired
// choices are never rewritten. Disabled modules are not compiled, saving RAM.
// It is not an in-place update API: the installer must stop/dispose the old
// engine before creating a replacement. Safe mode creates no engine at all.
func Restore(ctx context.Context, snapshot Snapshot, limits ext.Limits, source Source, safeMode bool) (*ext.Engine, Report, error) {
	report := Report{Revision: snapshot.Revision, Modules: []Status{}}
	if err := snapshot.Validate(); err != nil {
		return nil, report, err
	}
	copy := clone(snapshot)
	order, err := normalize(&copy)
	if err != nil {
		return nil, report, err
	}
	byID := map[string]Entry{}
	indices := map[string]int{}
	for _, entry := range copy.Entries {
		id := entry.Descriptor.ID
		byID[id] = entry
		indices[id] = len(report.Modules)
		status := Status{ID: id, Wanted: entry.Enabled, State: "disabled"}
		if entry.Enabled {
			status.State = "blocked"
			status.Code = "not_started"
			if safeMode {
				status.State = "suspended"
				status.Code = "safe_mode"
			}
		}
		report.Modules = append(report.Modules, status)
	}
	if safeMode {
		return nil, report, nil
	}
	if source == nil {
		return nil, report, errors.New("verified source required")
	}
	engine, err := ext.New(ctx, limits)
	if err != nil {
		return nil, report, err
	}
	for _, id := range order {
		if err := ctx.Err(); err != nil {
			_ = engine.Close()
			return nil, report, err
		}
		entry := byID[id]
		status := &report.Modules[indices[id]]
		blocked := false
		// Descriptor order is stable even when the choices map is not.
		for _, cap := range entry.Descriptor.Dependencies {
			provider := entry.Choices[cap]
			if report.Modules[indices[provider]].State != "active" {
				status.Code = "dependency_unavailable"
				status.Provider = provider
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		// Source receives a copy: a loader cannot rewrite persisted intent.
		isolated := clone(Snapshot{Entries: []Entry{entry}}).Entries[0]
		artifact, loadErr := source(ctx, isolated)
		if loadErr != nil {
			status.State = "failed"
			status.Code = "artifact_unavailable"
			continue
		}
		hash := sha256.Sum256(artifact.Code)
		// JSON normalizes omitted/nil slices without accepting changed contents.
		a, _ := json.Marshal(artifact.Descriptor)
		b, _ := json.Marshal(entry.Descriptor)
		if hex.EncodeToString(hash[:]) != entry.CodeSHA256 || !bytes.Equal(a, b) {
			status.State = "failed"
			status.Code = "artifact_mismatch"
			continue
		}
		if err = engine.Load(ctx, entry.Descriptor, artifact.Code); err != nil {
			status.State = "failed"
			status.Code = "load_failed"
			continue
		}
		if err = engine.EnableWithProviders(id, entry.Choices); err != nil {
			_ = engine.Unload(id)
			status.State = "failed"
			status.Code = "activation_failed"
			continue
		}
		status.State = "active"
		status.Code = ""
	}
	if err := ctx.Err(); err != nil {
		_ = engine.Close()
		return nil, report, err
	}
	return engine, report, nil
}
