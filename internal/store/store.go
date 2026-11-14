package store

import (
	"github.com/Skoowshot/qrix-runtime/internal/cas"
	"github.com/Skoowshot/qrix-runtime/internal/meta"
)

// ID is a unique runtime hot-path identifier of Store.
type ID uint64

// Name is a unique cold-path name of Store.
type Name string

// Store binds cas.Mount with meta.Registry.
type Store struct {
	Mount cas.Mount
	Meta  *meta.Registry
}
