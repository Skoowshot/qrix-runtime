package workspace

import "errors"

var (
	ErrNameAlreadyExists = errors.New("name already exists")
	ErrNotFound          = errors.New("workspace not found")
)
