package workspace

import "github.com/Skoowshot/qrix-runtime/internal/store"

type Workspace struct {
	Stores *store.Registry
}
