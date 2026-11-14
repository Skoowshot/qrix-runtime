package meta

import (
	"fmt"
	"sync"
)

// Registry maps ID to Meta and Name to ID.
type Registry struct {
	nextID   uint64
	byID     map[ID]Meta
	nameToID map[Name]ID
	mu       sync.RWMutex
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]Meta),
		nameToID: make(map[Name]ID),
	}
}

// Put maps the given Meta by a generated ID, and maps the given Name to the generated ID.
func (r *Registry) Put(name Name, meta Meta) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.nameToID[name]; ok {
		return 0, fmt.Errorf("%w: %v", ErrNameAlreadyExists, name)
	}

	r.nextID++
	id := ID(r.nextID)
	r.byID[id] = meta
	r.nameToID[name] = id
	return id, nil
}

// Get returns the Meta by the given ID.
func (r *Registry) Get(id ID) (Meta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, ok := r.byID[id]
	return meta, ok
}

// GetID returns the ID by the given Name.
func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, ok := r.nameToID[name]
	return meta, ok
}
