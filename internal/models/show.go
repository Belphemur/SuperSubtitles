package models

// Show represents a TV show with basic information
type Show struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Year     int    `json:"year"`
	ImageURL string `json:"imageUrl"`
}
