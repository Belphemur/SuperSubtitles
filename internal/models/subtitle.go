package models

import (
	"time"
)

// Subtitle represents a normalized subtitle in our application
type Subtitle struct {
	ID            string    `json:"id"`
	ShowID        int       `json:"showId"`   // Show ID from feliratok.eu (extracted from category link)
	ShowName      string    `json:"showName"` // Show name (may be empty in HTML parsing)
	Name          string    `json:"name"`     // Subtitle name/title from HTML
	Language      string    `json:"language"`
	Season        int       `json:"season"`
	Episode       int       `json:"episode"`
	Filename      string    `json:"filename"` // Subtitle filename from download URL
	DownloadURL   string    `json:"downloadUrl"`
	Uploader      string    `json:"uploader"`
	UploadedAt    time.Time `json:"uploadedAt"`
	Qualities     []Quality `json:"qualities"`     // All matching qualities
	ReleaseGroups []string  `json:"releaseGroups"` // Multiple release groups (comma-separated in HTML)
	Release       string    `json:"release"`       // Release info (formats, quality) from HTML
	IsSeasonPack  bool      `json:"isSeasonPack"`
}

// SubtitleCollection represents a collection of subtitles for a show
type SubtitleCollection struct {
	ShowName  string     `json:"showName"`
	Subtitles []Subtitle `json:"subtitles"`
	Total     int        `json:"total"`
}
