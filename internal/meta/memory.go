package meta

import (
	"context"
	"slices"
	"sync"
)

// Memory is an implementation of Meta, storing entries in memory.
type Memory struct {
	entries map[EntryName]EntryData
	mu      sync.Mutex
}

// NewMemory returns a new instance of Memory
func NewMemory() *Memory {
	return &Memory{
		entries: make(map[EntryName]EntryData),
	}
}

// Entry returns the entry from memory.
func (m *Memory) Entry(name EntryName) Entry {
	return memoryEntry{memoryMeta: m, name: name}
}

// memoryEntry is a pointer to a specific entry inside Memory.
type memoryEntry struct {
	memoryMeta *Memory
	name       EntryName
}

// Get returns the current entry data from Memory.
func (m memoryEntry) Get(ctx context.Context) (EntryData, error) {
	m.memoryMeta.mu.Lock()
	defer m.memoryMeta.mu.Unlock()

	v, ok := m.memoryMeta.entries[m.name]
	if !ok {
		return nil, ErrNotFound
	}
	return slices.Clone(v), nil
}

// Set sets the current entry data in Memory.
func (m memoryEntry) Set(ctx context.Context, data EntryData) error {
	m.memoryMeta.mu.Lock()
	defer m.memoryMeta.mu.Unlock()

	m.memoryMeta.entries[m.name] = slices.Clone(data)
	return nil
}

// Delete removes the current entry data from Memory.
func (m memoryEntry) Delete(ctx context.Context) error {
	m.memoryMeta.mu.Lock()
	defer m.memoryMeta.mu.Unlock()

	delete(m.memoryMeta.entries, m.name)
	return nil
}
