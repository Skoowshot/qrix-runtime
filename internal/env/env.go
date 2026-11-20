package env

import "io"

// Environment receives and sends messages.
type Environment interface {
	io.Closer
}
