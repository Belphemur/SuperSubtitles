package client

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// compressionTransport wraps an http.RoundTripper to automatically handle
// response decompression for gzip, brotli, and zstd encodings
type compressionTransport struct {
	transport http.RoundTripper
}

// newCompressionTransport creates a new transport that handles automatic decompression
func newCompressionTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &compressionTransport{transport: base}
}

// RoundTrip executes a single HTTP transaction, adding Accept-Encoding header
// and automatically decompressing the response
func (t *compressionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = cloneRequest(req)

	// Add Accept-Encoding header to indicate supported compression formats
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, br, zstd")
	}

	// Execute the request
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Skip decompression if there's no body to decompress (HEAD, 204, 304 responses)
	if resp.Body == nil || resp.Body == http.NoBody {
		return resp, nil
	}

	// Decompress response body based on Content-Encoding header
	// Parse the Content-Encoding header to handle comma-separated lists and whitespace
	encoding := parseContentEncoding(resp.Header.Get("Content-Encoding"))
	if encoding == "" {
		return resp, nil
	}

	var reader io.ReadCloser
	switch encoding {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
	case "br":
		reader = io.NopCloser(brotli.NewReader(resp.Body))
	case "zstd":
		zr, err := zstd.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		reader = zr.IOReadCloser()
	default:
		// Unknown encoding, return response as-is
		return resp, nil
	}

	// Wrap the reader to close both the decompressor and original body
	resp.Body = &decompressReadCloser{
		reader:       reader,
		originalBody: resp.Body,
	}

	// Remove Content-Encoding header since we've decompressed
	resp.Header.Del("Content-Encoding")
	// Remove Content-Length as it's no longer valid after decompression
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1

	return resp, nil
}

// decompressReadCloser wraps a decompressor reader and ensures both
// the decompressor and the original body are closed
type decompressReadCloser struct {
	reader       io.ReadCloser
	originalBody io.ReadCloser
}

func (d *decompressReadCloser) Read(p []byte) (int, error) {
	return d.reader.Read(p)
}

func (d *decompressReadCloser) Close() error {
	// Close both the decompressor and the original body
	readerErr := d.reader.Close()
	bodyErr := d.originalBody.Close()

	// Return the first error if any
	if readerErr != nil {
		return readerErr
	}
	return bodyErr
}

// cloneRequest creates a shallow copy of the request
func cloneRequest(req *http.Request) *http.Request {
	// Shallow copy
	r := new(http.Request)
	*r = *req

	// Deep copy headers
	r.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		r.Header[k] = append([]string(nil), v...)
	}

	return r
}

// parseContentEncoding extracts the first/primary encoding from a Content-Encoding header.
// Handles comma-separated lists and whitespace (e.g., "gzip, br" or "gzip ").
// Returns the first encoding found, normalized to lowercase, or empty string if none.
func parseContentEncoding(header string) string {
	if header == "" {
		return ""
	}

	// Trim whitespace
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	// Handle comma-separated list - take the last encoding (applied last, needs to be removed first)
	parts := strings.Split(header, ",")
	if len(parts) > 0 {
		// Get the last encoding (outermost encoding, applied last)
		encoding := strings.TrimSpace(parts[len(parts)-1])
		return strings.ToLower(encoding)
	}

	return strings.ToLower(header)
}
