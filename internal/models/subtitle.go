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
	ID           string    `json:"id"`
	ShowName     string    `json:"showName"`
	Language     string    `json:"language"`
	Season       int       `json:"season"`
	Episode      int       `json:"episode"`
	Filename     string    `json:"filename"`
	DownloadURL  string    `json:"downloadUrl"`
	Uploader     string    `json:"uploader"`
	UploadedAt   time.Time `json:"uploadedAt"`
	Quality      Quality   `json:"quality"`      // Video quality enum
	ReleaseGroup string    `json:"releaseGroup"` // Original name from API
	Source       string    `json:"source"`       // Original name from API
	IsSeasonPack bool      `json:"isSeasonPack"`
	ExactMatch   int       `json:"exactMatch"` // Converted exact match score
}

// SubtitleCollection represents a collection of subtitles for a show
type SubtitleCollection struct {
	ShowName  string     `json:"showName"`
	Subtitles []Subtitle `json:"subtitles"`
	Total     int        `json:"total"`
}
