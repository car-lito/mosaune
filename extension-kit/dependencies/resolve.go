// Package dependencies resolves versioned capabilities without knowing media
// types, guest runtimes, storage, or installation sources. It grants no access.
package dependencies

import (
	"fmt"
	"mosaune/extensionkit/protocol"
	"slices"
	"sort"
)

type Candidate struct {
	Descriptor protocol.Descriptor
	Enabled    bool
}

// Issue contains stable codes for host UI translations. Candidates are sorted
// IDs, never an implicit priority based on installation order.
type Issue struct {
	Code       string   `json:"code"`
	Capability string   `json:"capability"`
	Candidates []string `json:"candidates,omitempty"`
}

func (i *Issue) Error() string { return fmt.Sprintf("dependency %s: %s", i.Capability, i.Code) }

// Resolve requires dependencies to be active already. Choices are host-owned;
// a guest cannot pick another module to gain permissions. Multiple compatible
// providers require an explicit choice. No provider is installed or enabled.
func Resolve(owner protocol.Descriptor, available []Candidate, choices map[string]string) (map[string]string, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, c := range available {
		if err := c.Descriptor.Validate(); err != nil {
			return nil, err
		}
		if ids[c.Descriptor.ID] {
			return nil, fmt.Errorf("duplicate candidate %s", c.Descriptor.ID)
		}
		ids[c.Descriptor.ID] = true
	}
	keys := make([]string, 0, len(choices))
	for key := range choices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !slices.Contains(owner.Dependencies, key) || choices[key] == "" {
			return nil, &Issue{Code: "invalid_choice", Capability: key}
		}
	}
	resolved := map[string]string{}
	for _, capability := range owner.Dependencies {
		all, active := []string{}, []string{}
		for _, c := range available {
			if c.Descriptor.ID == owner.ID || !slices.Contains(c.Descriptor.Capabilities, capability) {
				continue
			}
			all = append(all, c.Descriptor.ID)
			if c.Enabled {
				active = append(active, c.Descriptor.ID)
			}
		}
		sort.Strings(all)
		sort.Strings(active)
		if selected, ok := choices[capability]; ok {
			if !slices.Contains(all, selected) {
				return nil, &Issue{Code: "invalid_choice", Capability: capability, Candidates: all}
			}
			if !slices.Contains(active, selected) {
				return nil, &Issue{Code: "disabled", Capability: capability, Candidates: []string{selected}}
			}
			resolved[capability] = selected
			continue
		}
		switch len(active) {
		case 0:
			code := "missing"
			if len(all) > 0 {
				code = "disabled"
			}
			return nil, &Issue{Code: code, Capability: capability, Candidates: all}
		case 1:
			resolved[capability] = active[0]
		default:
			return nil, &Issue{Code: "ambiguous", Capability: capability, Candidates: active}
		}
	}
	return resolved, nil
}
