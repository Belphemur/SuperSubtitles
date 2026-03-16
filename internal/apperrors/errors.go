package apperrors

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

// GRPCBindableError describes an application error that carries a canonical
// gRPC code and an equivalent HTTP status used by API translation layers.
type GRPCBindableError interface {
	error
	GRPCCode() codes.Code
	HTTPStatusCode() int
}

// ErrNotFound represents an error when a requested resource is not found.
type ErrNotFound struct {
	Resource string
	ID       any
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

// GRPCCode returns the gRPC status code for this error.
func (e *ErrNotFound) GRPCCode() codes.Code {
	return codes.NotFound
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ErrNotFound) HTTPStatusCode() int {
	return http.StatusNotFound
}

// NewNotFoundError creates a new ErrNotFound.
func NewNotFoundError(resource string, id any) *ErrNotFound {
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

// ErrSubtitleNotFoundInArchive is returned when the requested episode subtitle is not found in a season-pack archive.
type ErrSubtitleNotFoundInArchive struct {
	Episode   int
	FileCount int
}

// Error implements the error interface.
func (e *ErrSubtitleNotFoundInArchive) Error() string {
	return fmt.Sprintf("episode %d not found in season pack archive (searched %d files)", e.Episode, e.FileCount)
}

// Is allows for error checking with errors.Is().
func (e *ErrSubtitleNotFoundInArchive) Is(target error) bool {
	_, ok := target.(*ErrSubtitleNotFoundInArchive)
	return ok
}

// GRPCCode returns the gRPC status code for this error.
func (e *ErrSubtitleNotFoundInArchive) GRPCCode() codes.Code {
	return codes.NotFound
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ErrSubtitleNotFoundInArchive) HTTPStatusCode() int {
	return http.StatusNotFound
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

// GRPCCode returns the gRPC status code for this error.
func (e *ErrSubtitleResourceNotFound) GRPCCode() codes.Code {
	return codes.NotFound
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ErrSubtitleResourceNotFound) HTTPStatusCode() int {
	return http.StatusNotFound
}

// ArchiveError represents failures while validating, converting, or extracting
// subtitle archive content.
type ArchiveError struct {
	Message string
	URL     string
	Err     error
}

// Error implements the error interface.
func (e *ArchiveError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if e.URL != "" {
		if msg != "" {
			msg = fmt.Sprintf("%s (url: %s)", msg, e.URL)
		} else {
			msg = fmt.Sprintf("url: %s", e.URL)
		}
	}
	if e.Err == nil {
		return msg
	}
	if msg == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", msg, e.Err)
}

// Unwrap returns the wrapped cause.
func (e *ArchiveError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is allows for error checking with errors.Is().
func (e *ArchiveError) Is(target error) bool {
	_, ok := target.(*ArchiveError)
	return ok
}

// GRPCCode returns the gRPC status code for this error.
func (e *ArchiveError) GRPCCode() codes.Code {
	return codes.FailedPrecondition
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ArchiveError) HTTPStatusCode() int {
	return http.StatusUnprocessableEntity
}

// NewArchiveError creates a new ArchiveError.
func NewArchiveError(message string, err error) *ArchiveError {
	return &ArchiveError{Message: message, Err: err}
}

// NewArchiveErrorWithURL creates a new ArchiveError that includes the source URL.
func NewArchiveErrorWithURL(message, url string, err error) *ArchiveError {
	return &ArchiveError{Message: message, URL: url, Err: err}
}
