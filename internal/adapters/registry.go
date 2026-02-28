package adapters

import (
	"fmt"
	"sort"
	"strings"
)

// Registry resolves adapters by client name.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry returns an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: map[string]Adapter{},
	}
}

// Register adds or replaces an adapter by normalized name.
func (r *Registry) Register(adapter Adapter) {
	name := normalizeAdapterName(adapter.Name())
	r.adapters[name] = adapter
}

// Get returns an adapter by client name.
func (r *Registry) Get(client string) (Adapter, error) {
	name := normalizeAdapterName(client)
	adapter, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("adapter not registered for client %q", client)
	}
	return adapter, nil
}

// Names returns registered adapter names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeAdapterName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}
