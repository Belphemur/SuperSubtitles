package parser

import (
	"strings"
	"testing"

	"SuperSubtitles/internal/models"
)

func TestShowParser_ParseHtml(t *testing.T) {
	// Sample HTML content based on the SuperSubtitles website structure
	htmlContent := `
		<html>
		<body>
			<table>
				<tbody>
					<tr>
						<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
							2025
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12190"><img class="kategk" src="sorozat_cat.php?kep=12190"/></a>
						</td>
						<td class="sangol">
							<div>
								7 Bears
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #fff">
						<td style="padding: 5px;">
							<a href="index.php?sid=12347"><img class="kategk" src="sorozat_cat.php?kep=12347"/></a>
						</td>
						<td class="sangol">
							<div>
								#1 Happy Family USA
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12549"><img class="kategk" src="sorozat_cat.php?kep=12549"/></a>
						</td>
						<td class="sangol">
							<div>
								A Thousand Blows
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #fff">
						<td style="padding: 5px;">
							<a href="index.php?sid=12076"><img class="kategk" src="sorozat_cat.php?kep=12076"/></a>
						</td>
						<td class="sangol">
							<div>
								Adults
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr>
						<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
							2024
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12007"><img class="kategk" src="sorozat_cat.php?kep=12007"/></a>
						</td>
						<td class="sangol">
							<div>
								Asura
							</div>
							<div class="sev"></div>
						</td>
					</tr>
				</tbody>
			</table>
		</body>
		</html>
	`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	// Test that parsing succeeds
	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	// Test that we got the expected number of shows
	expectedCount := 5
	if len(shows) != expectedCount {
		t.Errorf("Expected %d shows, got %d", expectedCount, len(shows))
	}

	// Test specific shows
	expectedShows := []models.Show{
		{Name: "7 Bears", ID: 12190, Year: 2025, ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12190"},
		{Name: "#1 Happy Family USA", ID: 12347, Year: 2025, ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12347"},
		{Name: "A Thousand Blows", ID: 12549, Year: 2025, ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12549"},
		{Name: "Adults", ID: 12076, Year: 2025, ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12076"},
		{Name: "Asura", ID: 12007, Year: 2024, ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12007"},
	}

	for i, expected := range expectedShows {
		if i >= len(shows) {
			t.Errorf("Missing show at index %d", i)
			continue
		}

		actual := shows[i]
		if actual.Name != expected.Name {
			t.Errorf("Show %d: expected name %q, got %q", i, expected.Name, actual.Name)
		}
		if actual.ID != expected.ID {
			t.Errorf("Show %d: expected ID %q, got %q", i, expected.ID, actual.ID)
		}
		if actual.Year != expected.Year {
			t.Errorf("Show %d: expected year %d, got %d", i, expected.Year, actual.Year)
		}
		if actual.ImageURL != expected.ImageURL {
			t.Errorf("Show %d: expected imageURL %q, got %q", i, expected.ImageURL, actual.ImageURL)
		}
	}
}

func TestShowParser_ParseHtml_EmptyHTML(t *testing.T) {
	htmlContent := `<html><body></body></html>`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed on empty HTML: %v", err)
	}

	if len(shows) != 0 {
		t.Errorf("Expected 0 shows from empty HTML, got %d", len(shows))
	}
}

func TestShowParser_ParseHtml_InvalidHTML(t *testing.T) {
	htmlContent := `<html><body><table><tr><td>Invalid structure</td></tr></table></body></html>`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed on invalid HTML: %v", err)
	}

	// Should return empty slice for HTML without proper show structure
	if len(shows) != 0 {
		t.Errorf("Expected 0 shows from invalid HTML structure, got %d", len(shows))
	}
}

func TestShowParser_ParseHtml_MalformedYear(t *testing.T) {
	htmlContent := `
		<html><body>
			<table>
				<tr>
					<td colspan="10">Invalid Year</td>
				</tr>
				<tr>
					<td><a href="index.php?sid=12345"><img src="sorozat_cat.php?kep=12345"/></a></td>
					<td class="sangol"><div>Test Show</div></td>
				</tr>
			</table>
		</body></html>
	`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(shows) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(shows))
	}

	// Year should be 0 (default) when year parsing fails
	if shows[0].Year != 0 {
		t.Errorf("Expected year 0 for malformed year header, got %d", shows[0].Year)
	}
}

func TestShowParser_ParseHtml_MissingImage(t *testing.T) {
	htmlContent := `
		<html><body>
			<table>
				<tr>
					<td><a href="index.php?sid=12345"><img/></a></td>
					<td class="sangol"><div>Test Show</div></td>
				</tr>
			</table>
		</body></html>
	`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	// Should skip shows with missing images
	if len(shows) != 0 {
		t.Errorf("Expected 0 shows when image is missing, got %d", len(shows))
	}
}

func TestShowParser_ParseHtml_MissingName(t *testing.T) {
	htmlContent := `
		<html><body>
			<table>
				<tr>
					<td><a href="index.php?sid=12345"><img src="sorozat_cat.php?kep=12345"/></a></td>
					<td class="sangol"></td>
				</tr>
			</table>
		</body></html>
	`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(shows) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(shows))
	}

	// Should use fallback name when actual name is missing
	expectedName := "Show 12345"
	if shows[0].Name != expectedName {
		t.Errorf("Expected fallback name %q, got %q", expectedName, shows[0].Name)
	}
}

func TestShowParser_ParseHtml_Simple(t *testing.T) {
	htmlContent := `
		<html><body>
			<table>
				<tr>
					<td colspan="10">2025</td>
				</tr>
				<tr>
					<td><a href="index.php?sid=12345"><img src="sorozat_cat.php?kep=12345"/></a></td>
					<td class="sangol"><div>Test Show</div></td>
				</tr>
			</table>
		</body></html>
	`

	parser := NewShowParser("https://feliratok.eu")
	shows, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(shows) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(shows))
	}

	expected := models.Show{
		Name:     "Test Show",
		ID:       12345,
		Year:     2025,
		ImageURL: "https://feliratok.eu/sorozat_cat.php?kep=12345",
	}

	actual := shows[0]
	if actual.Name != expected.Name {
		t.Errorf("Expected name %q, got %q", expected.Name, actual.Name)
	}
	if actual.ID != expected.ID {
		t.Errorf("Expected ID %q, got %q", expected.ID, actual.ID)
	}
	if actual.Year != expected.Year {
		t.Errorf("Expected year %d, got %d", expected.Year, actual.Year)
	}
	if actual.ImageURL != expected.ImageURL {
		t.Errorf("Expected imageURL %q, got %q", expected.ImageURL, actual.ImageURL)
	}
}

func TestShowParser_extractIDFromHref(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	tests := []struct {
		href     string
		expected int
	}{
		{"index.php?sid=12345", 12345},
		{"index.php?sid=123", 123},
		{"index.php?sid=", 0},
		{"index.php?sid=abc123", 0}, // Invalid number should return 0
		{"other.php?sid=12345", 0},
		{"index.php?other=12345", 0},
	}

	for _, test := range tests {
		result := parser.extractIDFromHref(test.href)
		if result != test.expected {
			t.Errorf("extractIDFromHref(%q) = %d, expected %d", test.href, result, test.expected)
		}
	}
}

func TestShowParser_extractImageURL(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	tests := []struct {
		src      string
		expected string
	}{
		{"sorozat_cat.php?kep=12345", "https://feliratok.eu/sorozat_cat.php?kep=12345"},
		{"sorozat_cat.php?kep=abc123", "https://feliratok.eu/sorozat_cat.php?kep=abc123"},
		{"sorozat_cat.php?kep=", "https://feliratok.eu/sorozat_cat.php?kep="},
		{"other.php?kep=12345", ""},
		{"sorozat_cat.php?other=12345", ""},
	}

	for _, test := range tests {
		result := parser.extractImageURL(test.src)
		if result != test.expected {
			t.Errorf("extractImageURL(%q) = %q, expected %q", test.src, result, test.expected)
		}
	}
}
