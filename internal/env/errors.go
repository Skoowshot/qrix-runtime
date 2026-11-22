package env

import "errors"

var (
	ErrNameAlreadyExists  = errors.New("name already exists")
	ErrTokenAlreadyExists = errors.New("token already exists")
	ErrClosed             = errors.New("closed")
)
