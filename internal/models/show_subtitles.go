package models

// ShowSubtitles represents a TV show with its third-party service IDs and subtitle collection
type ShowSubtitles struct {
	Show               `json:",inline"`   // Embedded Show struct with Name, ID, Year, ImageURL
	ThirdPartyIds      ThirdPartyIds      `json:"thirdPartyIds"`      // Third-party service identifiers (IMDB, TVDB, TVMaze, Trakt)
	SubtitleCollection SubtitleCollection `json:"subtitleCollection"` // All subtitles for this show
}
