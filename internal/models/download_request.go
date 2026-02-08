package models

// DownloadRequest represents a request to download a specific subtitle
type DownloadRequest struct {
	SubtitleID string // The subtitle ID from the API
	Episode    int    // Episode number to extract from season pack (0 = download entire file)
}

// DownloadResult represents the result of a subtitle download
type DownloadResult struct {
	Filename    string // Name of the subtitle file
	Content     []byte // Content of the subtitle file
	ContentType string // MIME type (e.g., "application/x-subrip", "application/zip")
}
