package workspace

import (
	"errors"

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
}

// NewWorkspace creates a Workspace.
func NewWorkspace() *Workspace {
	return &Workspace{
		slots: make(map[plugin.ID]*plugin.Slot),
	}
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
