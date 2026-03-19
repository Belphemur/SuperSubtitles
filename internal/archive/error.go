package archive

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

// ArchiveError represents failures while validating, converting, or extracting
// subtitle archive content.
type ArchiveError struct {
	Message       string
	URL           string
	Err           error
	Unrecoverable bool
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
	if e != nil && e.Unrecoverable {
		return codes.DataLoss
	}
	return codes.FailedPrecondition
}

// HTTPStatusCode returns the HTTP status code equivalent for this error.
func (e *ArchiveError) HTTPStatusCode() int {
	return http.StatusUnprocessableEntity
}

// NewError creates a new recoverable ArchiveError.
func NewError(message string, err error) *ArchiveError {
	return &ArchiveError{Message: message, Err: err}
}

// NewErrorWithURL creates a new recoverable ArchiveError that includes the source URL.
func NewErrorWithURL(message, url string, err error) *ArchiveError {
	return &ArchiveError{Message: message, URL: url, Err: err}
}

// NewUnrecoverableError creates a new unrecoverable ArchiveError.
func NewUnrecoverableError(message string, err error) *ArchiveError {
	return &ArchiveError{Message: message, Err: err, Unrecoverable: true}
}

// NewUnrecoverableErrorWithURL creates a new unrecoverable ArchiveError that includes the source URL.
func NewUnrecoverableErrorWithURL(message, url string, err error) *ArchiveError {
	return &ArchiveError{Message: message, URL: url, Err: err, Unrecoverable: true}
}
