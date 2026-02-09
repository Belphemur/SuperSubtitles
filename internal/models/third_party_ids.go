package models

// ThirdPartyIds represents identifiers from various third-party services
type ThirdPartyIds struct {
	IMDBID   string `json:"imdbId,omitempty"`   // IMDB identifier
	TVDBID   uint   `json:"tvdbId,omitempty"`   // TVDB identifier
	TVMazeID uint   `json:"tvMazeId,omitempty"` // TVMaze identifier
	TraktID  uint   `json:"traktId,omitempty"`  // Trakt identifier
}
