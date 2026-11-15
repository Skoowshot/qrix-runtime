package workspace

import (
	"fmt"
	"sync"
)

// Registry maps ID to Workspace and Name to ID.
type Registry struct {
	nextID   uint64
	byID     map[ID]*Workspace
	nameToID map[Name]ID
	mu       sync.RWMutex
}

// NewRegistry creates a Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]*Workspace),
		nameToID: make(map[Name]ID),
	}
}

// Put maps the given Workspace by a generated ID, and maps the given Name to generated ID.
func (r *Registry) Put(name Name, workspace *Workspace) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.nameToID[name]; ok {
		return 0, fmt.Errorf("%w: %v", ErrNameAlreadyExists, name)
	}

	r.nextID++
	id := ID(r.nextID)
	r.byID[id] = workspace
	r.nameToID[name] = id
	return id, nil
}

// Get returns the Workspace by ID.
func (r *Registry) Get(id ID) (*Workspace, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workspace, ok := r.byID[id]
	return workspace, ok
}

// GetID returns the ID by Name.
func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workspace, ok := r.nameToID[name]
	return workspace, ok
}
