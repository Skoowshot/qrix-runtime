package store

import (
	"fmt"
	"sync"
)

type Registry struct {
	nextID   uint64
	byID     map[ID]Store
	nameToID map[Name]ID
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]Store),
		nameToID: make(map[Name]ID),
	}
}

func (r *Registry) Put(name Name, store Store) (ID, error) {
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

func (r *Registry) Get(id ID) (Store, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	store, ok := r.byID[id]
	return store, ok
}

func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	store, ok := r.nameToID[name]
	return store, ok
}
