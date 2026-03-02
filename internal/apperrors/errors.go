package apperrors

import "fmt"

// ErrNotFound represents an error when a requested resource is not found.
type ErrNotFound struct {
	Resource string
	ID       interface{}
}

// Error implements the error interface.
func (e *ErrNotFound) Error() string {
	if e.ID != nil {
		return fmt.Sprintf("%s with ID %v not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// Is allows for error checking with errors.Is().
func (e *ErrNotFound) Is(target error) bool {
	_, ok := target.(*ErrNotFound)
	return ok
}

// NewNotFoundError creates a new ErrNotFound.
func NewNotFoundError(resource string, id interface{}) *ErrNotFound {
	return &ErrNotFound{
		Resource: resource,
		ID:       id,
	}
}

// NewSubtitlesNotFoundError creates a specific error for when subtitles are not found.
func NewSubtitlesNotFoundError(showID int) *ErrNotFound {
	return &ErrNotFound{
		Resource: "subtitles",
		ID:       showID,
	}
}

// ErrSubtitleNotFoundInZip is returned when the requested episode subtitle is not found in a ZIP archive.
type ErrSubtitleNotFoundInZip struct {
	Episode   int
	FileCount int
}

// Error implements the error interface.
func (e *ErrSubtitleNotFoundInZip) Error() string {
	return fmt.Sprintf("episode %d not found in season pack ZIP (searched %d files)", e.Episode, e.FileCount)
}

// Is allows for error checking with errors.Is().
func (e *ErrSubtitleNotFoundInZip) Is(target error) bool {
	_, ok := target.(*ErrSubtitleNotFoundInZip)
	return ok
}

// ErrSubtitleResourceNotFound is returned when the subtitle download URL returns HTTP 404.
type ErrSubtitleResourceNotFound struct {
	URL string
}

// Error implements the error interface.
func (e *ErrSubtitleResourceNotFound) Error() string {
	return fmt.Sprintf("subtitle resource not found at URL: %s", e.URL)
}

// Is allows for error checking with errors.Is().
func (e *ErrSubtitleResourceNotFound) Is(target error) bool {
	_, ok := target.(*ErrSubtitleResourceNotFound)
	return ok
}
