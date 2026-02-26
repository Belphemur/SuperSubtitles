package parser

import (
	"io"

	"golang.org/x/net/html/charset"
)

// NewUTF8Reader wraps an io.Reader with automatic character encoding detection and conversion to UTF-8.
// This ensures that HTML content from any encoding (ISO-8859-1, Windows-1252, UTF-8, etc.)
// is properly converted to UTF-8 before parsing with goquery.
//
// The charset is detected from:
// 1. HTML <meta charset="..."> or <meta http-equiv="Content-Type"> tags
// 2. XML <?xml encoding="..."> declarations
// 3. Byte order marks (BOM)
// 4. Heuristic detection if none of the above are present
//
// If the content is already UTF-8, this is a no-op wrapper with minimal overhead.
func NewUTF8Reader(body io.Reader) (io.Reader, error) {
	// charset.NewReader automatically detects encoding and converts to UTF-8
	// contentType is empty because we want it to detect from the HTML content itself
	return charset.NewReader(body, "")
}
