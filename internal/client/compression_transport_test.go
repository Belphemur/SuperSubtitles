package client

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestCompressionTransport_Gzip(t *testing.T) {
	testData := []byte("This is test data that should be compressed with gzip")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header was set
		if r.Header.Get("Accept-Encoding") != "gzip, br, zstd" {
			t.Errorf("Expected Accept-Encoding header to be 'gzip, br, zstd', got %q", r.Header.Get("Accept-Encoding"))
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)

		gzWriter := gzip.NewWriter(w)
		_, _ = gzWriter.Write(testData)
		_ = gzWriter.Close()
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be automatically decompressed)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}

	// Content-Encoding header should be removed after decompression
	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("Expected Content-Encoding header to be removed, got %q", resp.Header.Get("Content-Encoding"))
	}
}

func TestCompressionTransport_Brotli(t *testing.T) {
	testData := []byte("This is test data that should be compressed with brotli")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header was set
		if r.Header.Get("Accept-Encoding") != "gzip, br, zstd" {
			t.Errorf("Expected Accept-Encoding header to be 'gzip, br, zstd', got %q", r.Header.Get("Accept-Encoding"))
		}

		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(http.StatusOK)

		brWriter := brotli.NewWriter(w)
		_, _ = brWriter.Write(testData)
		_ = brWriter.Close()
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be automatically decompressed)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}

	// Content-Encoding header should be removed after decompression
	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("Expected Content-Encoding header to be removed, got %q", resp.Header.Get("Content-Encoding"))
	}
}

func TestCompressionTransport_Zstd(t *testing.T) {
	testData := []byte("This is test data that should be compressed with zstd")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header was set
		if r.Header.Get("Accept-Encoding") != "gzip, br, zstd" {
			t.Errorf("Expected Accept-Encoding header to be 'gzip, br, zstd', got %q", r.Header.Get("Accept-Encoding"))
		}

		w.Header().Set("Content-Encoding", "zstd")
		w.WriteHeader(http.StatusOK)

		// zstd.NewWriter() with default options never fails
		zstdWriter, _ := zstd.NewWriter(w)
		_, _ = zstdWriter.Write(testData)
		_ = zstdWriter.Close()
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be automatically decompressed)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}

	// Content-Encoding header should be removed after decompression
	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("Expected Content-Encoding header to be removed, got %q", resp.Header.Get("Content-Encoding"))
	}
}

func TestCompressionTransport_NoCompression(t *testing.T) {
	testData := []byte("This is uncompressed test data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header was set
		if r.Header.Get("Accept-Encoding") != "gzip, br, zstd" {
			t.Errorf("Expected Accept-Encoding header to be 'gzip, br, zstd', got %q", r.Header.Get("Accept-Encoding"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be uncompressed)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}
}

func TestCompressionTransport_PreserveExistingAcceptEncoding(t *testing.T) {
	testData := []byte("Test data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom Accept-Encoding header was preserved
		if r.Header.Get("Accept-Encoding") != "custom-encoding" {
			t.Errorf("Expected Accept-Encoding header to be 'custom-encoding', got %q", r.Header.Get("Accept-Encoding"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	// Create request with custom Accept-Encoding
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept-Encoding", "custom-encoding")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}
}

func TestCompressionTransport_UnknownEncoding(t *testing.T) {
	testData := []byte("Test data with unknown encoding")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "unknown-encoding")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be returned as-is)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}

	// Content-Encoding header should NOT be removed for unknown encodings
	if resp.Header.Get("Content-Encoding") != "unknown-encoding" {
		t.Errorf("Expected Content-Encoding header to be 'unknown-encoding', got %q", resp.Header.Get("Content-Encoding"))
	}
}

func TestCompressionTransport_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a HEAD or 204 response with Content-Encoding but no body
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should not fail even though Content-Encoding is set
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestCompressionTransport_CommaListEncoding(t *testing.T) {
	testData := []byte("This is test data with multiple encodings")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a response with comma-separated Content-Encoding
		w.Header().Set("Content-Encoding", "identity, gzip")
		w.WriteHeader(http.StatusOK)

		gzWriter := gzip.NewWriter(w)
		_, _ = gzWriter.Write(testData)
		_ = gzWriter.Close()
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be automatically decompressed based on last encoding)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}
}

func TestCompressionTransport_WhitespaceEncoding(t *testing.T) {
	testData := []byte("This is test data with whitespace in encoding header")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a response with whitespace in Content-Encoding
		w.Header().Set("Content-Encoding", " gzip ")
		w.WriteHeader(http.StatusOK)

		gzWriter := gzip.NewWriter(w)
		_, _ = gzWriter.Write(testData)
		_ = gzWriter.Close()
	}))
	defer server.Close()

	// Create HTTP client with compression transport
	client := &http.Client{
		Transport: newCompressionTransport(nil),
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body (should be automatically decompressed)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !bytes.Equal(body, testData) {
		t.Errorf("Expected body %q, got %q", testData, body)
	}
}

func TestParseContentEncoding(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"simple gzip", "gzip", "gzip"},
		{"simple brotli", "br", "br"},
		{"simple zstd", "zstd", "zstd"},
		{"with leading whitespace", " gzip", "gzip"},
		{"with trailing whitespace", "gzip ", "gzip"},
		{"with both whitespace", " gzip ", "gzip"},
		{"comma list - identity, gzip", "identity, gzip", "gzip"},
		{"comma list - gzip, br", "gzip, br", "br"},
		{"comma list with whitespace", "identity , gzip", "gzip"},
		{"uppercase", "GZIP", "gzip"},
		{"mixed case", "GzIp", "gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseContentEncoding(tt.header)
			if result != tt.expected {
				t.Errorf("parseContentEncoding(%q) = %q, expected %q", tt.header, result, tt.expected)
			}
		})
	}
}

