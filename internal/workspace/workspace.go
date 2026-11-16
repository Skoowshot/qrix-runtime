package workspace

import (
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/Skoowshot/qrix-runtime/internal/network"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
)

// ID is a unique runtime hot-path identifier of Workspace.
type ID uint64

// Name is a unique cold-path name of Workspace.
type Name string

// Workspace is an isolated self-contained unit holding application state.
type Workspace struct {
	slots   map[plugin.ID]*plugin.Slot
	network *network.Network

	mu sync.RWMutex
}

// NewWorkspace creates a Workspace.
func NewWorkspace() *Workspace {
	return &Workspace{
		slots: make(map[plugin.ID]*plugin.Slot),
	}
}

// PutSlot stores a plugin slot in the workspace if the ID is not already occupied.
func (w *Workspace) PutSlot(id plugin.ID, slot *plugin.Slot) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.slots[id]; exists {
		return false
	}
	w.slots[id] = slot
	return true
}

// Slot returns the plugin slot associated with the given ID.
func (w *Workspace) Slot(id plugin.ID) (*plugin.Slot, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	s, ok := w.slots[id]
	return s, ok
}

// ListSlots returns all plugin IDs currently attached to the workspace.
func (w *Workspace) ListSlots() []plugin.ID {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return slices.Collect(maps.Keys(w.slots))
}

// RemoveSlot detaches and returns the plugin slot for the given ID.
func (w *Workspace) RemoveSlot(id plugin.ID) *plugin.Slot {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := w.slots[id]
	delete(w.slots, id)
	return s
}

// Close disposes all plugin slots and joins any returned errors.
func (w *Workspace) Close() error {
	errs := make([]error, 0, len(w.slots))
	for _, v := range w.slots {
		if err := v.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	w.slots = nil
	return errors.Join(errs...)
}
