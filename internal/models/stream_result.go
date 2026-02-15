package models

// StreamResult holds either a value or an error from a streaming operation
type StreamResult[T any] struct {
	Value T
	Err   error
}
