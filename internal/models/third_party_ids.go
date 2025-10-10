package models

// ThirdPartyIds represents identifiers from various third-party services
type ThirdPartyIds struct {
	IMDBID   string `json:"imdbId,omitempty"`   // IMDB identifier
	TVDBID   int    `json:"tvdbId,omitempty"`   // TVDB identifier
	TVMazeID int    `json:"tvMazeId,omitempty"` // TVMaze identifier
	TraktID  int    `json:"traktId,omitempty"`  // Trakt identifier
}
