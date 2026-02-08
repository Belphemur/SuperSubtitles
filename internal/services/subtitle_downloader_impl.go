package services

import (
	"SuperSubtitles/internal/config"
	"SuperSubtitles/internal/models"
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
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

// DownloadSubtitle downloads a subtitle file, with support for extracting episodes from season packs
func (d *DefaultSubtitleDownloader) DownloadSubtitle(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error) {
	logger := config.GetLogger()
	logger.Info().
		Str("url", downloadURL).
		Str("subtitleID", req.SubtitleID).
		Int("episode", req.Episode).
		Msg("Downloading subtitle")

	// Download the file
	content, contentType, err := d.downloadFile(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download subtitle: %w", err)
	}

	// If not requesting a specific episode, or if it's not a ZIP file, return as-is
	if req.Episode == 0 || !strings.Contains(contentType, "zip") {
		logger.Info().
			Str("contentType", contentType).
			Int("size", len(content)).
			Msg("Returning downloaded file as-is")

		return &models.DownloadResult{
			Filename:    generateFilename(req.SubtitleID, contentType),
			Content:     content,
			ContentType: contentType,
		}, nil
	}

	// It's a ZIP file and we need a specific episode - extract it
	logger.Info().
		Int("episode", req.Episode).
		Int("zipSize", len(content)).
		Msg("Extracting episode from season pack ZIP")

	episodeFile, err := d.extractEpisodeFromZip(content, req.Episode)
	if err != nil {
		return nil, fmt.Errorf("failed to extract episode %d from ZIP: %w", req.Episode, err)
	}

	logger.Info().
		Str("filename", episodeFile.Filename).
		Int("size", len(episodeFile.Content)).
		Msg("Successfully extracted episode from season pack")

	return episodeFile, nil
}

// generateFilename creates a filename with appropriate extension based on content type
func generateFilename(subtitleID, contentType string) string {
	ext := getExtensionFromContentType(contentType)
	return fmt.Sprintf("%s%s", subtitleID, ext)
}

// getExtensionFromContentType derives file extension from MIME type
func getExtensionFromContentType(contentType string) string {
	ctLower := strings.ToLower(contentType)

	// Check most specific patterns first to avoid false matches
	if strings.Contains(ctLower, "zip") {
		return ".zip"
	}
	if strings.Contains(ctLower, "x-subrip") {
		return ".srt"
	}
	if strings.Contains(ctLower, "x-ass") || strings.Contains(ctLower, "/ass") {
		return ".ass"
	}
	if strings.Contains(ctLower, "vtt") || strings.Contains(ctLower, "webvtt") {
		return ".vtt"
	}
	if strings.Contains(ctLower, "x-sub") {
		return ".sub"
	}
	// Fallback for generic text/srt or similar
	if strings.Contains(ctLower, "srt") {
		return ".srt"
	}

	// Default to .srt for subtitle files
	return ".srt"
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Cache ZIP files
	if strings.Contains(contentType, "zip") {
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

	// Search through ZIP files
	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Get the base filename without path
		filename := filepath.Base(file.Name)

		// Evaluate the episode pattern match once and reuse for logging and control flow
		matches := episodePattern.MatchString(filename)

		logger.Debug().
			Str("filename", filename).
			Bool("matches", matches).
			Msg("Checking file in ZIP")

		// Check if this file matches the episode pattern
		if matches {
			// Found matching episode - extract it
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s in ZIP: %w", file.Name, err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s from ZIP: %w", file.Name, err)
			}

			logger.Info().
				Str("filename", filename).
				Int("size", len(content)).
				Msg("Found and extracted episode from ZIP")

			return &models.DownloadResult{
				Filename:    filename,
				Content:     content,
				ContentType: getContentTypeFromFilename(filename),
			}, nil
		}
	}

	// Episode not found in ZIP
	return nil, fmt.Errorf("episode %d not found in season pack ZIP (searched %d files)", episode, len(zipReader.File))
}
