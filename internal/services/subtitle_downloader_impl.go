package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/Belphemur/SuperSubtitles/v2/internal/archive"
	"github.com/Belphemur/SuperSubtitles/v2/internal/cache"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"

	"github.com/rs/zerolog"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

const (
	cacheKeyNormalizedArchivePrefix = "normalized:"
	cacheKeyEpisodeArchivePrefix    = "episode:"

	// Maximum download size to prevent OOM before archive processing (150 MB)
	maxDownloadSize = 150 * 1024 * 1024
)

// DefaultSubtitleDownloader implements SubtitleDownloader with caching
type DefaultSubtitleDownloader struct {
	httpClient   *http.Client
	archiveCache cache.Cache
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
		content, contentType, err := d.downloadSubtitleContent(ctx, downloadURL)
		if err != nil {
			metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
			return nil, fmt.Errorf("failed to download subtitle %s: %w", downloadURL, err)
		}

		logger.Info().
			Str("contentType", contentType).
			Int("size", len(content)).
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

	content, _, err := d.downloadArchiveForEpisode(ctx, downloadURL)
	if err != nil {
		metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to download subtitle %s: %w", downloadURL, err)
	}

	// downloadArchiveForEpisode guarantees ZIP content (RAR is converted, unknown format errors).
	logger.Info().
		Int("episode", *episode).
		Int("zipSize", len(content)).
		Msg("Extracting episode from season pack ZIP")

	episodeFile, err := d.extractEpisodeFromZip(content, *episode)
	if err != nil {
		metrics.SubtitleDownloadsTotal.WithLabelValues("error").Inc()
		return nil, wrapArchiveError(fmt.Sprintf("failed to extract episode %d from archive", *episode), downloadURL, err)
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
		if before, _, ok := strings.Cut(contentType, ";"); ok {
			mediaType = strings.TrimSpace(before)
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

func normalizedArchiveCacheKey(url string) string {
	return cacheKeyNormalizedArchivePrefix + url
}

func episodeArchiveCacheKey(url string) string {
	return cacheKeyEpisodeArchivePrefix + url
}

func wrapArchiveError(message, url string, err error) error {
	if err == nil {
		return nil
	}
	var episodeErr *archive.ErrEpisodeNotFound
	if errors.As(err, &episodeErr) {
		return &apperrors.ErrSubtitleNotFoundInArchive{Episode: episodeErr.Episode, FileCount: episodeErr.FileCount}
	}
	if errors.Is(err, &apperrors.ErrSubtitleNotFoundInArchive{}) {
		return err
	}
	if errors.Is(err, &apperrors.ArchiveError{}) {
		return err
	}
	return apperrors.NewArchiveErrorWithURL(message, url, err)
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

// downloadSubtitleContent downloads a subtitle resource and returns its content.
// The response may be a plain text subtitle (e.g. SRT), a ZIP archive, or a RAR archive.
// ZIP files are returned as-is, RAR files are normalized to ZIP, and text files are
// returned with their original content type. Only archives are cached.
func (d *DefaultSubtitleDownloader) downloadSubtitleContent(ctx context.Context, url string) ([]byte, string, error) {
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

	archiveFormat := archive.DetectFormat(content, contentType)
	switch archiveFormat {
	case archive.FormatZIP:
		if archive.IsZipFile(content) {
			d.archiveCache.Set(cacheKey, content)
			logger.Debug().
				Str("url", url).
				Int("size", len(content)).
				Msg("Cached ZIP download archive")
		}
		return content, "application/zip", nil
	case archive.FormatRAR:
		normalized, err := archive.ConvertRarToZip(content)
		if err != nil {
			return nil, "", apperrors.NewArchiveError("failed to normalize RAR archive to ZIP", err)
		}

		d.archiveCache.Set(cacheKey, normalized)
		logger.Info().
			Str("url", url).
			Int("rarSize", len(content)).
			Int("zipSize", len(normalized)).
			Msg("Normalized RAR archive to ZIP for download and cached it")
		return normalized, "application/zip", nil
	default:
		return content, archive.NormalizeContentType(contentType, archiveFormat), nil
	}
}

// downloadArchiveForEpisode downloads and returns a ZIP archive suitable for episode extraction.
// RAR archives are automatically converted to ZIP before caching.
func (d *DefaultSubtitleDownloader) downloadArchiveForEpisode(ctx context.Context, url string) ([]byte, string, error) {
	logger := config.GetLogger()

	cacheKey := episodeArchiveCacheKey(url)
	if cached, found := d.archiveCache.Get(cacheKey); found {
		logger.Debug().
			Str("url", url).
			Msg("Retrieved episode archive from cache")
		return cached, "application/zip", nil
	}

	content, contentType, err := d.downloadFile(ctx, url)
	if err != nil {
		return nil, "", err
	}

	archiveFormat := archive.DetectFormat(content, contentType)
	switch archiveFormat {
	case archive.FormatZIP:
		if archive.IsZipFile(content) {
			d.archiveCache.Set(cacheKey, content)
			logger.Debug().
				Str("url", url).
				Int("size", len(content)).
				Msg("Cached ZIP episode archive")
		}
		return content, "application/zip", nil
	case archive.FormatRAR:
		normalized, err := archive.ConvertRarToZip(content)
		if err != nil {
			return nil, "", apperrors.NewArchiveError("failed to convert RAR archive to ZIP for episode extraction", err)
		}
		d.archiveCache.Set(cacheKey, normalized)
		logger.Info().
			Str("url", url).
			Int("rarSize", len(content)).
			Int("zipSize", len(normalized)).
			Msg("Converted RAR to ZIP for episode extraction and cached it")
		return normalized, "application/zip", nil
	default:
		return nil, "", &apperrors.ArchiveError{
			Message: fmt.Sprintf("unsupported archive format for episode extraction (content-type: %s)", contentType),
		}
	}
}

// extractEpisodeFromZip extracts a specific episode's subtitle from a season pack ZIP.
func (d *DefaultSubtitleDownloader) extractEpisodeFromZip(zipContent []byte, episode int) (*models.DownloadResult, error) {
	logger := config.GetLogger()

	episodeFile, err := archive.ExtractEpisodeFromZip(zipContent, episode, logger)
	if err != nil {
		return nil, err
	}

	contentType := getContentTypeFromFilename(episodeFile.Filename)
	content := episodeFile.Content
	if isTextSubtitleContentType(contentType) {
		content = convertToUTF8(content)
	}

	return &models.DownloadResult{
		Filename:    episodeFile.Filename,
		Content:     content,
		ContentType: contentType,
	}, nil
}
