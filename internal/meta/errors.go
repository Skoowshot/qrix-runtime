package meta

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrNameAlreadyExists = errors.New("name already exists")
)
