package testutil

import (
	"context"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// CollectSubtitles consumes a subtitle stream and returns a SubtitleCollection.
// This is a test helper and should not be used in production code.
func CollectSubtitles(ctx context.Context, stream <-chan models.StreamResult[models.Subtitle]) (*models.SubtitleCollection, error) {
	var subtitles []models.Subtitle
	for {
		select {
		case result, ok := <-stream:
			if !ok {
				return buildSubtitleCollection(subtitles), nil
			}
			if result.Err != nil {
				return nil, result.Err
			}
			subtitles = append(subtitles, result.Value)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// CollectShowSubtitles consumes a ShowSubtitles stream and returns a slice of ShowSubtitles.
// This is a test helper and should not be used in production code.
func CollectShowSubtitles(ctx context.Context, stream <-chan models.StreamResult[models.ShowSubtitles]) ([]models.ShowSubtitles, error) {
	var results []models.ShowSubtitles
	for {
		select {
		case item, ok := <-stream:
			if !ok {
				return results, nil
			}
			if item.Err != nil {
				return nil, item.Err
			}
			results = append(results, item.Value)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// CollectShows consumes a Show stream and returns a slice of Shows.
// This is a test helper and should not be used in production code.
func CollectShows(ctx context.Context, stream <-chan models.StreamResult[models.Show]) ([]models.Show, error) {
	var shows []models.Show
	for {
		select {
		case result, ok := <-stream:
			if !ok {
				return shows, nil
			}
			if result.Err != nil {
				return nil, result.Err
			}
			shows = append(shows, result.Value)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// buildSubtitleCollection constructs a SubtitleCollection from subtitles
func buildSubtitleCollection(subtitles []models.Subtitle) *models.SubtitleCollection {
	showName := ""
	if len(subtitles) > 0 {
		showName = subtitles[0].ShowName
	}

	return &models.SubtitleCollection{
		ShowName:  showName,
		Subtitles: subtitles,
		Total:     len(subtitles),
	}
}
