package env

import (
	"fmt"
	"sync"
)

// Registry maps ID to Environment and Name to ID.
type Registry struct {
	nextID   uint64
	byID     map[ID]*Environment
	nameToID map[Name]ID
	mu       sync.RWMutex
}

// NewRegistry creates a Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]*Environment),
		nameToID: make(map[Name]ID),
	}
}

// Put maps the given Environment by a generated ID, and maps the given Name to generated ID.
func (r *Registry) Put(name Name, env *Environment) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.nameToID[name]; ok {
		return 0, fmt.Errorf("%w: %v", ErrNameAlreadyExists, name)
	}

	r.nextID++
	id := ID(r.nextID)
	r.byID[id] = env
	r.nameToID[name] = id
	return id, nil
}

// Get returns the Environment by ID.
func (r *Registry) Get(id ID) (*Environment, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	env, ok := r.byID[id]
	return env, ok
}

// GetID returns the ID by Name.
func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.nameToID[name]
	return id, ok
}
