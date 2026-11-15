package plugin

import (
	"errors"
	"io"
	"slices"
)

// Slot provides an isolated execution context for a plugin and manages the lifecycle of its tracked resources.
type Slot struct {
	closers []io.Closer
}

// NewSlot creates a Slot.
func NewSlot() *Slot {
	return &Slot{
		closers: make([]io.Closer, 0),
	}
}

// TrackCloser registers an io.Closer to be disposed of when the slot closes.
// Nil closers are ignored.
func (s *Slot) TrackCloser(closer io.Closer) {
	if closer == nil {
		return
	}
	s.closers = append(s.closers, closer)
}

// Close disposes of all tracked closers in LIFO order and joins any returned errors.
func (s *Slot) Close() error {
	errs := make([]error, 0, len(s.closers))
	for _, v := range slices.Backward(s.closers) {
		if err := v.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	s.closers = nil
	return errors.Join(errs...)
}
