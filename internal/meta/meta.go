package meta

// ID is a unique runtime hot-path identifier of Meta.
type ID uint64

// Name is a unique cold-path name of Meta
type Name string

// Meta represents metadata
type Meta interface {
	// Entry returns the Entry by the given EntryName.
	Entry(name EntryName) Entry
}
