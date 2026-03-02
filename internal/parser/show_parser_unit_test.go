// show_parser_unit_test.go tests individual show parser helper functions in isolation.
// The companion show_parser_test.go uses testutil.GenerateShowTableHTML for integration-style
// tests; this file focuses on unit testing individual unexported methods directly using goquery.
package parser

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// ---------------------------------------------------------------------------
// extractShowNameFromGoquery
// ---------------------------------------------------------------------------

func TestShowParser_extractShowNameFromGoquery(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "no parent td found",
			html: `<div><a href="test">link</a></div>`,
			want: "",
		},
		{
			name: "next sibling td has no sangol class",
			html: `<table><tr><td><a href="test">link</a></td><td>name</td></tr></table>`,
			want: "",
		},
		{
			name: "no div found in td.sangol",
			html: `<table><tr><td><a href="test">link</a></td><td class="sangol"><span>name</span></td></tr></table>`,
			want: "",
		},
		{
			name: "name is Tuiskoms",
			html: `<table><tr><td><a href="test">link</a></td><td class="sangol"><div>(Tuiskoms)</div></td></tr></table>`,
			want: "",
		},
		{
			name: "valid extraction",
			html: `<table><tr><td><a href="test">link</a></td><td class="sangol"><div>Breaking Bad</div></td></tr></table>`,
			want: "Breaking Bad",
		},
		{
			name: "empty div text",
			html: `<table><tr><td><a href="test">link</a></td><td class="sangol"><div>  </div></td></tr></table>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}
			link := doc.Find("a").First()
			got := parser.extractShowNameFromGoquery(link)
			if got != tt.want {
				t.Errorf("extractShowNameFromGoquery() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ExtractLastPage
// ---------------------------------------------------------------------------

func TestShowParser_ExtractLastPage_NoPaginationDiv(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	html := `<html><body><p>No pagination here</p></body></html>`
	got := parser.ExtractLastPage(strings.NewReader(html))
	if got != 1 {
		t.Errorf("ExtractLastPage() = %d, want 1 for missing pagination div", got)
	}
}

func TestShowParser_ExtractLastPage_NonNumericLinkText(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	// Navigation links like ">" contain oldal= in href but non-numeric text
	html := `<html><body>
		<div class="pagination">
			<a href="/index.php?oldal=2&sorf=abc">></a>
			<a href="/index.php?oldal=3&sorf=abc">3</a>
		</div>
	</body></html>`

	got := parser.ExtractLastPage(strings.NewReader(html))
	// ">" is non-numeric so it's skipped; only "3" is parsed
	if got != 3 {
		t.Errorf("ExtractLastPage() = %d, want 3", got)
	}
}

func TestShowParser_ExtractLastPage_MultiplePages(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	html := `<html><body>
		<div class="pagination">
			<a href="/index.php?oldal=1&sorf=abc">1</a>
			<a href="/index.php?oldal=2&sorf=abc">2</a>
			<a href="/index.php?oldal=5&sorf=abc">5</a>
			<a href="/index.php?oldal=10&sorf=abc">10</a>
			<a href="/index.php?oldal=11&sorf=abc">></a>
		</div>
	</body></html>`

	got := parser.ExtractLastPage(strings.NewReader(html))
	if got != 10 {
		t.Errorf("ExtractLastPage() = %d, want 10", got)
	}
}

func TestShowParser_ExtractLastPage_LinkWithoutOldal(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	// Links in pagination div that don't contain "oldal=" should be ignored
	html := `<html><body>
		<div class="pagination">
			<a href="/index.php?page=1">1</a>
			<a href="/index.php?oldal=7&sorf=abc">7</a>
		</div>
	</body></html>`

	got := parser.ExtractLastPage(strings.NewReader(html))
	if got != 7 {
		t.Errorf("ExtractLastPage() = %d, want 7", got)
	}
}

// ---------------------------------------------------------------------------
// extractShowFromGoquery
// ---------------------------------------------------------------------------

func TestShowParser_extractShowFromGoquery(t *testing.T) {
	parser := NewShowParser("https://feliratok.eu")

	tests := []struct {
		name     string
		html     string
		year     int
		wantNil  bool
		wantID   int
		wantName string
	}{
		{
			name:    "link missing href",
			html:    `<a>no href</a>`,
			year:    2025,
			wantNil: true,
		},
		{
			name:    "link with invalid ID",
			html:    `<a href="index.php?sid=abc">link</a>`,
			year:    2025,
			wantNil: true,
		},
		{
			name:    "link with no image",
			html:    `<a href="index.php?sid=123">link</a>`,
			year:    2025,
			wantNil: true,
		},
		{
			name:    "image missing src",
			html:    `<a href="index.php?sid=123"><img></a>`,
			year:    2025,
			wantNil: true,
		},
		{
			name:    "invalid image URL prefix",
			html:    `<a href="index.php?sid=123"><img src="other.php?kep=123"></a>`,
			year:    2025,
			wantNil: true,
		},
		{
			name:     "valid show with fallback name",
			html:     `<div><a href="index.php?sid=456"><img src="sorozat_cat.php?kep=456"></a></div>`,
			year:     2024,
			wantNil:  false,
			wantID:   456,
			wantName: "Show 456",
		},
		{
			name: "valid show with name",
			html: `<table><tr>
				<td><a href="index.php?sid=789"><img src="sorozat_cat.php?kep=789"></a></td>
				<td class="sangol"><div>The Wire</div></td>
			</tr></table>`,
			year:     2002,
			wantNil:  false,
			wantID:   789,
			wantName: "The Wire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}
			link := doc.Find("a").First()
			got := parser.extractShowFromGoquery(link, tt.year)

			if tt.wantNil {
				if got != nil {
					t.Errorf("extractShowFromGoquery() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("extractShowFromGoquery() = nil, want non-nil")
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", got.ID, tt.wantID)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Year != tt.year {
				t.Errorf("Year = %d, want %d", got.Year, tt.year)
			}
		})
	}
}
