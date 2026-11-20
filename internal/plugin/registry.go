package plugin

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Registry maps ID to Plugin and Name to ID.
type Registry struct {
	nextID   uint64
	byID     map[ID]Plugin
	nameToID map[Name]ID
	mu       sync.RWMutex
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]Plugin),
		nameToID: make(map[Name]ID),
	}
}

// Put maps the given Plugin by a generated ID, and maps the given Name to the generated ID.
func (r *Registry) Put(name Name, plugin Plugin) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.nameToID[name]; ok {
		return 0, fmt.Errorf("%w: %v", ErrNameAlreadyExists, name)
	}

	r.nextID++
	id := ID(r.nextID)
	r.byID[id] = plugin
	r.nameToID[name] = id
	return id, nil
}

// Get returns the Plugin by the given ID.
func (r *Registry) Get(id ID) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.byID[id]
	return plugin, ok
}

// GetID returns the ID by the given Name.
func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.nameToID[name]
	return plugin, ok
}

func (r *Registry) Snapshot() (ids []ID, mapNames map[Name]ID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Collect(maps.Keys(r.byID)), maps.Clone(r.nameToID)
}
