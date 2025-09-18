package parser

import "io"

// Parser defines a generic interface for parsing HTML content
type Parser[T any] interface {
	ParseHtml(body io.Reader) ([]T, error)
}
