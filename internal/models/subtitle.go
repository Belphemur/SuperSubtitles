package models

import (
	"time"
)

// SuperSubtitle represents the raw subtitle data from SuperSubtitles API response
type SuperSubtitle struct {
	Language     string `json:"language"`       // Language of the subtitle (e.g., "Angol", "Magyar")
	Name         string `json:"nev"`            // Name/title with episode/season info
	BaseLink     string `json:"baselink"`       // Base URL for the subtitle service
	Filename     string `json:"fnev"`           // Filename of the subtitle
	SubtitleID   string `json:"felirat"`        // Timestamp ID for the subtitle
	Season       string `json:"evad"`           // Season number ("-1" for season packs)
	Episode      string `json:"ep"`             // Episode number ("-1" for season packs)
	Uploader     string `json:"feltolto"`       // Name of the uploader
	ExactMatch   string `json:"pontos_talalat"` // Exact match score
	IsSeasonPack string `json:"evadpakk"`       // Whether it's a season pack ("1" or "0")
}

// SuperSubtitleResponse represents the complete API response from SuperSubtitles
// The keys are string IDs and values are SuperSubtitle objects
type SuperSubtitleResponse map[string]SuperSubtitle

// Subtitle represents a normalized subtitle in our application
type Subtitle struct {
	ID            string    `json:"id"`
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
