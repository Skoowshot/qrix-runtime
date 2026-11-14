package meta

import (
	"fmt"
	"sync"
)

type Registry struct {
	nextID   uint64
	byID     map[ID]Meta
	nameToID map[Name]ID
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		byID:     make(map[ID]Meta),
		nameToID: make(map[Name]ID),
	}
}

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

func (r *Registry) Get(id ID) (Meta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, ok := r.byID[id]
	return meta, ok
}

func (r *Registry) GetID(name Name) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, ok := r.nameToID[name]
	return meta, ok
}
