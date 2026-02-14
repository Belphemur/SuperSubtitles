package models

// ShowSubtitles represents a TV show with its third-party service IDs and subtitle collection
type ShowSubtitles struct {
	Show               `json:",inline"`   // Embedded Show struct with Name, ID, Year, ImageURL
	ThirdPartyIds      ThirdPartyIds      `json:"thirdPartyIds"`      // Third-party service identifiers (IMDB, TVDB, TVMaze, Trakt)
	SubtitleCollection SubtitleCollection `json:"subtitleCollection"` // All subtitles for this show
}

// ShowInfo represents a TV show with its third-party IDs (without subtitles)
type ShowInfo struct {
	Show          Show          `json:"show"`
	ThirdPartyIds ThirdPartyIds `json:"thirdPartyIds"`
}

// ShowSubtitleItem represents a streaming item that is either show info or a subtitle.
// Exactly one of ShowInfo or Subtitle will be non-nil.
type ShowSubtitleItem struct {
	ShowInfo *ShowInfo `json:"showInfo,omitempty"`
	Subtitle *Subtitle `json:"subtitle,omitempty"`
}
