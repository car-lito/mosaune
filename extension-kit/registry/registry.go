// Package registry stores host-owned package intent separately from runtime
// health. It does not authenticate packages or grant execution permissions.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"mosaune/extensionkit/dependencies"
	"mosaune/extensionkit/protocol"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const MaxEntries = 128
const MaxBytes = 256 << 10

var ErrConflict = errors.New("extension registry changed; preview again")
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Entry struct {
	Descriptor protocol.Descriptor `json:"module"`
	CodeSHA256 string              `json:"codeSha256"`
	Enabled    bool                `json:"enabled"` // desired state, never a claim of runtime health
	Choices    map[string]string   `json:"choices"`
}
type Snapshot struct {
	Schema   int     `json:"schema"`
	Revision int64   `json:"revision"`
	Entries  []Entry `json:"entries"`
}

func Empty() Snapshot { return Snapshot{Schema: 1, Entries: []Entry{}} }

// Change is supplied by the host installer, after verifying package identity,
// code, signatures, permissions and compatibility. It is not an upload API.
type Change struct {
	Operation string            `json:"operation"` // install, update, restore, enable, disable, choose, remove
	ID        string            `json:"id"`
	Entry     *Entry            `json:"entry,omitempty"`
	Choices   map[string]string `json:"choices,omitempty"`
	Cascade   bool              `json:"cascade,omitempty"`
}
type Plan struct {
	BeforeRevision int64    `json:"beforeRevision"`
	After          Snapshot `json:"after"`
	Disabled       []string `json:"disabled"`
}
type DependentsError struct {
	ID         string
	Dependents []string
}

func (e *DependentsError) Error() string {
	return fmt.Sprintf("extension %s is required by %v", e.ID, e.Dependents)
}

func clone(s Snapshot) Snapshot {
	raw, _ := json.Marshal(s)
	var next Snapshot
	_ = json.Unmarshal(raw, &next)
	return next
}
func validID(id string) bool {
	return (protocol.Descriptor{ID: id, Version: "1.0.0", Capabilities: []string{"registry.identity.v1"}}).Validate() == nil
}
func shape(s Snapshot) error {
	if s.Schema != 1 || s.Revision < 0 || s.Revision == math.MaxInt64 || len(s.Entries) > MaxEntries {
		return errors.New("invalid registry schema, revision or size")
	}
	seen := map[string]bool{}
	for _, e := range s.Entries {
		if err := e.Descriptor.Validate(); err != nil {
			return err
		}
		if seen[e.Descriptor.ID] || !digest.MatchString(e.CodeSHA256) || len(e.Choices) > 16 {
			return errors.New("invalid registry entry")
		}
		seen[e.Descriptor.ID] = true
		for cap, id := range e.Choices {
			if !slices.Contains(e.Descriptor.Dependencies, cap) || !validID(id) || id == e.Descriptor.ID {
				return errors.New("invalid provider choice")
			}
		}
	}
	raw, err := json.Marshal(s)
	if err != nil || len(raw) > MaxBytes {
		return errors.New("registry too large")
	}
	return nil
}

// normalize pins resolved choices and returns providers before consumers.
// Active cycles and missing dependencies are rejected before any persistence.
func normalize(s *Snapshot) ([]string, error) {
	if err := shape(*s); err != nil {
		return nil, err
	}
	available := make([]dependencies.Candidate, 0, len(s.Entries))
	byID := map[string]int{}
	for i, e := range s.Entries {
		available = append(available, dependencies.Candidate{Descriptor: e.Descriptor, Enabled: e.Enabled})
		byID[e.Descriptor.ID] = i
	}
	for i := range s.Entries {
		e := &s.Entries[i]
		if !e.Enabled {
			continue
		}
		choices, err := dependencies.Resolve(e.Descriptor, available, e.Choices)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Descriptor.ID, err)
		}
		e.Choices = choices
	}
	visited := map[string]int{}
	order := []string{}
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] == 1 {
			return fmt.Errorf("dependency cycle at %s", id)
		}
		if visited[id] == 2 {
			return nil
		}
		visited[id] = 1
		e := s.Entries[byID[id]]
		providers := []string{}
		for _, p := range e.Choices {
			providers = append(providers, p)
		}
		sort.Strings(providers)
		for _, p := range providers {
			if err := visit(p); err != nil {
				return err
			}
		}
		visited[id] = 2
		order = append(order, id)
		return nil
	}
	ids := []string{}
	for _, e := range s.Entries {
		if e.Enabled {
			ids = append(ids, e.Descriptor.ID)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// Validate requires already-pinned active choices. A damaged/edited stored
// snapshot cannot silently select a different provider while being read.
func (s Snapshot) Validate() error {
	if err := shape(s); err != nil {
		return err
	}
	for _, e := range s.Entries {
		if e.Enabled && len(e.Choices) != len(e.Descriptor.Dependencies) {
			return errors.New("active choices must be explicit")
		}
	}
	copy := clone(s)
	_, err := normalize(&copy)
	return err
}
func dependents(s Snapshot, id string) []string {
	seen := map[string]bool{id: true}
	result := []string{}
	var visit func(string)
	visit = func(target string) {
		children := []string{}
		for _, e := range s.Entries {
			if !e.Enabled || seen[e.Descriptor.ID] {
				continue
			}
			for _, p := range e.Choices {
				if p == target {
					children = append(children, e.Descriptor.ID)
					break
				}
			}
		}
		sort.Strings(children)
		for _, child := range children {
			if seen[child] {
				continue
			}
			seen[child] = true
			visit(child)
			result = append(result, child)
		}
	}
	visit(id)
	return result
}
func newer(a, b string) bool {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(aa[i])
		y, _ := strconv.Atoi(bb[i])
		if x != y {
			return x > y
		}
	}
	return false
}

// Preview is pure: failed changes leave the input untouched. The revision is
// checked again by the store at commit. No package or media file is deleted.
func Preview(current Snapshot, change Change) (Plan, error) {
	var plan Plan
	if err := current.Validate(); err != nil {
		return plan, err
	}
	next := clone(current)
	idx := -1
	for i, e := range next.Entries {
		if e.Descriptor.ID == change.ID {
			idx = i
			break
		}
	}
	if !validID(change.ID) {
		return plan, errors.New("invalid module ID")
	}
	plan.BeforeRevision = current.Revision
	plan.Disabled = []string{}
	switch change.Operation {
	case "install", "update", "restore":
		if change.Entry == nil || change.Entry.Descriptor.ID != change.ID || change.Entry.Enabled || len(change.Entry.Choices) != 0 {
			return plan, errors.New("verified inactive entry required")
		}
		entry := clone(Snapshot{Entries: []Entry{*change.Entry}}).Entries[0]
		if err := entry.Descriptor.Validate(); err != nil {
			return plan, err
		}
		if change.Operation == "install" {
			if idx != -1 {
				return plan, errors.New("module already installed")
			}
			next.Entries = append(next.Entries, entry)
		} else {
			if idx == -1 || change.Operation == "update" && !newer(entry.Descriptor.Version, next.Entries[idx].Descriptor.Version) {
				return plan, errors.New("newer installed version required")
			}
			old := next.Entries[idx]
			entry.Enabled = old.Enabled
			entry.Choices = map[string]string{}
			for cap, id := range old.Choices {
				if slices.Contains(entry.Descriptor.Dependencies, cap) {
					entry.Choices[cap] = id
				}
			}
			next.Entries[idx] = entry
		}
	case "enable", "choose":
		if idx == -1 {
			return plan, errors.New("module not installed")
		}
		entry := &next.Entries[idx]
		if change.Choices != nil {
			if entry.Enabled && !maps.Equal(entry.Choices, change.Choices) {
				return plan, errors.New("disable module before changing providers")
			}
			entry.Choices = maps.Clone(change.Choices)
		}
		if change.Operation == "enable" {
			entry.Enabled = true
		}
		if change.Operation == "choose" {
			// A disabled module may choose an installed disabled provider, but an
			// unknown/incompatible selection is never accepted as a new preference.
			for cap, id := range entry.Choices {
				found := false
				for _, provider := range next.Entries {
					if provider.Descriptor.ID == id && slices.Contains(provider.Descriptor.Capabilities, cap) {
						found = true
					}
				}
				if !found {
					return plan, errors.New("chosen provider missing or incompatible")
				}
			}
		}
	case "disable", "remove":
		if idx == -1 {
			return plan, errors.New("module not installed")
		}
		impact := dependents(next, change.ID)
		if len(impact) > 0 && !change.Cascade {
			return plan, &DependentsError{ID: change.ID, Dependents: impact}
		}
		plan.Disabled = append(plan.Disabled, impact...)
		if next.Entries[idx].Enabled {
			plan.Disabled = append(plan.Disabled, change.ID)
		}
		for i := range next.Entries {
			if slices.Contains(plan.Disabled, next.Entries[i].Descriptor.ID) {
				next.Entries[i].Enabled = false
			}
		}
		if change.Operation == "remove" {
			next.Entries = append(next.Entries[:idx], next.Entries[idx+1:]...)
		}
	default:
		return plan, errors.New("unknown registry operation")
	}
	if _, err := normalize(&next); err != nil {
		return Plan{}, err
	}
	next.Revision++
	if err := next.Validate(); err != nil {
		return Plan{}, err
	}
	sort.Slice(next.Entries, func(i, j int) bool { return next.Entries[i].Descriptor.ID < next.Entries[j].Descriptor.ID })
	plan.After = next
	return plan, nil
}
