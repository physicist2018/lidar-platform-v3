package domain

import "errors"

var (
	ErrInvalidPath    = errors.New("invalid storage path: bucket and path must not be empty")
	ErrObjectNotFound = errors.New("storage object not found")
)
