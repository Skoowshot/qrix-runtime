package meta

import (
	"context"
)

// EntryName is a key identifier for Entry.
type EntryName string

// EntryData represents Entry internal data.
type EntryData []byte

// Entry allows modifying and retrieving EntryData from Meta.
type Entry interface {
	Get(ctx context.Context) (EntryData, error)
	Set(ctx context.Context, data EntryData) error
	Delete(ctx context.Context) error
}
