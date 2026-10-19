package runtime

import (
	"fmt"
	"maps"
	"sort"
)

type DependentsError struct {
	Module     string   `json:"module"`
	Dependents []string `json:"dependents"`
}

func (e *DependentsError) Error() string {
	return fmt.Sprintf("extension %s is required by %v", e.Module, e.Dependents)
}

// dependentsLocked returns active consumers, deepest first, in stable order.
// Enable only binds already-active modules, so cycles cannot become active.
func (e *Engine) dependentsLocked(id string) []string {
	seen := map[string]bool{id: true}
	result := []string{}
	var visit func(string)
	visit = func(target string) {
		children := []string{}
		for child, p := range e.plugins {
			if !p.enabled || seen[child] {
				continue
			}
			for _, provider := range p.bindings {
				if provider == target {
					children = append(children, child)
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

// Dependents supports impact previews. A mutation recomputes the graph while
// holding the engine lock; this snapshot is not an authorization token.
func (e *Engine) Dependents(id string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || e.plugins[id] == nil {
		return nil, ErrUnavailable
	}
	return e.dependentsLocked(id), nil
}

// Bindings returns a copy of the current active binding. The host still needs
// independent per-operation authorization before routing any call.
func (e *Engine) Bindings(id string) (map[string]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.plugins[id]
	if e.closed || e.ctx.Err() != nil || p == nil || !p.enabled {
		return nil, ErrUnavailable
	}
	return maps.Clone(p.bindings), nil
}

// DisableCascade is an explicit host operation, unlike Disable which refuses
// to break active consumers. All generations are cancelled under one lock.
// No package, configuration, catalogue or shared provider is deleted.
func (e *Engine) DisableCascade(id string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || e.plugins[id] == nil {
		return nil, ErrUnavailable
	}
	order := append(e.dependentsLocked(id), id)
	for _, target := range order {
		p := e.plugins[target]
		if p.enabled {
			p.enabled = false
			p.cancel()
		}
	}
	return order, nil
}
