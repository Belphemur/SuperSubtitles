package parser

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

func TestSubtitleParser_ParseHtmlWithPagination_ExampleOutlander(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="result">
				<tr>
					<th>Kategoria</th><th>Nyelv</th><th>Leiras</th><th>Feltolto</th><th>Ido</th><th>Letoltes</th>
				</tr>
				<tr>
					<td>cat</td>
					<td>Magyar</td>
					<td>
						<div class="magyar">Outlander - Az idegen - 7x16</div>
						<div class="eredeti">Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)</div>
					</td>
					<td>kissoreg</td>
					<td>2025-01-21</td>
					<td><a href="/index.php?action=letolt&fnev=outlander.s07e16.srt&felirat=1737439811">dl</a></td>
				</tr>
				<tr>
					<td colspan="7" id="adatlap">ignored</td>
				</tr>
			</table>
			<div class="pagination">
				<span class="current">1</span>
				<a href="/index.php?page=2">2</a>
			</div>
		</body>
		</html>
	`

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(result.Subtitles))
	}

	subtitle := result.Subtitles[0]
	if subtitle.Language != "Magyar" {
		t.Errorf("Expected language %q, got %q", "Magyar", subtitle.Language)
	}
	if !strings.Contains(subtitle.Name, "Outlander - Az idegen - 7x16") {
		t.Errorf("Expected name to contain episode title, got %q", subtitle.Name)
	}
	if subtitle.ShowName != "Outlander - Az idegen" {
		t.Errorf("Expected show name %q, got %q", "Outlander - Az idegen", subtitle.ShowName)
	}
	if subtitle.Season != 7 || subtitle.Episode != 16 {
		t.Errorf("Expected season 7 episode 16, got %d %d", subtitle.Season, subtitle.Episode)
	}
	if subtitle.IsSeasonPack {
		t.Errorf("Expected IsSeasonPack false")
	}

	expectedURL := "https://feliratok.eu/index.php?action=letolt&fnev=outlander.s07e16.srt&felirat=1737439811"
	if subtitle.DownloadURL != expectedURL {
		t.Errorf("Expected download URL %q, got %q", expectedURL, subtitle.DownloadURL)
	}
	if subtitle.ID != "1737439811" {
		t.Errorf("Expected ID %q, got %q", "1737439811", subtitle.ID)
	}
	if subtitle.Filename != "outlander.s07e16.srt" {
		t.Errorf("Expected filename %q, got %q", "outlander.s07e16.srt", subtitle.Filename)
	}
	if subtitle.Uploader != "kissoreg" {
		t.Errorf("Expected uploader %q, got %q", "kissoreg", subtitle.Uploader)
	}

	expectedDate := time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC)
	if !subtitle.UploadedAt.Equal(expectedDate) {
		t.Errorf("Expected uploaded date %v, got %v", expectedDate, subtitle.UploadedAt)
	}

	expectedQualities := []models.Quality{models.Quality720p, models.Quality1080p}
	if !reflect.DeepEqual(subtitle.Qualities, expectedQualities) {
		t.Errorf("Expected qualities %v, got %v", expectedQualities, subtitle.Qualities)
	}

	expectedGroups := []string{"FLUX", "SuccessfulCrab"}
	if !reflect.DeepEqual(subtitle.ReleaseGroups, expectedGroups) {
		t.Errorf("Expected release groups %v, got %v", expectedGroups, subtitle.ReleaseGroups)
	}

	if subtitle.Release != "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab" {
		t.Errorf("Expected release info %q, got %q", "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab", subtitle.Release)
	}

	if result.CurrentPage != 1 || result.TotalPages != 1 || result.HasNextPage {
		t.Errorf("Expected pagination 1/1 with no next page, got %d/%d next=%v", result.CurrentPage, result.TotalPages, result.HasNextPage)
	}
}

func TestSubtitleParser_ParseHtmlWithPagination_SeasonPack(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="result">
				<tr>
					<td>cat</td>
					<td>Magyar</td>
					<td>- Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.2160p-RAWR)</td>
					<td>gricsi</td>
					<td>2024-09-14</td>
					<td><a href="/index.php?action=letolt&fnev=billy.s02.zip&felirat=1726325505">dl</a></td>
				</tr>
			</table>
		</body>
		</html>
	`

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(result.Subtitles))
	}

	subtitle := result.Subtitles[0]
	if subtitle.ShowName != "Billy the Kid" {
		t.Errorf("Expected show name %q, got %q", "Billy the Kid", subtitle.ShowName)
	}
	if subtitle.Season != 2 || subtitle.Episode != -1 {
		t.Errorf("Expected season 2 episode -1, got %d %d", subtitle.Season, subtitle.Episode)
	}
	if !subtitle.IsSeasonPack {
		t.Errorf("Expected IsSeasonPack true")
	}

	expectedQualities := []models.Quality{models.Quality720p, models.Quality2160p}
	if !reflect.DeepEqual(subtitle.Qualities, expectedQualities) {
		t.Errorf("Expected qualities %v, got %v", expectedQualities, subtitle.Qualities)
	}

	expectedGroups := []string{"EDITH", "RAWR"}
	if !reflect.DeepEqual(subtitle.ReleaseGroups, expectedGroups) {
		t.Errorf("Expected release groups %v, got %v", expectedGroups, subtitle.ReleaseGroups)
	}

	if subtitle.Filename != "billy.s02.zip" {
		t.Errorf("Expected filename %q, got %q", "billy.s02.zip", subtitle.Filename)
	}
}

func TestSubtitleParser_ParseHtmlWithPagination_OldalPagination(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<div class="pagination">
				<span class="current">1</span>
				<a href="/index.php?oldal=2">2</a>
				<a href="/index.php?oldal=3">3</a>
			</div>
		</body>
		</html>
	`

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if result.CurrentPage != 1 || result.TotalPages != 3 || !result.HasNextPage {
		t.Errorf("Expected pagination 1/3 with next page, got %d/%d next=%v", result.CurrentPage, result.TotalPages, result.HasNextPage)
	}
}

func TestSubtitleParser_ParseHtml_ReturnsSubtitlesOnly(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="result">
				<tr>
					<td>cat</td>
					<td>Angol</td>
					<td>Outlander - 7x15 - Written in My Own Heart's Blood (AMZN.WEB-DL.720p-NTb)</td>
					<td>J1GG4</td>
					<td>2025-01-17</td>
					<td><a href="/index.php?action=letolt&fnev=outlander.s07e15.srt&felirat=1737139076">dl</a></td>
				</tr>
			</table>
		</body>
		</html>
	`

	parser := NewSubtitleParser("https://feliratok.eu")
	subtitles, err := parser.ParseHtml(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(subtitles))
	}
}

func TestSubtitleParser_ExtractFilenameFromDownloadLink_URLEncoded(t *testing.T) {
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name     string
		link     string
		expected string
	}{
		{
			name:     "URL encoded filename with spaces and special chars",
			link:     "/index.php?action=letolt&fnev=Billy%20The%20Kid%20-%2003x04%20-%20The%20Shepherds%20Hut.EDITH.English.C.orig.Addic7ed.com.srt&felirat=1760949698",
			expected: "Billy The Kid - 03x04 - The Shepherds Hut.EDITH.English.C.orig.Addic7ed.com.srt",
		},
		{
			name:     "URL encoded with parentheses",
			link:     "/index.php?action=letolt&fnev=Show%20Name%20%282024%29.srt&felirat=123456",
			expected: "Show Name (2024).srt",
		},
		{
			name:     "Simple filename without encoding",
			link:     "/index.php?action=letolt&fnev=outlander.s07e16.srt&felirat=1737439811",
			expected: "outlander.s07e16.srt",
		},
		{
			name:     "No fnev parameter",
			link:     "/index.php?action=letolt&felirat=123456",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractFilenameFromDownloadLink(tt.link)
			if result != tt.expected {
				t.Errorf("Expected filename %q, got %q", tt.expected, result)
			}
		})
	}
}
