package models

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// UpdateCheckResponse represents the response from the recheck endpoint
// The values represent the count of available episodes/movies since the given ID
type UpdateCheckResponse struct {
	Film    int `json:"film"`    // Number of films available since the given episode ID
	Sorozat int `json:"sorozat"` // Number of series episodes available since the given episode ID
}

// UnmarshalJSON implements custom JSON unmarshaling for UpdateCheckResponse
// to handle both string and integer values for film and sorozat fields
func (u *UpdateCheckResponse) UnmarshalJSON(data []byte) error {
	type alias UpdateCheckResponse
	aux := struct {
		Film    interface{} `json:"film"`
		Sorozat interface{} `json:"sorozat"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Helper function to convert interface{} to int
	convertToInt := func(v interface{}) (int, error) {
		switch val := v.(type) {
		case float64:
			return int(val), nil
		case string:
			return strconv.Atoi(val)
		case nil:
			return 0, nil
		default:
			return 0, fmt.Errorf("cannot convert %T to int", v)
		}
	}

	filmCount, err := convertToInt(aux.Film)
	if err != nil {
		return fmt.Errorf("failed to parse film count: %w", err)
	}
	u.Film = filmCount

	seriesCount, err := convertToInt(aux.Sorozat)
	if err != nil {
		return fmt.Errorf("failed to parse sorozat count: %w", err)
	}
	u.Sorozat = seriesCount

	return nil
}

// UpdateCheckResult represents the normalized result of an update check
type UpdateCheckResult struct {
	FilmCount   int  `json:"filmCount"`   // Number of films available since the given episode ID
	SeriesCount int  `json:"seriesCount"` // Number of series episodes available since the given episode ID
	HasUpdates  bool `json:"hasUpdates"`  // True if there are any updates available (count > 0)
}
