package testutil

import (
	"context"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// CollectSubtitles consumes a subtitle stream and returns a SubtitleCollection.
// This is a test helper and should not be used in production code.
func CollectSubtitles(ctx context.Context, stream <-chan models.StreamResult[models.Subtitle]) (*models.SubtitleCollection, error) {
	var subtitles []models.Subtitle
	for result := range stream {
		if result.Err != nil {
			return nil, result.Err
		}
		subtitles = append(subtitles, result.Value)
	}
	return buildSubtitleCollection(subtitles), nil
}

// CollectShowSubtitles consumes a ShowSubtitleItem stream and returns a slice of ShowSubtitles.
// This is a test helper and should not be used in production code.
func CollectShowSubtitles(ctx context.Context, stream <-chan models.StreamResult[models.ShowSubtitleItem]) ([]models.ShowSubtitles, error) {
	// Collect streamed items and group by show
	showInfoMap := make(map[int]*models.ShowInfo)
	subtitlesByShow := make(map[int][]models.Subtitle)
	var showOrder []int

	for item := range stream {
		if item.Err != nil {
			// Return on first error for test simplicity
			return nil, item.Err
		}
		if item.Value.ShowInfo != nil {
			sid := item.Value.ShowInfo.Show.ID
			showInfoMap[sid] = item.Value.ShowInfo
			showOrder = append(showOrder, sid)
		}
		if item.Value.Subtitle != nil {
			subtitlesByShow[item.Value.Subtitle.ShowID] = append(subtitlesByShow[item.Value.Subtitle.ShowID], *item.Value.Subtitle)
		}
	}

	// Build ShowSubtitles results in order
	var results []models.ShowSubtitles
	for _, sid := range showOrder {
		info := showInfoMap[sid]
		subs := subtitlesByShow[sid]
		showName := info.Show.Name
		if len(subs) > 0 {
			showName = subs[0].ShowName
		}
		results = append(results, models.ShowSubtitles{
			Show:          info.Show,
			ThirdPartyIds: info.ThirdPartyIds,
			SubtitleCollection: models.SubtitleCollection{
				ShowName:  showName,
				Subtitles: subs,
				Total:     len(subs),
			},
		})
	}

	return results, nil
}

// CollectShows consumes a Show stream and returns a slice of Shows.
// This is a test helper and should not be used in production code.
func CollectShows(ctx context.Context, stream <-chan models.StreamResult[models.Show]) ([]models.Show, error) {
	var shows []models.Show
	for result := range stream {
		if result.Err != nil {
			return nil, result.Err
		}
		shows = append(shows, result.Value)
	}
	return shows, nil
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
