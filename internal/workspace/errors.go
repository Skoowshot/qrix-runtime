package workspace

import "errors"

var (
	ErrNameAlreadyExists = errors.New("name already exists")
	ErrNotFound          = errors.New("not found")
)
