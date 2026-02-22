package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

// ZIP bomb detection constants
const (
	// Maximum compression ratio (uncompressed/compressed)
	// Note: Highly repetitive content (like repeated chars) can legitimately compress to 1000:1 or more
	// Real subtitle files rarely exceed 20:1, but we set a generous limit to avoid false positives
	maxCompressionRatio = 10000
	// Maximum uncompressed size for a single file (20 MB - generous for subtitle files)
	maxUncompressedFileSize = 20 * 1024 * 1024
	// Maximum total uncompressed size for all files in ZIP (100 MB - for large season packs)
	maxTotalUncompressedSize = 100 * 1024 * 1024
	// Maximum download size to prevent OOM before ZIP bomb detection runs (150 MB)
	maxDownloadSize = 150 * 1024 * 1024
)

// zipCacheEntry represents a cached ZIP file with its content
type zipCacheEntry struct {
	content  []byte
	cachedAt time.Time
}

// DefaultSubtitleDownloader implements SubtitleDownloader with caching
type DefaultSubtitleDownloader struct {
	httpClient *http.Client
	zipCache   *lru.LRU[string, *zipCacheEntry]
}

// NewSubtitleDownloader creates a new subtitle downloader with LRU cache
// Cache stores up to 100 ZIP files with 1-hour TTL
func NewSubtitleDownloader(httpClient *http.Client) SubtitleDownloader {
	return &DefaultSubtitleDownloader{
		httpClient: httpClient,
		zipCache:   lru.NewLRU[string, *zipCacheEntry](100, nil, time.Hour),
	}
}

// DownloadSubtitle downloads a subtitle file, with support for extracting episodes from season packs.
// If episode is nil, the entire file is returned without extraction.
func (d *DefaultSubtitleDownloader) DownloadSubtitle(ctx context.Context, downloadURL string, episode *int) (*models.DownloadResult, error) {
	logger := config.GetLogger()
	subtitleID := extractSubtitleID(downloadURL)
	logEvent := logger.Info().
		Str("url", downloadURL).
		Str("subtitleID", subtitleID)
	if episode != nil {
		logEvent = logEvent.Int("episode", *episode)
	}
	logEvent.Msg("Downloading subtitle")

	// Download the file
	content, contentType, err := d.downloadFile(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download subtitle: %w", err)
	}

	// Check if it's a ZIP file using both content-type and magic number
	isZip := isZipFile(content) || isZipContentType(contentType)

	// If not requesting a specific episode, or if it's not a ZIP file, return as-is
	if episode == nil || !isZip {
		logger.Info().
			Str("contentType", contentType).
			Int("size", len(content)).
			Bool("isZip", isZip).
			Msg("Returning downloaded file as-is")

		return &models.DownloadResult{
			Filename:    generateFilename(subtitleID, contentType),
			Content:     content,
			ContentType: contentType,
		}, nil
	}

	// It's a ZIP file and we need a specific episode - extract it
	logger.Info().
		Int("episode", *episode).
		Int("zipSize", len(content)).
		Msg("Extracting episode from season pack ZIP")

	episodeFile, err := d.extractEpisodeFromZip(content, *episode)
	if err != nil {
		return nil, fmt.Errorf("failed to extract episode %d from ZIP: %w", *episode, err)
	}

	logger.Info().
		Str("filename", episodeFile.Filename).
		Int("size", len(episodeFile.Content)).
		Msg("Successfully extracted episode from season pack")

	return episodeFile, nil
}

// generateFilename creates a filename with appropriate extension based on content type
func generateFilename(subtitleID, contentType string) string {
	if subtitleID == "" {
		subtitleID = "subtitle"
	}
	ext := getExtensionFromContentType(contentType)
	return fmt.Sprintf("%s%s", subtitleID, ext)
}

func extractSubtitleID(downloadURL string) string {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return ""
	}

	return parsedURL.Query().Get("felirat")
}

// getExtensionFromContentType derives file extension from MIME type
func getExtensionFromContentType(contentType string) string {
	// Parse the media type to handle parameters properly
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If parsing fails, try to extract the type before any semicolon
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		} else {
			mediaType = contentType
		}
	}
	mediaType = strings.ToLower(mediaType)

	// Check for specific MIME types (most specific first)
	switch mediaType {
	case "application/zip", "application/x-zip-compressed":
		return ".zip"
	case "application/x-subrip":
		return ".srt"
	case "application/x-ass", "text/ass":
		return ".ass"
	case "text/vtt", "text/webvtt":
		return ".vtt"
	case "application/x-sub":
		return ".sub"
	}

	// Fallback for generic patterns
	if strings.Contains(mediaType, "srt") {
		return ".srt"
	}

	// Default to .srt for subtitle files
	return ".srt"
}

// isZipFile checks if the content is a ZIP file using magic number detection
// ZIP files start with PK\x03\x04 (0x504B0304) or PK\x05\x06 (empty archive) or PK\x07\x08 (spanned archive)
func isZipFile(content []byte) bool {
	if len(content) < 4 {
		return false
	}
	// Check for ZIP magic numbers
	return (content[0] == 0x50 && content[1] == 0x4B &&
		(content[2] == 0x03 && content[3] == 0x04 || // Standard ZIP
			content[2] == 0x05 && content[3] == 0x06 || // Empty ZIP
			content[2] == 0x07 && content[3] == 0x08)) // Spanned ZIP
}

// isZipContentType checks if the MIME type indicates a ZIP file
func isZipContentType(contentType string) bool {
	// Parse the media type to handle parameters and case-insensitivity
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If parsing fails, fall back to simple check
		return strings.EqualFold(contentType, "application/zip")
	}
	// Check for known ZIP media types
	return mediaType == "application/zip" ||
		mediaType == "application/x-zip-compressed"
}

// detectZipBomb analyzes a ZIP file for characteristics of a ZIP bomb
func detectZipBomb(zipContent []byte) error {
	// Open ZIP archive for inspection
	zipReader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return fmt.Errorf("failed to open ZIP for bomb detection: %w", err)
	}

	compressedSize := int64(len(zipContent))
	var totalUncompressedSize uint64

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		uncompressedSize := file.UncompressedSize64
		totalUncompressedSize += uncompressedSize

		// Check 1: Individual file size limit
		if uncompressedSize > maxUncompressedFileSize {
			return fmt.Errorf("ZIP bomb detected: file %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				file.Name, uncompressedSize, maxUncompressedFileSize)
		}

		// Check 2: Compression ratio per file (avoid division by zero)
		if file.CompressedSize64 > 0 {
			ratio := float64(uncompressedSize) / float64(file.CompressedSize64)
			if ratio > maxCompressionRatio {
				return fmt.Errorf("ZIP bomb detected: file %s has suspicious compression ratio (%.2f > %d)",
					file.Name, ratio, maxCompressionRatio)
			}
		}
	}

	// Check 3: Total uncompressed size limit
	if totalUncompressedSize > maxTotalUncompressedSize {
		return fmt.Errorf("ZIP bomb detected: total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
			totalUncompressedSize, maxTotalUncompressedSize)
	}

	// Check 4: Overall compression ratio
	if compressedSize > 0 {
		overallRatio := float64(totalUncompressedSize) / float64(compressedSize)
		if overallRatio > maxCompressionRatio {
			return fmt.Errorf("ZIP bomb detected: overall compression ratio is suspicious (%.2f > %d)",
				overallRatio, maxCompressionRatio)
		}
	}

	return nil
}

// getContentTypeFromFilename derives MIME type from file extension
func getContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".srt":
		return "application/x-subrip"
	case ".ass":
		return "application/x-ass"
	case ".vtt":
		return "text/vtt"
	case ".sub":
		return "application/x-sub"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// downloadFile downloads a file from the given URL with caching for ZIP files
func (d *DefaultSubtitleDownloader) downloadFile(ctx context.Context, url string) ([]byte, string, error) {
	logger := config.GetLogger()

	// Check cache first
	if cached, found := d.zipCache.Get(url); found {
		logger.Debug().
			Str("url", url).
			Time("cachedAt", cached.cachedAt).
			Msg("Retrieved file from cache")
		return cached.content, "application/zip", nil
	}

	// Download from URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit reading to prevent OOM with very large files
	// Use LimitReader to cap at maxDownloadSize + 1 byte to detect oversized responses
	limitedReader := io.LimitReader(resp.Body, int64(maxDownloadSize+1))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if download exceeded size limit
	if len(content) > maxDownloadSize {
		logger.Warn().
			Str("url", url).
			Int("size", len(content)).
			Int("limit", maxDownloadSize).
			Msg("Download exceeded size limit")
		return nil, "", fmt.Errorf("download size (%d bytes) exceeds limit (%d bytes)", len(content), maxDownloadSize)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Cache ZIP files based on magic number detection (more reliable than content-type)
	if isZipFile(content) {
		d.zipCache.Add(url, &zipCacheEntry{
			content:  content,
			cachedAt: time.Now(),
		})
		logger.Debug().
			Str("url", url).
			Int("size", len(content)).
			Msg("Cached ZIP file")
	}

	return content, contentType, nil
}

// extractEpisodeFromZip extracts a specific episode's subtitle from a season pack ZIP
func (d *DefaultSubtitleDownloader) extractEpisodeFromZip(zipContent []byte, episode int) (*models.DownloadResult, error) {
	logger := config.GetLogger()

	// Detect ZIP bombs before processing
	if err := detectZipBomb(zipContent); err != nil {
		logger.Warn().Err(err).Msg("ZIP bomb detected and blocked")
		return nil, err
	}

	// Open ZIP archive
	zipReader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP archive: %w", err)
	}

	// Pattern to match episode numbers in filenames with word boundaries to prevent false positives
	// Matches: S03E01, s03e01, 3x01, E01 (but not E010 when looking for E01)
	episodePattern := regexp.MustCompile(fmt.Sprintf(`(?i)(?:s\d+e%02d(?:\D|$)|e%02d(?:\D|$)|\d+x%02d(?:\D|$))`, episode, episode, episode))

	logger.Debug().
		Int("fileCount", len(zipReader.File)).
		Int("episode", episode).
		Msg("Searching for episode in ZIP archive")

	// Collect all matching subtitle files
	type matchedFile struct {
		file     *zip.File
		filename string
		fullPath string
		priority int // Lower is better: .srt=0, .ass=1, .vtt=2, .sub=3, other=4
	}
	var matches []matchedFile

	// Known subtitle extensions in priority order
	subtitleExtensions := map[string]int{
		".srt": 0,
		".ass": 1,
		".vtt": 2,
		".sub": 3,
	}

	// Search through ZIP files
	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Check both the full path and the base filename for episode pattern
		// This handles both flat structures (episode in filename) and nested structures (episode in folder name)
		filename := filepath.Base(file.Name)
		fullPath := file.Name

		// Evaluate the episode pattern match once for both filename and full path
		matchesFilename := episodePattern.MatchString(filename)
		matchesPath := episodePattern.MatchString(fullPath)
		matchesEpisode := matchesFilename || matchesPath

		logger.Debug().
			Str("filename", filename).
			Str("fullPath", fullPath).
			Bool("matches", matchesEpisode).
			Msg("Checking file in ZIP")

		// Check if this file matches the episode pattern
		if matchesEpisode {
			// Check if it's a known subtitle file type
			ext := strings.ToLower(filepath.Ext(filename))
			priority, isSubtitle := subtitleExtensions[ext]
			if !isSubtitle {
				// Unknown extension - assign lowest priority
				priority = 4
				logger.Debug().
					Str("filename", filename).
					Str("extension", ext).
					Msg("Matched file is not a known subtitle type, assigning low priority")
			}

			matches = append(matches, matchedFile{
				file:     file,
				filename: filename,
				fullPath: fullPath,
				priority: priority,
			})
		}
	}

	// No matches found
	if len(matches) == 0 {
		return nil, fmt.Errorf("episode %d not found in season pack ZIP (searched %d files)", episode, len(zipReader.File))
	}

	// Sort matches: first by priority (prefer .srt over others), then by filename for determinism
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].priority != matches[j].priority {
			return matches[i].priority < matches[j].priority
		}
		// Same priority, sort alphabetically for determinism
		return matches[i].filename < matches[j].filename
	})

	// Use the best match
	bestMatch := matches[0]

	logger.Info().
		Str("filename", bestMatch.filename).
		Int("priority", bestMatch.priority).
		Int("totalMatches", len(matches)).
		Msg("Selected best matching subtitle from ZIP")

	// Extract the selected file
	rc, err := bestMatch.file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s in ZIP: %w", bestMatch.file.Name, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s from ZIP: %w", bestMatch.file.Name, err)
	}

	return &models.DownloadResult{
		Filename:    bestMatch.filename,
		Content:     content,
		ContentType: getContentTypeFromFilename(bestMatch.filename),
	}, nil
}
