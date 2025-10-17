package models

// UpdateCheckResponse represents the response from the recheck endpoint
// The values represent the count of available episodes/movies since the given ID
type UpdateCheckResponse struct {
	Film    string `json:"film"`    // Number of films available since the given episode ID
	Sorozat string `json:"sorozat"` // Number of series episodes available since the given episode ID
}

// UpdateCheckResult represents the normalized result of an update check
type UpdateCheckResult struct {
	FilmCount   int  `json:"filmCount"`   // Number of films available since the given episode ID
	SeriesCount int  `json:"seriesCount"` // Number of series episodes available since the given episode ID
	HasUpdates  bool `json:"hasUpdates"`  // True if there are any updates available (count > 0)
}
