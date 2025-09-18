package client

import "fmt"

// ErrNotFound represents an error when a requested resource is not found
type ErrNotFound struct {
	Resource string
	ID       interface{}
}

// Error implements the error interface
func (e *ErrNotFound) Error() string {
	if e.ID != nil {
		return fmt.Sprintf("%s with ID %v not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// Is allows for error checking with errors.Is()
func (e *ErrNotFound) Is(target error) bool {
	_, ok := target.(*ErrNotFound)
	return ok
}

// NewNotFoundError creates a new ErrNotFound
func NewNotFoundError(resource string, id interface{}) *ErrNotFound {
	return &ErrNotFound{
		Resource: resource,
		ID:       id,
	}
}

// NewSubtitlesNotFoundError creates a specific error for when subtitles are not found
func NewSubtitlesNotFoundError(showID int) *ErrNotFound {
	return &ErrNotFound{
		Resource: "subtitles",
		ID:       showID,
	}
}
