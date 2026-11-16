package plugin

import "errors"

var (
	ErrNameAlreadyExists = errors.New("name already exists")
	ErrNotFound          = errors.New("plugin not found")
	ErrSlotAlreadyExists = errors.New("slot already exists")
)
