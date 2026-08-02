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

// ErrCircuitOpen is returned when an upstream request is rejected because the
// circuit breaker protecting calls to feliratok.eu is open (or half-open and
// out of trial permits), meaning recent requests have been failing repeatedly
// and further calls are being short-circuited to let the upstream recover.
type ErrCircuitOpen struct {
	// Endpoint identifies the upstream URL that was being called when the
	// circuit breaker rejected the request.
	Endpoint string
}

// Error implements the error interface.
func (e *ErrCircuitOpen) Error() string {
	if e.Endpoint != "" {
		return fmt.Sprintf("circuit breaker open: too many recent failures calling %s, request rejected to allow recovery", e.Endpoint)
	}
	return "circuit breaker open: too many recent failures calling the upstream service, request rejected to allow recovery"
}

// Is allows for error checking with errors.Is().
func (e *ErrCircuitOpen) Is(target error) bool {
	_, ok := target.(*ErrCircuitOpen)
	return ok
}

// GRPCCode returns the gRPC status code for this error.
func (e *ErrCircuitOpen) GRPCCode() codes.Code {
	return codes.Unavailable
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ErrCircuitOpen) HTTPStatusCode() int {
	return http.StatusServiceUnavailable
}

// NewCircuitOpenError creates a new ErrCircuitOpen for the given upstream endpoint.
func NewCircuitOpenError(endpoint string) *ErrCircuitOpen {
	return &ErrCircuitOpen{Endpoint: endpoint}
}
