package env

import (
	"encoding/json"
	"io"
)

// ID is a unique runtime hot-path identifier of Environment.
type ID uint64

// Name is a unique cold-path name of Environment.
type Name string

// Environment receives and sends messages.
type Environment interface {
	io.Closer
}

// Metadata contains external Environment data.
type Metadata = json.RawMessage
