package parser

import "io"

// Parser defines a generic interface for parsing HTML content
type Parser[T any] interface {
	ParseHtml(body io.Reader) ([]T, error)
}

// SingleResultParser defines an interface for parsers that return a single result
type SingleResultParser[T any] interface {
	ParseHtml(body io.Reader) (T, error)
}
