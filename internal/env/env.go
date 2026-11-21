package env

import "io"

// ID is a unique runtime hot-path identifier of Environment.
type ID uint64

// Name is a unique cold-path name of Environment.
type Name string

// Environment receives and sends messages.
type Environment interface {
	io.Closer
}
