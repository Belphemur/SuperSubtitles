package models

// DownloadResult represents the result of a subtitle download
type DownloadResult struct {
	Filename    string // Name of the subtitle file
	Content     []byte // Content of the subtitle file
	ContentType string // MIME type (e.g., "application/x-subrip", "application/zip")
}
