package services

import (
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// SubtitleConverter defines the interface for converting subtitle data
type SubtitleConverter interface {
	// ConvertSuperSubtitle converts a single SuperSubtitle to normalized Subtitle
	ConvertSuperSubtitle(superSub *models.SuperSubtitle) models.Subtitle

	// ConvertResponse converts a SuperSubtitleResponse to a SubtitleCollection
	ConvertResponse(response models.SuperSubtitleResponse) models.SubtitleCollection
}
