package workspace

import "github.com/Skoowshot/qrix-runtime/internal/store"

// ID is a unique runtime hot-path identifier of Workspace.
type ID uint64

// Name is a unique cold-path name of Workspace.
type Name string

// Workspace is an isolated self-contained unit holding application state.
type Workspace struct {
	Stores *store.Registry
}
