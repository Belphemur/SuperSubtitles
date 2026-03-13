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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/Belphemur/SuperSubtitles/v2/internal/cache"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"

	"github.com/nwaples/rardecode/v2"
	"github.com/rs/zerolog"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// ZIP bomb detection constants
const (
	archiveFormatUnknown = ""
	archiveFormatZIP     = "zip"
	archiveFormatRAR     = "rar"

	cacheKeyNormalizedArchivePrefix = "normalized:"
	cacheKeyEpisodeArchivePrefix    = "episode:"

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

// DefaultSubtitleDownloader implements SubtitleDownloader with caching
type DefaultSubtitleDownloader struct {
	httpClient   *http.Client
	archiveCache cache.Cache
	rarToZip     func([]byte) ([]byte, error)
}

// resolveCacheConfig returns the cache size and TTL from cfg, with fallback defaults.
// If cfg is nil, both defaults are returned.
func resolveCacheConfig(cfg *config.Config) (size int, ttl time.Duration) {
	size = 2000
	ttl = 24 * time.Hour

	if cfg == nil {
		return
	}

	if cfg.Cache.Size > 0 {
		size = cfg.Cache.Size
	}
	if cfg.Cache.TTL != "" {
		if d, err := time.ParseDuration(cfg.Cache.TTL); err == nil {
			ttl = d
		} else {
			logger := config.GetLogger()
			logger.Warn().
				Str("cacheTTL", cfg.Cache.TTL).
				Dur("defaultTTL", ttl).
				Msg("Invalid cache TTL in configuration, falling back to default")
		}
	}
	return
}

// NewSubtitleDownloader creates a new subtitle downloader with a pluggable cache.
// The cache backend ("memory" or "redis") is selected via config (cache.type).
// Cache size and TTL are read from config (cache.size and cache.ttl).
// Defaults: memory backend, 2000 entries, 24-hour TTL.
func NewSubtitleDownloader(httpClient *http.Client) SubtitleDownloader {
	cfg := config.GetConfig()
	cacheSize, cacheTTL := resolveCacheConfig(cfg)

	cacheType := "memory"
	if cfg != nil && cfg.Cache.Type != "" {
		cacheType = cfg.Cache.Type
	}

	providerCfg := cache.ProviderConfig{
		Size:   cacheSize,
		TTL:    cacheTTL,
		Group:  "archive",
		Logger: &zerologCacheLogger{logger: config.GetLogger()},
	}
	if cfg != nil {
		providerCfg.RedisAddress = cfg.Cache.Redis.Address
		providerCfg.RedisPassword = cfg.Cache.Redis.Password
		providerCfg.RedisDB = cfg.Cache.Redis.DB
	}

	logger := config.GetLogger()
	activeType := cacheType
	archiveCache, err := cache.New(cacheType, providerCfg)
	if err != nil {
		logger.Warn().Err(err).
			Str("cacheType", cacheType).
			Msg("Failed to create cache, falling back to memory")
		activeType = "memory"
		archiveCache, err = cache.New("memory", providerCfg)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to create fallback memory cache")
		}
	}

	logger.Info().
		Str("cacheType", activeType).
		Int("cacheSize", cacheSize).
		Dur("cacheTTL", cacheTTL).
		Msg("Subtitle downloader cache initialized")

	return &DefaultSubtitleDownloader{
		httpClient:   httpClient,
		archiveCache: archiveCache,
		rarToZip:     convertRarToZip,
	}
}

// Close releases resources held by the downloader, such as cache connections.
func (d *DefaultSubtitleDownloader) Close() error {
	if d.archiveCache != nil {
		return d.archiveCache.Close()
	}
	return nil
}

// zerologCacheLogger adapts zerolog to the cache.Logger interface.
type zerologCacheLogger struct {
	logger zerolog.Logger
}

func (z *zerologCacheLogger) Error(msg string, err error) {
	z.logger.Error().Err(err).Msg(msg)
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

	if episode == nil {
		content, contentType, err := d.downloadArchiveForDownload(ctx, downloadURL)
		if err != nil {
			metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
			return nil, fmt.Errorf("failed to download subtitle: %w", err)
		}

		archiveFormat := detectArchiveFormat(content, contentType)
		contentType = normalizeArchiveContentType(contentType, archiveFormat)

		logger.Info().
			Str("contentType", contentType).
			Int("size", len(content)).
			Bool("isZip", archiveFormat == archiveFormatZIP).
			Bool("isRar", archiveFormat == archiveFormatRAR).
			Msg("Returning downloaded subtitle file")

		if isTextSubtitleContentType(contentType) {
			content = convertToUTF8(content)
		}

		metrics.SubtitleDownloadsTotal.WithLabelValues("success").Inc()
		return &models.DownloadResult{
			Filename:    generateFilename(subtitleID, contentType),
			Content:     content,
			ContentType: contentType,
		}, nil
	}

	content, contentType, err := d.downloadArchiveForEpisode(ctx, downloadURL)
	if err != nil {
		metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to download subtitle: %w", err)
	}

	archiveFormat := detectArchiveFormat(content, contentType)
	if archiveFormat == archiveFormatUnknown {
		logger.Info().
			Str("contentType", contentType).
			Int("size", len(content)).
			Bool("isZip", false).
			Bool("isRar", false).
			Msg("Returning downloaded file as-is")

		if isTextSubtitleContentType(contentType) {
			content = convertToUTF8(content)
		}

		metrics.SubtitleDownloadsTotal.WithLabelValues("success").Inc()
		return &models.DownloadResult{
			Filename:    generateFilename(subtitleID, contentType),
			Content:     content,
			ContentType: contentType,
		}, nil
	}

	var episodeFile *models.DownloadResult
	switch archiveFormat {
	case archiveFormatZIP:
		logger.Info().
			Int("episode", *episode).
			Int("zipSize", len(content)).
			Msg("Extracting episode from season pack ZIP")

		episodeFile, err = d.extractEpisodeFromZip(content, *episode)
		if err != nil {
			metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
			return nil, fmt.Errorf("failed to extract episode %d from ZIP: %w", *episode, err)
		}
	case archiveFormatRAR:
		logger.Info().
			Int("episode", *episode).
			Int("rarSize", len(content)).
			Msg("Extracting episode from season pack RAR")

		episodeFile, err = d.extractEpisodeFromRar(content, *episode)
		if err != nil {
			metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
			return nil, fmt.Errorf("failed to extract episode %d from RAR: %w", *episode, err)
		}
	default:
		metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("unsupported archive format for episode extraction: %s", archiveFormat)
	}

	logger.Info().
		Str("filename", episodeFile.Filename).
		Int("size", len(episodeFile.Content)).
		Msg("Successfully extracted episode from season pack")

	metrics.SubtitleDownloadsTotal.WithLabelValues("success").Inc()
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
	case "application/vnd.rar", "application/x-rar-compressed", "application/x-rar":
		return ".rar"
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

// isRarFile checks if the content is a RAR file using magic number detection.
// RAR 4 uses Rar!\x1A\x07\x00 and RAR 5 uses Rar!\x1A\x07\x01\x00.
func isRarFile(content []byte) bool {
	return (len(content) >= 7 && bytes.Equal(content[:7], []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x00})) ||
		(len(content) >= 8 && bytes.Equal(content[:8], []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x01, 0x00}))
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

// isRarContentType checks if the MIME type indicates a RAR file.
func isRarContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(contentType))
	}

	switch mediaType {
	case "application/vnd.rar", "application/x-rar-compressed", "application/x-rar":
		return true
	default:
		return false
	}
}

func detectArchiveFormat(content []byte, contentType string) string {
	switch {
	case isRarFile(content):
		return archiveFormatRAR
	case isZipFile(content):
		return archiveFormatZIP
	case isRarContentType(contentType):
		return archiveFormatRAR
	case isZipContentType(contentType):
		return archiveFormatZIP
	default:
		return archiveFormatUnknown
	}
}

func normalizeArchiveContentType(contentType, archiveFormat string) string {
	switch archiveFormat {
	case archiveFormatZIP:
		return "application/zip"
	case archiveFormatRAR:
		return "application/vnd.rar"
	default:
		return contentType
	}
}

func normalizedArchiveCacheKey(url string) string {
	return cacheKeyNormalizedArchivePrefix + url
}

func episodeArchiveCacheKey(url string) string {
	return cacheKeyEpisodeArchivePrefix + url
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
	case ".rar":
		return "application/vnd.rar"
	default:
		return "application/octet-stream"
	}
}

// isTextSubtitleContentType checks if the content type is a text-based subtitle format
// that should be converted to UTF-8
func isTextSubtitleContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	switch mediaType {
	case "application/x-subrip", "application/x-ass", "text/ass",
		"text/vtt", "text/webvtt", "application/x-sub",
		"text/plain":
		return true
	}
	return false
}

// convertToUTF8 detects the character encoding of text content and converts it to UTF-8.
// It handles BOM detection and uses heuristic charset detection.
// If the content is already valid UTF-8, this is a no-op.
func convertToUTF8(content []byte) []byte {
	if len(content) == 0 || utf8.Valid(content) {
		return content
	}

	// Try to detect encoding from the content
	// We pass a fake "text/plain" content type so charset.DetermineEncoding uses
	// the BOM and content heuristics rather than a declared charset
	encoding, _, _ := charset.DetermineEncoding(content, "text/plain")

	// Transform the content to UTF-8
	decoded, _, err := transform.Bytes(encoding.NewDecoder(), content)
	if err != nil {
		// If transformation fails, return original content
		logger := config.GetLogger()
		logger.Warn().Err(err).Msg("Failed to convert subtitle content to UTF-8, returning original")
		return content
	}

	return decoded
}

// downloadFile downloads a file from the given URL without archive normalization.
func (d *DefaultSubtitleDownloader) downloadFile(ctx context.Context, url string) ([]byte, string, error) {
	logger := config.GetLogger()

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", &apperrors.ErrSubtitleResourceNotFound{URL: url}
	}

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

	return content, contentType, nil
}

// downloadArchiveForDownload returns an archive suitable for whole-file downloads.
// ZIP files are returned as-is while RAR files are normalized to ZIP.
func (d *DefaultSubtitleDownloader) downloadArchiveForDownload(ctx context.Context, url string) ([]byte, string, error) {
	logger := config.GetLogger()

	cacheKey := normalizedArchiveCacheKey(url)
	if cached, found := d.archiveCache.Get(cacheKey); found {
		logger.Debug().
			Str("url", url).
			Msg("Retrieved normalized download archive from cache")
		return cached, "application/zip", nil
	}

	content, contentType, err := d.downloadFile(ctx, url)
	if err != nil {
		return nil, "", err
	}

	archiveFormat := detectArchiveFormat(content, contentType)
	switch archiveFormat {
	case archiveFormatZIP:
		if isZipFile(content) {
			d.archiveCache.Set(cacheKey, content)
			logger.Debug().
				Str("url", url).
				Int("size", len(content)).
				Msg("Cached ZIP download archive")
		}
		return content, "application/zip", nil
	case archiveFormatRAR:
		converter := d.rarToZip
		if converter == nil {
			converter = convertRarToZip
		}

		normalized, err := converter(content)
		if err != nil {
			return nil, "", fmt.Errorf("failed to normalize RAR archive to ZIP: %w", err)
		}

		d.archiveCache.Set(cacheKey, normalized)
		logger.Info().
			Str("url", url).
			Int("rarSize", len(content)).
			Int("zipSize", len(normalized)).
			Msg("Normalized RAR archive to ZIP for download and cached it")
		return normalized, "application/zip", nil
	default:
		return content, normalizeArchiveContentType(contentType, archiveFormat), nil
	}
}

// downloadArchiveForEpisode returns the original archive bytes suitable for episode extraction.
// Signature-validated ZIP and RAR archives are cached separately from download normalization.
func (d *DefaultSubtitleDownloader) downloadArchiveForEpisode(ctx context.Context, url string) ([]byte, string, error) {
	logger := config.GetLogger()

	cacheKey := episodeArchiveCacheKey(url)
	if cached, found := d.archiveCache.Get(cacheKey); found {
		archiveFormat := detectArchiveFormat(cached, "")
		logger.Debug().
			Str("url", url).
			Str("archiveFormat", archiveFormat).
			Msg("Retrieved episode archive from cache")
		return cached, normalizeArchiveContentType("application/octet-stream", archiveFormat), nil
	}

	content, contentType, err := d.downloadFile(ctx, url)
	if err != nil {
		return nil, "", err
	}

	archiveFormat := detectArchiveFormat(content, contentType)
	if isZipFile(content) || isRarFile(content) {
		d.archiveCache.Set(cacheKey, content)
		logger.Debug().
			Str("url", url).
			Str("archiveFormat", archiveFormat).
			Int("size", len(content)).
			Msg("Cached episode archive")
	}

	return content, normalizeArchiveContentType(contentType, archiveFormat), nil
}

type archiveLimitWriter struct {
	writer       io.Writer
	fileName     string
	fileWritten  int64
	totalWritten *int64
}

func (w *archiveLimitWriter) Write(p []byte) (int, error) {
	fileSize := w.fileWritten + int64(len(p))
	if fileSize > maxUncompressedFileSize {
		return 0, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
			w.fileName, fileSize, maxUncompressedFileSize)
	}

	totalSize := *w.totalWritten + int64(len(p))
	if totalSize > maxTotalUncompressedSize {
		return 0, fmt.Errorf("RAR archive total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
			totalSize, maxTotalUncompressedSize)
	}

	n, err := w.writer.Write(p)
	w.fileWritten += int64(n)
	*w.totalWritten += int64(n)
	return n, err
}

func convertRarToZip(rarContent []byte) ([]byte, error) {
	rarReader, err := rardecode.NewReader(
		bytes.NewReader(rarContent),
		rardecode.MaxDictionarySize(maxTotalUncompressedSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open RAR archive: %w", err)
	}

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	var totalWritten int64

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read RAR entry: %w", err)
		}
		if header.IsDir {
			continue
		}

		entryName := strings.ToValidUTF8(header.Name, "�")

		// Sanitize the entry name to prevent Zip-Slip path traversal attacks.
		// RAR archives can store paths with backslashes or absolute paths.
		entryName = strings.ReplaceAll(entryName, "\\", "/")
		entryName = path.Clean(entryName)
		entryName = strings.TrimLeft(entryName, "/")
		for _, component := range strings.Split(entryName, "/") {
			if component == ".." {
				return nil, fmt.Errorf("RAR archive contains path traversal in entry name: %q", header.Name)
			}
		}
		if entryName == "" || entryName == "." {
			entryName = "subtitle"
		}

		if header.UnPackedSize > maxUncompressedFileSize {
			return nil, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				entryName, header.UnPackedSize, maxUncompressedFileSize)
		}

		entryWriter, err := zipWriter.Create(entryName)
		if err != nil {
			return nil, fmt.Errorf("failed to create ZIP entry %s: %w", entryName, err)
		}

		limitWriter := &archiveLimitWriter{
			writer:       entryWriter,
			fileName:     entryName,
			totalWritten: &totalWritten,
		}

		if _, err := io.Copy(limitWriter, rarReader); err != nil {
			return nil, fmt.Errorf("failed to copy RAR entry %s: %w", entryName, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize ZIP archive: %w", err)
	}

	return zipBuffer.Bytes(), nil
}

// extractEpisodeFromRar extracts a specific episode's subtitle directly from a RAR season pack.
func (d *DefaultSubtitleDownloader) extractEpisodeFromRar(rarContent []byte, episode int) (*models.DownloadResult, error) {
	logger := config.GetLogger()

	rarReader, err := rardecode.NewReader(
		bytes.NewReader(rarContent),
		rardecode.MaxDictionarySize(maxTotalUncompressedSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open RAR archive: %w", err)
	}

	episodePattern := regexp.MustCompile(fmt.Sprintf(`(?i)(?:s\d+e%02d(?:\D|$)|e%02d(?:\D|$)|\d+x%02d(?:\D|$))`, episode, episode, episode))

	type matchedFile struct {
		filename string
		fullPath string
		priority int
	}

	subtitleExtensions := map[string]int{
		".srt": 0,
		".ass": 1,
		".vtt": 2,
		".sub": 3,
	}

	var matches []matchedFile
	fileCount := 0
	var totalUncompressed int64

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read RAR entry: %w", err)
		}
		if header.IsDir {
			continue
		}

		fileCount++
		entryName := strings.ToValidUTF8(header.Name, "�")
		// Normalize path separators before extracting the base name so that RAR
		// archives storing Windows-style backslash paths work correctly on all platforms.
		normalizedName := strings.ReplaceAll(header.Name, "\\", "/")
		filename := strings.ToValidUTF8(path.Base(normalizedName), "�")

		if header.UnPackedSize > maxUncompressedFileSize {
			return nil, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				entryName, header.UnPackedSize, maxUncompressedFileSize)
		}

		totalUncompressed += header.UnPackedSize
		if totalUncompressed > maxTotalUncompressedSize {
			return nil, fmt.Errorf("RAR archive total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
				totalUncompressed, maxTotalUncompressedSize)
		}

		matchesFilename := episodePattern.MatchString(filename)
		matchesPath := episodePattern.MatchString(entryName)
		matchesEpisode := matchesFilename || matchesPath

		logger.Debug().
			Str("filename", filename).
			Str("fullPath", entryName).
			Bool("matches", matchesEpisode).
			Msg("Checking file in RAR")

		if !matchesEpisode {
			continue
		}

		ext := strings.ToLower(filepath.Ext(filename))
		priority, isSubtitle := subtitleExtensions[ext]
		if !isSubtitle {
			priority = 4
		}

		matches = append(matches, matchedFile{
			filename: filename,
			fullPath: entryName,
			priority: priority,
		})
	}

	if len(matches) == 0 {
		return nil, &apperrors.ErrSubtitleNotFoundInArchive{Episode: episode, FileCount: fileCount}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].priority != matches[j].priority {
			return matches[i].priority < matches[j].priority
		}
		return matches[i].filename < matches[j].filename
	})

	bestMatch := matches[0]
	logger.Info().
		Str("filename", bestMatch.filename).
		Int("priority", bestMatch.priority).
		Int("totalMatches", len(matches)).
		Msg("Selected best matching subtitle from RAR")

	rarReader, err = rardecode.NewReader(
		bytes.NewReader(rarContent),
		rardecode.MaxDictionarySize(maxTotalUncompressedSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen RAR archive: %w", err)
	}

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read RAR entry: %w", err)
		}
		if header.IsDir {
			continue
		}

		entryName := strings.ToValidUTF8(header.Name, "�")
		if entryName != bestMatch.fullPath {
			continue
		}

		content, err := io.ReadAll(io.LimitReader(rarReader, int64(maxUncompressedFileSize+1)))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s from RAR: %w", entryName, err)
		}
		if len(content) > maxUncompressedFileSize {
			return nil, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				entryName, len(content), maxUncompressedFileSize)
		}

		contentType := getContentTypeFromFilename(bestMatch.filename)
		if isTextSubtitleContentType(contentType) {
			content = convertToUTF8(content)
		}

		return &models.DownloadResult{
			Filename:    bestMatch.filename,
			Content:     content,
			ContentType: contentType,
		}, nil
	}

	return nil, fmt.Errorf("failed to locate selected RAR entry %s during extraction", bestMatch.fullPath)
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
		// ZIP filenames may not be valid UTF-8 (e.g., CP437, local encoding), so sanitize them
		filename := strings.ToValidUTF8(filepath.Base(file.Name), "�")
		fullPath := strings.ToValidUTF8(file.Name, "�")

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
		return nil, &apperrors.ErrSubtitleNotFoundInArchive{Episode: episode, FileCount: len(zipReader.File)}
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

	contentType := getContentTypeFromFilename(bestMatch.filename)

	// Convert text-based subtitle files to UTF-8
	if isTextSubtitleContentType(contentType) {
		content = convertToUTF8(content)
	}

	return &models.DownloadResult{
		Filename:    bestMatch.filename,
		Content:     content,
		ContentType: contentType,
	}, nil
}
