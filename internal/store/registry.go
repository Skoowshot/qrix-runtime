package store

import (
	"fmt"
	"sync"
)

// Registry maps ID to Store and Name to ID.
type Registry struct {
	nextID   uint64
	byID     map[ID]*Store
	nameToID map[Name]ID
	mu       sync.RWMutex
}

// NewRegistry creates a Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]*Store),
		nameToID: make(map[Name]ID),
	}
}

// Put maps the given Store by a generated ID, and maps the given Name to generated ID.
func (r *Registry) Put(name Name, store *Store) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.nameToID[name]; ok {
		return 0, fmt.Errorf("%w: %v", ErrNameAlreadyExists, name)
	}

	r.nextID++
	id := ID(r.nextID)
	r.byID[id] = store
	r.nameToID[name] = id
	return id, nil
}

// Get returns the Store by ID.
func (r *Registry) Get(id ID) (*Store, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	store, ok := r.byID[id]
	return store, ok
}

// GetID returns the ID by Name.
func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	store, ok := r.nameToID[name]
	return store, ok
}
