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

// CollectShowSubtitles consumes a ShowSubtitleItem stream and returns a slice of ShowSubtitles.
// This is a test helper and should not be used in production code.
func CollectShowSubtitles(ctx context.Context, stream <-chan models.StreamResult[models.ShowSubtitleItem]) ([]models.ShowSubtitles, error) {
	// Collect streamed items and group by show
	showInfoMap := make(map[int]*models.ShowInfo)
	subtitlesByShow := make(map[int][]models.Subtitle)
	showOrderMap := make(map[int]bool) // Track which shows are already in showOrder
	var showOrder []int

	// Helper to ensure a show ID is in showOrder exactly once
	ensureShowInOrder := func(sid int) {
		if !showOrderMap[sid] {
			showOrder = append(showOrder, sid)
			showOrderMap[sid] = true
		}
	}

	for {
		select {
		case item, ok := <-stream:
			if !ok {
				// Channel closed, build results
				return buildShowSubtitlesResults(showInfoMap, subtitlesByShow, showOrder), nil
			}
			if item.Err != nil {
				// Return on first error for test simplicity
				return nil, item.Err
			}
			if item.Value.ShowInfo != nil {
				sid := item.Value.ShowInfo.Show.ID
				showInfoMap[sid] = item.Value.ShowInfo
				ensureShowInOrder(sid)
			}
			if item.Value.Subtitle != nil {
				sid := item.Value.Subtitle.ShowID
				subtitlesByShow[sid] = append(subtitlesByShow[sid], *item.Value.Subtitle)
				// Ensure showOrder includes shows that appear only in subtitles
				ensureShowInOrder(sid)
			}
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

// buildShowSubtitlesResults constructs ShowSubtitles results from collected data.
// Handles cases where ShowInfo may be missing for some shows (subtitles-only).
func buildShowSubtitlesResults(showInfoMap map[int]*models.ShowInfo, subtitlesByShow map[int][]models.Subtitle, showOrder []int) []models.ShowSubtitles {
	var results []models.ShowSubtitles
	for _, sid := range showOrder {
		info := showInfoMap[sid]
		subs := subtitlesByShow[sid]

		var (
			show          models.Show
			thirdPartyIds models.ThirdPartyIds
			showName      string
		)

		if info != nil {
			show = info.Show
			thirdPartyIds = info.ThirdPartyIds
			showName = info.Show.Name
		}

		if len(subs) > 0 {
			// ShowName from subtitles is more reliable than ShowInfo for display
			showName = subs[0].ShowName
		}

		results = append(results, models.ShowSubtitles{
			Show:          show,
			ThirdPartyIds: thirdPartyIds,
			SubtitleCollection: models.SubtitleCollection{
				ShowName:  showName,
				Subtitles: subs,
				Total:     len(subs),
			},
		})
	}
	return results
}
