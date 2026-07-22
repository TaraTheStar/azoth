// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"sort"
	"sync"

	"github.com/TaraTheStar/azoth/llm"
)

// Registry is a goroutine-safe set of tools keyed by name. Sibling workflow
// roles and spawned child agents share one registry and call ToolDefs
// concurrently per turn, so all access to the map and the memoized defs cache
// is guarded by mu.
type Registry[Ctx any] struct {
	mu    sync.RWMutex
	tools map[string]Tool[Ctx]

	// defsCache memoizes the sorted ToolDefs slice. Nil means "stale,
	// recompute on next call." Cleared by Register/Unregister. Filter/Without
	// build fresh registries, so their caches start stale naturally.
	defsCache []llm.ToolDef
}

// NewRegistry returns an empty registry.
func NewRegistry[Ctx any]() *Registry[Ctx] {
	return &Registry[Ctx]{tools: make(map[string]Tool[Ctx])}
}

// Register adds a tool, replacing any previous tool of the same name.
func (r *Registry[Ctx]) Register(t Tool[Ctx]) {
	r.mu.Lock()
	r.tools[t.Name()] = t
	r.defsCache = nil
	r.mu.Unlock()
}

// Unregister removes the named tools, if present. Used when an MCP gate goes
// away on reconnect/reload so the agent stops being offered tools it can no
// longer reach. Names absent from the registry are ignored.
func (r *Registry[Ctx]) Unregister(names ...string) {
	r.mu.Lock()
	for _, name := range names {
		delete(r.tools, name)
	}
	r.defsCache = nil
	r.mu.Unlock()
}

// Get returns a tool by name and whether it was present.
func (r *Registry[Ctx]) Get(name string) (Tool[Ctx], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools in unspecified order.
func (r *Registry[Ctx]) List() []Tool[Ctx] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool[Ctx], 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Filter returns a new registry holding only the named tools from r. Names not
// present in r are silently skipped. Used by skills' allowed-tools restriction,
// per-child tool subsets, and per-role workflow restriction.
func (r *Registry[Ctx]) Filter(names ...string) *Registry[Ctx] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	child := NewRegistry[Ctx]()
	for _, n := range names {
		if t, ok := r.tools[n]; ok {
			child.tools[n] = t
		}
	}
	return child
}

// Without returns a new registry with the named tools removed. Names not
// present in r are silently skipped. Used by declarative agents' denied-tools
// list.
func (r *Registry[Ctx]) Without(names ...string) *Registry[Ctx] {
	excluded := make(map[string]struct{}, len(names))
	for _, n := range names {
		excluded[n] = struct{}{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	child := NewRegistry[Ctx]()
	for name, t := range r.tools {
		if _, drop := excluded[name]; drop {
			continue
		}
		child.tools[name] = t
	}
	return child
}

// ToolDefs returns the registry's tools as OpenAI-compatible llm.ToolDef
// entries.
//
// Sorted by name so the serialized prompt prefix is byte-stable across turns —
// otherwise Go's randomized map iteration shuffles the tools array each call
// and busts the prompt-prefix cache, forcing a full re-prefill.
//
// Memoized: registries are effectively immutable after they finish wiring up,
// so the sort + alloc runs once. Register/Unregister invalidate the cache.
func (r *Registry[Ctx]) ToolDefs() []llm.ToolDef {
	// Fast path: warm cache under a read lock. The cached slice is replaced
	// wholesale (never mutated in place) by Register/Unregister, so returning
	// it without holding the lock is safe.
	r.mu.RLock()
	cached := r.defsCache
	r.mu.RUnlock()
	if cached != nil {
		return cached
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Re-check: another goroutine may have populated the cache between the
	// read-unlock above and the write-lock here.
	if r.defsCache != nil {
		return r.defsCache
	}
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	defs := make([]llm.ToolDef, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		defs = append(defs, llm.ToolDef{
			Type: "function",
			Function: llm.ToolFunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	r.defsCache = defs
	return defs
}
