package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

func TestSubtitleParser_ParseHtml_SingleSubtitle(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/felirat/123/sulis-ott-es-itt-7x16">Szulics Ótt és Ott - 7x16 Az örvény (WEB.720p-FLUX, AMZN.1080p-SUCCESS)</a></td>
				<td>uploader_name</td>
				<td>2024-01-15</td>
				<td><a href="/download/123">Download</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subtitles))
	}

	subtitle := subtitles[0]
	if subtitle.Name == "" {
		t.Error("expected Name to be set")
	}
	if subtitle.Language != "Magyar" {
		t.Errorf("expected Language to be 'Magyar', got %q", subtitle.Language)
	}
	if subtitle.Uploader != "uploader_name" {
		t.Errorf("expected Uploader to be 'uploader_name', got %q", subtitle.Uploader)
	}
	if !subtitle.UploadedAt.Equal(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected UploadedAt to be 2024-01-15, got %v", subtitle.UploadedAt)
	}
}

func TestSubtitleParser_ParseHtml_MultipleReleaseGroups(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/felirat/456/test-1x01">Test Show - 1x01 Pilot (WEB.720p-FLUX, AMZN.1080p-SUCCESS, NF.1080p-PREMIUM)</a></td>
				<td>user1</td>
				<td>2024-02-01</td>
				<td><a href="/download/456">Download</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subtitles))
	}

	subtitle := subtitles[0]
	if len(subtitle.ReleaseGroups) != 3 {
		t.Fatalf("expected 3 release groups, got %d: %v", len(subtitle.ReleaseGroups), subtitle.ReleaseGroups)
	}

	expectedGroups := []string{"FLUX", "SUCCESS", "PREMIUM"}
	for i, expected := range expectedGroups {
		if i >= len(subtitle.ReleaseGroups) {
			t.Fatalf("missing release group at index %d", i)
		}
		if subtitle.ReleaseGroups[i] != expected {
			t.Errorf("expected release group[%d] to be %q, got %q", i, expected, subtitle.ReleaseGroups[i])
		}
	}

	if subtitle.Quality != models.Quality720p {
		t.Errorf("expected Quality to be Quality720p, got %v", subtitle.Quality)
	}
}

func TestSubtitleParser_ParseHtml_SeasonPack(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/felirat/789/billy-the-kid-season-2">Billy the Kid (Season 2) (NF.1080p-EDITH, AMZN.WEB-DL.720p-FLUX)</a></td>
				<td>season_user</td>
				<td>2024-02-10</td>
				<td><a href="/download/789">Download</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subtitles))
	}

	subtitle := subtitles[0]
	if !subtitle.IsSeasonPack {
		t.Error("expected IsSeasonPack to be true")
	}
	if subtitle.Season != 2 {
		t.Errorf("expected Season to be 2, got %d", subtitle.Season)
	}
	if subtitle.Episode != -1 {
		t.Errorf("expected Episode to be -1 for season pack, got %d", subtitle.Episode)
	}
	if len(subtitle.ReleaseGroups) != 2 {
		t.Fatalf("expected 2 release groups, got %d", len(subtitle.ReleaseGroups))
	}
}

func TestSubtitleParser_ParseHtml_QualityDetection(t *testing.T) {
	tests := []struct {
		releaseInfo string
		expected    models.Quality
	}{
		{"2160p-GROUP", models.Quality2160p},
		{"4K-HDR-GROUP", models.Quality2160p},
		{"1080p-GROUP", models.Quality1080p},
		{"720p-GROUP", models.Quality720p},
		{"480p-GROUP", models.Quality480p},
		{"360p-GROUP", models.Quality360p},
		{"UNKNOWN-GROUP", models.QualityUnknown},
	}

	for _, tt := range tests {
		html := `
		<html>
		<body>
			<table>
				<tr>
					<td>Magyar</td>
					<td><a href="/test">Test - 1x01 (` + tt.releaseInfo + `)</a></td>
					<td>user</td>
					<td>2024-01-01</td>
					<td><a href="/download/1">Download</a></td>
				</tr>
			</table>
		</body>
		</html>
		`

		parser := NewSubtitleParser("https://example.com")
		subtitles, err := parser.ParseHtml(strings.NewReader(html))

		if err != nil {
			t.Fatalf("ParseHtml failed for %q: %v", tt.releaseInfo, err)
		}

		if len(subtitles) == 0 {
			t.Fatalf("no subtitles parsed for %q", tt.releaseInfo)
		}

		if subtitles[0].Quality != tt.expected {
			t.Errorf("for %q: expected Quality %v, got %v", tt.releaseInfo, tt.expected, subtitles[0].Quality)
		}
	}
}

func TestSubtitleParser_ParseHtml_MultipleSubtitles(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/test1">Show 1 - 1x01 (WEB-GROUP1)</a></td>
				<td>user1</td>
				<td>2024-01-01</td>
				<td><a href="/download/1">Download</a></td>
			</tr>
			<tr>
				<td>English</td>
				<td><a href="/test2">Show 2 - 2x05 (WEB-GROUP2)</a></td>
				<td>user2</td>
				<td>2024-01-02</td>
				<td><a href="/download/2">Download</a></td>
			</tr>
			<tr>
				<td>Portuguese</td>
				<td><a href="/test3">Show 3 (Season 1) (WEB-GROUP3)</a></td>
				<td>user3</td>
				<td>2024-01-03</td>
				<td><a href="/download/3">Download</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 3 {
		t.Fatalf("expected 3 subtitles, got %d", len(subtitles))
	}

	expectedLanguages := []string{"Magyar", "English", "Portuguese"}
	for i, expected := range expectedLanguages {
		if subtitles[i].Language != expected {
			t.Errorf("subtitle[%d]: expected Language %q, got %q", i, expected, subtitles[i].Language)
		}
	}
}

func TestSubtitleParser_ParseHtml_IgnoresInvalidRows(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Only</td>
				<td>Two</td>
			</tr>
			<tr>
				<td>Magyar</td>
				<td><a href="/test">Test - 1x01 (GROUP)</a></td>
				<td>user</td>
				<td>2024-01-01</td>
				<td><a href="/download/1">Download</a></td>
			</tr>
			<tr>
				<td></td>
				<td></td>
				<td></td>
				<td></td>
				<td></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("expected 1 valid subtitle, got %d", len(subtitles))
	}
}

func TestSubtitleParser_ParseHtml_NoDownloadLink(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td>Test - 1x01 (GROUP)</td>
				<td>user</td>
				<td>2024-01-01</td>
				<td><a href="/download/1">Download</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	subtitles, err := parser.ParseHtml(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 0 {
		t.Fatalf("expected 0 subtitles (no download link), got %d", len(subtitles))
	}
}

func TestSubtitleParser_ParseDescription_Episode(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		description  string
		showName     string
		season       int
		episode      int
		isSeasonPack bool
	}{
		{
			"Outlander - Az idegen - 7x16 Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab, AMZN.WEB-DL.1080p-FLUX) új évadra vár (kissoreg)",
			"Outlander - Az idegen",
			7,
			16,
			false,
		},
		{
			"- Billy the Kid - 3x07 - The Last Buffalo (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR, AMZN.WEB-DL.2160p-RAWR) fordítás alatt (gricsi)",
			"Billy the Kid",
			3,
			7,
			false,
		},
		{
			"- Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.720p-FLUX, AMZN.WEB-DL.720p-MADSKY, WEB.1080p-EDITH, WEB.1080p-LAZYCUNTS, WEB.1080p-SuccessfulCrab, AMZN.WEB-DL.1080p-FLUX, AMZN.WEB-DL.1080p-MADSKY) fordítás alatt (gricsi)",
			"Billy the Kid",
			2,
			-1,
			true,
		},
		{
			"Something - 1x01 Pilot (GROUP)",
			"Something",
			1,
			1,
			false,
		},
	}

	for _, tt := range tests {
		showName, season, episode, _, isSeasonPack := parser.parseDescription(tt.description)

		if showName != tt.showName {
			t.Errorf("description %q: expected showName %q, got %q", tt.description, tt.showName, showName)
		}
		if season != tt.season {
			t.Errorf("description %q: expected season %d, got %d", tt.description, tt.season, season)
		}
		if episode != tt.episode {
			t.Errorf("description %q: expected episode %d, got %d", tt.description, tt.episode, episode)
		}
		if isSeasonPack != tt.isSeasonPack {
			t.Errorf("description %q: expected isSeasonPack %v, got %v", tt.description, tt.isSeasonPack, isSeasonPack)
		}
	}
}

func TestSubtitleParser_ParseReleaseInfo_MultipleGroups(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		releaseInfo     string
		expectedCount   int
		expectedFirst   string
		expectedQuality models.Quality
	}{
		{
			"AMZN.WEB-DL.720p-FLUX",
			1,
			"FLUX",
			models.Quality720p,
		},
		{
			"WEB.720p-SuccessfulCrab, AMZN.WEB-DL.1080p-PREMIUM",
			2,
			"SuccessfulCrab",
			models.Quality720p,
		},
		{
			"NF.1080p-EDITH, AMZN.WebDL.720p-FLUX, WEB.720p-CROWN",
			3,
			"EDITH",
			models.Quality1080p,
		},
	}

	for _, tt := range tests {
		quality, groups := parser.parseReleaseInfo(tt.releaseInfo)

		if len(groups) != tt.expectedCount {
			t.Errorf("release info %q: expected %d groups, got %d", tt.releaseInfo, tt.expectedCount, len(groups))
		}

		if len(groups) > 0 && groups[0] != tt.expectedFirst {
			t.Errorf("release info %q: expected first group %q, got %q", tt.releaseInfo, tt.expectedFirst, groups[0])
		}

		if quality != tt.expectedQuality {
			t.Errorf("release info %q: expected quality %v, got %v", tt.releaseInfo, tt.expectedQuality, quality)
		}
	}
}

func TestSubtitleParser_DetectQuality(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		release  string
		expected models.Quality
	}{
		{"something-2160p-GROUP", models.Quality2160p},
		{"something-4K-GROUP", models.Quality2160p},
		{"something-1080p-GROUP", models.Quality1080p},
		{"something-720p-GROUP", models.Quality720p},
		{"something-480p-GROUP", models.Quality480p},
		{"something-360p-GROUP", models.Quality360p},
		{"SD-GROUP", models.QualityUnknown},
		{"", models.QualityUnknown},
	}

	for _, tt := range tests {
		result := parser.detectQuality(tt.release)
		if result != tt.expected {
			t.Errorf("release %q: expected %v, got %v", tt.release, tt.expected, result)
		}
	}
}

func TestSubtitleParser_ConstructDownloadURL(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		link     string
		expected string
	}{
		{
			"/download/123",
			"https://example.com/download/123",
		},
		{
			"download/123",
			"https://example.com/download/123",
		},
		{
			"https://other.com/file",
			"https://other.com/file",
		},
		{
			"http://cdn.example.com/file",
			"http://cdn.example.com/file",
		},
	}

	for _, tt := range tests {
		result := parser.constructDownloadURL(tt.link)
		if result != tt.expected {
			t.Errorf("link %q: expected %q, got %q", tt.link, tt.expected, result)
		}
	}
}

func TestSubtitleParser_ParseDate(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		dateStr  string
		expected time.Time
	}{
		{
			"2024-01-15",
			time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			"2023-12-31",
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			"",
			time.Time{},
		},
		{
			"invalid",
			time.Time{},
		},
	}

	for _, tt := range tests {
		result := parser.parseDate(tt.dateStr)
		if !result.Equal(tt.expected) {
			t.Errorf("date %q: expected %v, got %v", tt.dateStr, tt.expected, result)
		}
	}
}

func TestSubtitleParser_ExtractIDFromDownloadLink(t *testing.T) {
	parser := NewSubtitleParser("https://example.com")

	tests := []struct {
		link     string
		expected string
	}{
		{
			"/felirat/123/slug",
			"123",
		},
		{
			"/felirat/456",
			"456",
		},
		{
			"/download?id=789",
			"789",
		},
		{
			"/path/999/file",
			"999",
		},
		{
			"unknown-link",
			"unknown-link",
		},
	}

	for _, tt := range tests {
		result := parser.extractIDFromDownloadLink(tt.link)
		if result != tt.expected {
			t.Errorf("link %q: expected %q, got %q", tt.link, tt.expected, result)
		}
	}
}

func TestSubtitleParser_ParseHtmlWithPagination(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/test">Test - 1x01 (GROUP)</a></td>
				<td>user</td>
				<td>2024-01-01</td>
				<td><a href="/download/1">Download</a></td>
			</tr>
		</table>
		<div class="pagination">
			<a href="?oldal=1">1</a>
			<a href="?oldal=2">2</a>
			<a href="?oldal=3">3</a>
		</div>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(result.Subtitles))
	}

	if result.CurrentPage != 1 {
		t.Errorf("expected CurrentPage to be 1, got %d", result.CurrentPage)
	}

	if result.TotalPages != 3 {
		t.Errorf("expected TotalPages to be 3, got %d", result.TotalPages)
	}

	if !result.HasNextPage {
		t.Error("expected HasNextPage to be true")
	}
}

func TestSubtitleParser_ParseHtmlWithPagination_LastPageNoNextPage(t *testing.T) {
	html := `
	<html>
	<body>
		<table>
			<tr>
				<td>Magyar</td>
				<td><a href="/test">Test - 1x01 (GROUP)</a></td>
				<td>user</td>
				<td>2024-01-01</td>
				<td><a href="/download/1">Download</a></td>
			</tr>
		</table>
		<div class="pagination">
			<a href="?oldal=1">1</a>
		</div>
	</body>
	</html>
	`

	parser := NewSubtitleParser("https://example.com")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(html))

	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if result.TotalPages != 1 {
		t.Errorf("expected TotalPages to be 1, got %d", result.TotalPages)
	}

	if result.HasNextPage {
		t.Error("expected HasNextPage to be false on last page")
	}
}
