package models

// Show represents a TV show with basic information
type Show struct {
	Name     string `json:"name"`
	ID       uint   `json:"id"`
	Year     uint   `json:"year"`
	ImageURL string `json:"imageUrl"`
}
