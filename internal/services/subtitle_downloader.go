package services

import (
	"context"
	"fmt"

	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// ErrSubtitleNotFoundInZip is returned when the requested episode subtitle is not found in a ZIP archive.
type ErrSubtitleNotFoundInZip struct {
	Episode   int
	FileCount int
}

// Error implements the error interface.
func (e *ErrSubtitleNotFoundInZip) Error() string {
	return fmt.Sprintf("episode %d not found in season pack ZIP (searched %d files)", e.Episode, e.FileCount)
}

// Is allows for error checking with errors.Is().
func (e *ErrSubtitleNotFoundInZip) Is(target error) bool {
	_, ok := target.(*ErrSubtitleNotFoundInZip)
	return ok
}

// SubtitleDownloader defines the interface for downloading subtitles
type SubtitleDownloader interface {
	// DownloadSubtitle downloads a subtitle, optionally extracting a specific episode from a season pack.
	// If episode is nil, the entire file is returned without extraction.
	// Returns ErrSubtitleNotFoundInZip if the requested episode is not found in a ZIP archive.
	DownloadSubtitle(ctx context.Context, downloadURL string, episode *int) (*models.DownloadResult, error)

	// Close releases any resources held by the downloader (e.g., cache connections).
	Close() error
}
