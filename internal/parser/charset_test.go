package parser

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestNewUTF8Reader_AlreadyUTF8 tests that UTF-8 content passes through unchanged
func TestNewUTF8Reader_AlreadyUTF8(t *testing.T) {
	t.Parallel()
	input := []byte("<html><body>Hello World - UTF-8: ☺</body></html>")
	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	if !bytes.Equal(output, input) {
		t.Errorf("Expected UTF-8 content to pass through unchanged, got different content")
	}
}

// TestNewUTF8Reader_ISO88591ToUTF8 tests conversion from ISO-8859-1 to UTF-8
func TestNewUTF8Reader_ISO88591ToUTF8(t *testing.T) {
	t.Parallel()
	// HTML with ISO-8859-1 encoded special characters (é = 0xE9 in ISO-8859-1)
	// The meta tag tells the charset detector this is ISO-8859-1
	input := []byte(`<html><head><meta charset="ISO-8859-1"></head><body>Caf` + string([]byte{0xE9}) + `</body></html>`)

	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	outputStr := string(output)
	// In UTF-8, the character 'é' should be properly encoded
	if !strings.Contains(outputStr, "Café") && !strings.Contains(outputStr, "Caf\u00e9") {
		t.Errorf("Expected 'Café' in UTF-8 output, got: %s", outputStr)
	}
}

// TestNewUTF8Reader_Windows1252ToUTF8 tests conversion from Windows-1252 to UTF-8
func TestNewUTF8Reader_Windows1252ToUTF8(t *testing.T) {
	t.Parallel()
	// HTML with Windows-1252 specific character (™ = 0x99 in Windows-1252)
	// Note: 0x99 is invalid in ISO-8859-1 but valid in Windows-1252
	input := []byte(`<html><head><meta charset="windows-1252"></head><body>Test` + string([]byte{0x99}) + `</body></html>`)

	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	outputStr := string(output)
	// The trademark symbol should be present in UTF-8
	if !strings.Contains(outputStr, "™") && !strings.Contains(outputStr, "\u2122") {
		t.Errorf("Expected '™' (trademark) in UTF-8 output, got: %s", outputStr)
	}
}

// TestNewUTF8Reader_MetaHttpEquiv tests detection from meta http-equiv tag
func TestNewUTF8Reader_MetaHttpEquiv(t *testing.T) {
	t.Parallel()
	// HTML with http-equiv meta tag (older style)
	input := []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1"></head><body>Test</body></html>`)

	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	// Should successfully read and convert to UTF-8
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
}

// TestNewUTF8Reader_NoCharsetDeclaration tests heuristic detection when no charset is declared
func TestNewUTF8Reader_NoCharsetDeclaration(t *testing.T) {
	t.Parallel()
	// HTML without charset declaration - should default to UTF-8 or use heuristics
	input := []byte("<html><body>Hello World</body></html>")

	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Hello World") {
		t.Errorf("Expected 'Hello World' in output, got: %s", outputStr)
	}
}

// TestNewUTF8Reader_HungarianCharacters tests common Hungarian special characters
func TestNewUTF8Reader_HungarianCharacters(t *testing.T) {
	t.Parallel()
	// HTML with Hungarian accented characters already in UTF-8
	hungarianText := "Árvíztűrő tükörfúrógép"
	input := []byte("<html><body>" + hungarianText + "</body></html>")

	reader, err := NewUTF8Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("NewUTF8Reader failed: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from UTF-8 reader: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, hungarianText) {
		t.Errorf("Expected Hungarian text to be preserved, got: %s", outputStr)
	}
}
