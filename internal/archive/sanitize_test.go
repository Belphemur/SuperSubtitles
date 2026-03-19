package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"sort"
	"strings"
	"testing"
)

// zipEntries is a helper that returns sorted entry names and a name→content map from a ZIP.
func zipEntries(t *testing.T, zipContent []byte) ([]string, map[string][]byte) {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		t.Fatalf("failed to open result ZIP: %v", err)
	}

	names := make([]string, 0, len(zr.File))
	contents := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open ZIP entry %q: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read ZIP entry %q: %v", f.Name, err)
		}
		names = append(names, f.Name)
		contents[f.Name] = data
	}
	sort.Strings(names)
	return names, contents
}

func TestSanitizeZip_RemovesNonSubtitleFiles(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"readme.txt":      "This is a readme",
		"show.s01e01.srt": "subtitle content",
		"image.jpg":       "fake image data",
		"show.s01e02.ass": "ass subtitle",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, contents := zipEntries(t, result)

	expected := []string{"show.s01e01.srt", "show.s01e02.ass"}
	sort.Strings(expected)
	if len(names) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("entry %d: expected %q, got %q", i, expected[i], name)
		}
	}

	if string(contents["show.s01e01.srt"]) != "subtitle content" {
		t.Errorf("unexpected content for show.s01e01.srt")
	}
	if string(contents["show.s01e02.ass"]) != "ass subtitle" {
		t.Errorf("unexpected content for show.s01e02.ass")
	}
}

func TestSanitizeZip_FlattensDirectoryStructure(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"Season 1/show.s01e01.srt":          "ep1 content",
		"Season 1/Extras/show.s01e02.vtt":   "ep2 content",
		"Season 1/Deep/Sub/show.s01e03.sub": "ep3 content",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, contents := zipEntries(t, result)

	// All paths should be flattened to just the base filename
	for _, name := range names {
		if strings.Contains(name, "/") {
			t.Errorf("entry %q still contains directory path", name)
		}
	}

	expected := []string{"show.s01e01.srt", "show.s01e02.vtt", "show.s01e03.sub"}
	sort.Strings(expected)
	sort.Strings(names)
	if len(names) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("entry %d: expected %q, got %q", i, expected[i], name)
		}
	}

	if string(contents["show.s01e01.srt"]) != "ep1 content" {
		t.Errorf("unexpected content for show.s01e01.srt")
	}
}

func TestSanitizeZip_DeduplicatesAfterFlattening(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"Season 1/subtitle.srt": "first",
		"Season 2/subtitle.srt": "second",
		"Season 3/subtitle.srt": "third",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, _ := zipEntries(t, result)

	if len(names) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(names), names)
	}

	// All entries should be unique
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			t.Errorf("duplicate entry name: %q", name)
		}
		seen[name] = true
	}

	// One should be the original, others should have suffixes
	hasOriginal := false
	hasSuffix := false
	for _, name := range names {
		if name == "subtitle.srt" {
			hasOriginal = true
		}
		if strings.Contains(name, "_") {
			hasSuffix = true
		}
	}
	if !hasOriginal {
		t.Error("expected one entry to keep original name 'subtitle.srt'")
	}
	if !hasSuffix {
		t.Error("expected deduplicated entries to have numeric suffixes")
	}
}

func TestSanitizeZip_AllExtensionsKept(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"a.srt": "srt",
		"b.ass": "ass",
		"c.vtt": "vtt",
		"d.sub": "sub",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, _ := zipEntries(t, result)

	expected := []string{"a.srt", "b.ass", "c.vtt", "d.sub"}
	sort.Strings(expected)
	if len(names) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("entry %d: expected %q, got %q", i, expected[i], name)
		}
	}
}

func TestSanitizeZip_NoSubtitlesReturnsEmptyZip(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"readme.txt":  "readme",
		"picture.jpg": "image",
		"video.mp4":   "video",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, _ := zipEntries(t, result)
	if len(names) != 0 {
		t.Errorf("expected empty ZIP, got %d entries: %v", len(names), names)
	}
}

func TestSanitizeZip_SkipsDirectoryEntries(t *testing.T) {
	t.Parallel()

	// Create a ZIP with explicit directory entries
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add a directory entry
	_, err := w.Create("Season 1/")
	if err != nil {
		t.Fatal(err)
	}

	// Add a subtitle inside it
	f, err := w.Create("Season 1/show.srt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("sub content")); err != nil {
		t.Fatal(err)
	}

	w.Close()

	result, err := SanitizeZip(buf.Bytes())
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, _ := zipEntries(t, result)

	if len(names) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(names), names)
	}
	if names[0] != "show.srt" {
		t.Errorf("expected 'show.srt', got %q", names[0])
	}
}

func TestSanitizeZip_CaseInsensitiveExtensions(t *testing.T) {
	t.Parallel()

	input := createTestZip(t, map[string]string{
		"show.SRT":   "upper srt",
		"show2.Ass":  "mixed ass",
		"show3.VTT":  "upper vtt",
		"show4.Sub":  "mixed sub",
		"readme.TXT": "readme",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, _ := zipEntries(t, result)

	if len(names) != 4 {
		t.Fatalf("expected 4 subtitle entries, got %d: %v", len(names), names)
	}

	// No non-subtitle files should survive
	for _, name := range names {
		ext := strings.ToLower(name[strings.LastIndex(name, "."):])
		if !subtitleExtensions[ext] {
			t.Errorf("non-subtitle file %q should have been removed", name)
		}
	}
}

func TestSanitizeZip_InvalidZip(t *testing.T) {
	t.Parallel()

	_, err := SanitizeZip([]byte("this is not a zip file"))
	if err == nil {
		t.Error("expected error for invalid ZIP, got nil")
	}
}

func TestSanitizeZip_ZipBombDetected(t *testing.T) {
	t.Parallel()

	// Create a ZIP with a file that exceeds the individual size limit
	input := createTestZip(t, map[string]string{
		"malicious.srt": strings.Repeat("X", 25*1024*1024), // 25 MB > 20 MB limit
	})

	_, err := SanitizeZip(input)
	if err == nil {
		t.Error("expected ZIP bomb detection error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum uncompressed size") {
		t.Errorf("expected ZIP bomb error, got: %v", err)
	}
}

func TestSanitizeZip_PreservesContent(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("1\n00:00:01,000 --> 00:00:02,000\nHello\n\n", 500)
	input := createTestZip(t, map[string]string{
		"dir/show.s01e01.srt": content,
		"dir/nfo.txt":         "info",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	_, contents := zipEntries(t, result)
	if string(contents["show.s01e01.srt"]) != content {
		t.Error("subtitle content was modified during sanitization")
	}
}

func TestSanitizeZip_ConvertsContentToUTF8(t *testing.T) {
	t.Parallel()

	// ISO-8859-1 encoded content: "Café" where é = 0xE9
	iso88591Content := "1\r\n00:00:01,000 --> 00:00:02,000\r\nCaf\xe9\r\n"
	input := createTestZip(t, map[string]string{
		"show.s01e01.srt": iso88591Content,
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	_, contents := zipEntries(t, result)
	resultStr := string(contents["show.s01e01.srt"])
	if !strings.Contains(resultStr, "Café") {
		t.Errorf("expected UTF-8 converted content to contain 'Café', got %q", resultStr)
	}
}

func TestSanitizeZip_UTF8ContentPassesThrough(t *testing.T) {
	t.Parallel()

	utf8Content := "1\r\n00:00:01,000 --> 00:00:02,000\r\nCafé\r\n"
	input := createTestZip(t, map[string]string{
		"show.s01e01.srt": utf8Content,
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	_, contents := zipEntries(t, result)
	if string(contents["show.s01e01.srt"]) != utf8Content {
		t.Error("valid UTF-8 content should pass through unchanged")
	}
}

func TestSanitizeZip_RarConvertedThenSanitized(t *testing.T) {
	t.Parallel()

	// Simulate the workflow: RAR converted to ZIP, then sanitized
	// Create a ZIP that mimics what ConvertRarToZip would produce (with paths and non-subtitle files)
	input := createTestZip(t, map[string]string{
		"Renegade.S01/Renegade.S01E01.srt": "ep1",
		"Renegade.S01/Renegade.S01E02.srt": "ep2",
		"Renegade.S01/readme.nfo":          "info file",
		"Renegade.S01/cover.jpg":           "cover image",
	})

	result, err := SanitizeZip(input)
	if err != nil {
		t.Fatalf("SanitizeZip returned unexpected error: %v", err)
	}

	names, contents := zipEntries(t, result)

	// Only SRT files should remain, flattened
	if len(names) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(names), names)
	}

	for _, name := range names {
		if strings.Contains(name, "/") {
			t.Errorf("entry %q still contains directory path", name)
		}
		if !strings.HasSuffix(name, ".srt") {
			t.Errorf("non-subtitle entry %q should have been removed", name)
		}
	}

	if string(contents["Renegade.S01E01.srt"]) != "ep1" {
		t.Errorf("unexpected content for Renegade.S01E01.srt")
	}
	if string(contents["Renegade.S01E02.srt"]) != "ep2" {
		t.Errorf("unexpected content for Renegade.S01E02.srt")
	}
}

func TestDeduplicate(t *testing.T) {
	t.Parallel()

	used := make(map[string]int)

	// First use returns original name
	name1 := deduplicate("subtitle.srt", used)
	if name1 != "subtitle.srt" {
		t.Errorf("first call: expected 'subtitle.srt', got %q", name1)
	}

	// Second use returns name with suffix
	name2 := deduplicate("subtitle.srt", used)
	if name2 != "subtitle_2.srt" {
		t.Errorf("second call: expected 'subtitle_2.srt', got %q", name2)
	}

	// Third use returns name with incremented suffix
	name3 := deduplicate("subtitle.srt", used)
	if name3 != "subtitle_3.srt" {
		t.Errorf("third call: expected 'subtitle_3.srt', got %q", name3)
	}

	// Different name returns original
	name4 := deduplicate("other.srt", used)
	if name4 != "other.srt" {
		t.Errorf("different name: expected 'other.srt', got %q", name4)
	}
}

func TestDeduplicate_CaseInsensitive(t *testing.T) {
	t.Parallel()

	used := make(map[string]int)

	name1 := deduplicate("Show.SRT", used)
	if name1 != "Show.SRT" {
		t.Errorf("first call: expected 'Show.SRT', got %q", name1)
	}

	// Same name, different case - should still deduplicate
	name2 := deduplicate("show.srt", used)
	if name2 != "show_2.srt" {
		t.Errorf("second call: expected 'show_2.srt', got %q", name2)
	}
}

func TestIsSubtitleFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"SRT file", "show.srt", true},
		{"ASS file", "show.ass", true},
		{"VTT file", "show.vtt", true},
		{"SUB file", "show.sub", true},
		{"Uppercase SRT", "SHOW.SRT", true},
		{"Mixed case", "Show.Srt", true},
		{"Text file", "readme.txt", false},
		{"NFO file", "info.nfo", false},
		{"JPG file", "cover.jpg", false},
		{"No extension", "subtitle", false},
		{"Empty string", "", false},
		{"RAR file", "archive.rar", false},
		{"ZIP file", "archive.zip", false},
		{"EXE file", "setup.exe", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isSubtitleFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isSubtitleFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}
