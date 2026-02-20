package testutil

import (
	"fmt"
	"strings"
)

// IntPtr is a helper for creating *int values in tests
func IntPtr(v int) *int {
	return &v
}

// BoolPtr is a helper for creating *bool values in tests
func BoolPtr(v bool) *bool {
	return &v
}

// SubtitleRowOptions contains options for generating a subtitle row
type SubtitleRowOptions struct {
	ShowID           int
	Language         string // "Magyar", "Angol", etc.
	FlagImage        string // "hungary.gif", "uk.gif", etc.
	MagyarTitle      string
	EredetiTitle     string
	Uploader         string
	UploaderBold     bool
	UploadDate       string
	DownloadAction   string
	DownloadFilename string
	SubtitleID       int
	BackgroundColor  string // Default alternates
	Status           string // Optional status like "fordítás alatt (Alice)"
}

// ShowRowOptions contains options for generating a show row
type ShowRowOptions struct {
	ShowID          int
	ShowName        string
	Year            int
	BackgroundColor string
	ImageSrc        string
	IncludeImage    *bool
	IncludeName     *bool
	YearHeaderLabel string
}

// GenerateSubtitleTableHTML generates a proper HTML table structure for subtitle listings
// based on the real feliratok.eu website structure
func GenerateSubtitleTableHTML(rows []SubtitleRowOptions) string {
	var sb strings.Builder

	sb.WriteString(`<html>
<body>
<table width="100%" align="center" border="0" cellspacing="0" cellpadding="5" class="result">
	<thead>
		<tr height="30">
			<th width="124px" style="text-align: center;">Kategória</th>
			<th width="35px">Nyelv</th>
			<th width="50%">
				<div style="float:left; margin-left:70px;">Magyar cím</div>
				<div style="float:right; margin-right:70px;">Külföldi cím</div>
			</th>
			<th style="text-align: center;">Feltöltő</th>
			<th width="65px" nowrap="">Idő</th>
			<th width="35px">Letöltés</th>
		</tr>
	</thead>
	<tbody>
`)

	for i, row := range rows {
		// Alternate background colors if not specified
		bgColor := row.BackgroundColor
		if bgColor == "" {
			if i%2 == 0 {
				bgColor = "#ffffff"
			} else {
				bgColor = "#ecf6fc"
			}
		}

		// Default values
		if row.Language == "" {
			row.Language = "Magyar"
		}
		if row.FlagImage == "" {
			switch row.Language {
			case "Magyar":
				row.FlagImage = "hungary.gif"
			case "Angol":
				row.FlagImage = "uk.gif"
			}
		}
		if row.ShowID == 0 {
			row.ShowID = 2967
		}
		if row.SubtitleID == 0 {
			row.SubtitleID = 1737439811 + i
		}

		uploaderTag := row.Uploader
		if row.UploaderBold {
			uploaderTag = fmt.Sprintf("<b>%s</b>", row.Uploader)
		}

		statusDiv := ""
		if row.Status != "" {
			statusDiv = fmt.Sprintf(`
                        <div><span style="color: rgb(0, 128, 0); font-size: 12px;"><b>%s</b> </span></div>`, row.Status)
		}

		fmt.Fprintf(&sb, `
		<tr id="vilagit" style="background-color: %s;">
			<td align="left">
				<a href="index.php?sid=%d"> <img class="kategk" src="img/sorozat_cat/%d.jpg"></a>
			</td>
			<td align="center" class="lang" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				<small><img src="img/flags/%s" alt="%s" border="0" width="30" title="%s"></small>
				%s
			</td>
			<td align="left" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')" style="cursor: pointer;">
					<div class="magyar">%s</div>
					<div class="eredeti">%s</div>%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				%s
			</td>
			<td align="center">
				<a href="/index.php?action=%s&amp;fnev=%s&amp;felirat=%d">
				<img src="img/download.png" border="0" alt="Letöltés" width="20"></a>
			</td>
		</tr>
		
		<tr><td colspan="7" id="adatlap" style="background-color: %s;">
				<div class="0" style="display:none;" id="a_%d">
					<div id="ajaxloader" style="width: 100%%; height:20px;">
						<img style="margin:0 auto; display: block;" src="img/ajaxloader.gif">
					</div>
				</div>
		</td>		
		</tr>`,
			bgColor,
			row.ShowID, row.ShowID,
			row.SubtitleID,
			row.FlagImage, row.Language, row.Language, row.Language,
			row.SubtitleID,
			row.MagyarTitle, row.EredetiTitle, statusDiv,
			row.SubtitleID,
			uploaderTag,
			row.SubtitleID,
			row.UploadDate,
			row.DownloadAction, row.DownloadFilename, row.SubtitleID,
			bgColor,
			row.SubtitleID,
		)
	}

	sb.WriteString(`	</tbody>
</table>
</body>
</html>`)

	return sb.String()
}

// GenerateSubtitleTableHTMLWithPagination generates HTML with pagination elements included
// This avoids brittle string manipulation by building the complete HTML structure
func GenerateSubtitleTableHTMLWithPagination(rows []SubtitleRowOptions, currentPage, totalPages int, useOldalParam bool) string {
	var sb strings.Builder

	sb.WriteString(`<html>
<body>
<table width="100%" align="center" border="0" cellspacing="0" cellpadding="5" class="result">
	<thead>
		<tr height="30">
			<th width="124px" style="text-align: center;">Kategória</th>
			<th width="35px">Nyelv</th>
			<th width="50%">
				<div style="float:left; margin-left:70px;">Magyar cím</div>
				<div style="float:right; margin-right:70px;">Külföldi cím</div>
			</th>
			<th style="text-align: center;">Feltöltő</th>
			<th width="65px" nowrap="">Idő</th>
			<th width="35px">Letöltés</th>
		</tr>
	</thead>
	<tbody>
`)

	for i, row := range rows {
		// Alternate background colors if not specified
		bgColor := row.BackgroundColor
		if bgColor == "" {
			if i%2 == 0 {
				bgColor = "#ffffff"
			} else {
				bgColor = "#ecf6fc"
			}
		}

		// Default values
		if row.Language == "" {
			row.Language = "Magyar"
		}
		if row.FlagImage == "" {
			switch row.Language {
			case "Magyar":
				row.FlagImage = "hungary.gif"
			case "Angol":
				row.FlagImage = "uk.gif"
			}
		}
		if row.ShowID == 0 {
			row.ShowID = 2967
		}
		if row.SubtitleID == 0 {
			row.SubtitleID = 1737439811 + i
		}

		uploaderTag := row.Uploader
		if row.UploaderBold {
			uploaderTag = fmt.Sprintf("<b>%s</b>", row.Uploader)
		}

		statusDiv := ""
		if row.Status != "" {
			statusDiv = fmt.Sprintf(`
                        <div><span style="color: rgb(0, 128, 0); font-size: 12px;"><b>%s</b> </span></div>`, row.Status)
		}

		fmt.Fprintf(&sb, `
		<tr id="vilagit" style="background-color: %s;">
			<td align="left">
				<a href="index.php?sid=%d"> <img class="kategk" src="img/sorozat_cat/%d.jpg"></a>
			</td>
			<td align="center" class="lang" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				<small><img src="img/flags/%s" alt="%s" border="0" width="30" title="%s"></small>
				%s
			</td>
			<td align="left" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')" style="cursor: pointer;">
					<div class="magyar">%s</div>
					<div class="eredeti">%s</div>%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%d')">
				%s
			</td>
			<td align="center">
				<a href="/index.php?action=%s&amp;fnev=%s&amp;felirat=%d">
				<img src="img/download.png" border="0" alt="Letöltés" width="20"></a>
			</td>
		</tr>
		
		<tr><td colspan="7" id="adatlap" style="background-color: %s;">
				<div class="0" style="display:none;" id="a_%d">
					<div id="ajaxloader" style="width: 100%%; height:20px;">
						<img style="margin:0 auto; display: block;" src="img/ajaxloader.gif">
					</div>
				</div>
		</td>		
		</tr>`,
			bgColor,
			row.ShowID, row.ShowID,
			row.SubtitleID,
			row.FlagImage, row.Language, row.Language, row.Language,
			row.SubtitleID,
			row.MagyarTitle, row.EredetiTitle, statusDiv,
			row.SubtitleID,
			uploaderTag,
			row.SubtitleID,
			row.UploadDate,
			row.DownloadAction, row.DownloadFilename, row.SubtitleID,
			bgColor,
			row.SubtitleID,
		)
	}

	sb.WriteString(`	</tbody>
</table>
`)

	// Add pagination before closing body tag
	sb.WriteString(GeneratePaginationHTML(currentPage, totalPages, useOldalParam))

	sb.WriteString(`
</body>
</html>`)

	return sb.String()
}

// GenerateShowTableHTML generates a proper HTML table structure for show listings
// based on the real feliratok.eu website structure
func GenerateShowTableHTML(shows []ShowRowOptions) string {
	var sb strings.Builder

	sb.WriteString(`<html>
<body>
<table>
	<tbody>
`)

	currentYear := 0
	rowIndex := 0

	for _, show := range shows {
		// Add year header if year changed
		if show.Year != currentYear {
			currentYear = show.Year
			yearLabel := fmt.Sprintf("%d", show.Year)
			if show.YearHeaderLabel != "" {
				yearLabel = show.YearHeaderLabel
			}
			fmt.Fprintf(&sb, `
		<tr>
			<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
				%s
			</td>
		</tr>`, yearLabel)
		}

		// Alternate background colors if not specified
		bgColor := show.BackgroundColor
		if bgColor == "" {
			if rowIndex%2 == 0 {
				bgColor = "#ffffff"
			} else {
				bgColor = "#ecf6fc"
			}
		}

		includeImage := true
		if show.IncludeImage != nil {
			includeImage = *show.IncludeImage
		}

		includeName := true
		if show.IncludeName != nil {
			includeName = *show.IncludeName
		}

		imageSrc := show.ImageSrc
		if imageSrc == "" {
			imageSrc = fmt.Sprintf("sorozat_cat.php?kep=%d", show.ShowID)
		}

		imageHTML := fmt.Sprintf(`<img class="kategk" src="%s"/>`, imageSrc)
		if !includeImage {
			imageHTML = `<img class="kategk"/>`
		}

		nameHTML := fmt.Sprintf(`<div>%s</div>
				<div class="sev"></div>`, show.ShowName)
		if !includeName {
			nameHTML = `<div class="sev"></div>`
		}

		fmt.Fprintf(&sb, `
		<tr style="background-color: %s">
			<td style="padding: 5px;">
				<a href="index.php?sid=%d">%s</a>
			</td>
			<td class="sangol">
				%s
			</td>
		</tr>`, bgColor, show.ShowID, imageHTML, nameHTML)

		rowIndex++
	}

	sb.WriteString(`	</tbody>
</table>
</body>
</html>`)

	return sb.String()
}

// GenerateShowTableHTMLMultiColumn generates HTML with multiple shows per row
// This matches the actual website structure for special show listing pages
// where shows are displayed in a grid layout (typically 2 columns)
func GenerateShowTableHTMLMultiColumn(shows []ShowRowOptions, columnsPerRow int) string {
	if columnsPerRow < 1 {
		columnsPerRow = 2 // Default to 2 columns
	}

	var sb strings.Builder

	sb.WriteString(`<html>
<body>
<table cellpadding="0" cellspacing="0" border="0" align="center" style="width: 100%;">
	<tbody>
`)

	currentYear := 0
	rowIndex := 0

	for i := 0; i < len(shows); i += columnsPerRow {
		// Check if we need a year header
		if shows[i].Year != currentYear {
			currentYear = shows[i].Year
			yearLabel := fmt.Sprintf("%d", shows[i].Year)
			if shows[i].YearHeaderLabel != "" {
				yearLabel = shows[i].YearHeaderLabel
			}
			fmt.Fprintf(&sb, `
		<tr>
			<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
				%s
			</td>
		</tr>`, yearLabel)
		}

		// Determine row background color
		bgColor := shows[i].BackgroundColor
		if bgColor == "" {
			if rowIndex%2 == 0 {
				bgColor = "#ECF6FC"
			} else {
				bgColor = "#FFFFFF"
			}
		}

		fmt.Fprintf(&sb, `
		<tr style="background-color: %s">`, bgColor)

		// Add shows for this row (up to columnsPerRow)
		for j := 0; j < columnsPerRow && i+j < len(shows); j++ {
			show := shows[i+j]

			includeImage := true
			if show.IncludeImage != nil {
				includeImage = *show.IncludeImage
			}

			includeName := true
			if show.IncludeName != nil {
				includeName = *show.IncludeName
			}

			imageSrc := show.ImageSrc
			if imageSrc == "" {
				imageSrc = fmt.Sprintf("sorozat_cat.php?kep=%d", show.ShowID)
			}

			imageHTML := fmt.Sprintf(`<img class="kategk" src="%s"/>`, imageSrc)
			if !includeImage {
				imageHTML = `<img class="kategk"/>`
			}

			nameHTML := fmt.Sprintf(`<div>%s</div>
				<div class="sev"></div>`, show.ShowName)
			if !includeName {
				nameHTML = `<div class="sev"></div>`
			}

			fmt.Fprintf(&sb, `
			<td style="padding: 5px;">
				<a href="index.php?sid=%d">%s</a>
			</td>
			<td class="sangol">
				%s
			</td>`, show.ShowID, imageHTML, nameHTML)
		}

		sb.WriteString(`
		</tr>`)
		rowIndex++
	}

	sb.WriteString(`	</tbody>
</table>
</body>
</html>`)

	return sb.String()
}

// GenerateShowTableHTMLWithPagination generates proper HTML with show listings and pagination
// based on the real feliratok.eu website structure
func GenerateShowTableHTMLWithPagination(shows []ShowRowOptions, currentPage, totalPages int, useOldalParam bool) string {
	var sb strings.Builder

	sb.WriteString(`<html>
<body>
<table>
	<tbody>
`)

	currentYear := 0
	rowIndex := 0

	for _, show := range shows {
		// Add year header if year changed
		if show.Year != currentYear {
			currentYear = show.Year
			yearLabel := fmt.Sprintf("%d", show.Year)
			if show.YearHeaderLabel != "" {
				yearLabel = show.YearHeaderLabel
			}
			fmt.Fprintf(&sb, `
		<tr>
			<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
				%s
			</td>
		</tr>`, yearLabel)
		}

		// Alternate background colors if not specified
		bgColor := show.BackgroundColor
		if bgColor == "" {
			if rowIndex%2 == 0 {
				bgColor = "#ffffff"
			} else {
				bgColor = "#ecf6fc"
			}
		}

		includeImage := true
		if show.IncludeImage != nil {
			includeImage = *show.IncludeImage
		}

		includeName := true
		if show.IncludeName != nil {
			includeName = *show.IncludeName
		}

		imageSrc := show.ImageSrc
		if imageSrc == "" {
			imageSrc = fmt.Sprintf("sorozat_cat.php?kep=%d", show.ShowID)
		}

		imageHTML := fmt.Sprintf(`<img class="kategk" src="%s"/>`, imageSrc)
		if !includeImage {
			imageHTML = `<img class="kategk"/>`
		}

		nameHTML := fmt.Sprintf(`<div>%s</div>
				<div class="sev"></div>`, show.ShowName)
		if !includeName {
			nameHTML = `<div class="sev"></div>`
		}

		fmt.Fprintf(&sb, `
		<tr style="background-color: %s">
			<td style="padding: 5px;">
				<a href="index.php?sid=%d">%s</a>
			</td>
			<td class="sangol">
				%s
			</td>
		</tr>`, bgColor, show.ShowID, imageHTML, nameHTML)

		rowIndex++
	}

	sb.WriteString(`	</tbody>
</table>
`)

	// Add pagination before closing body tag
	sb.WriteString(GeneratePaginationHTML(currentPage, totalPages, useOldalParam))

	sb.WriteString(`
</body>
</html>`)

	return sb.String()
}

// GenerateEmptyHTML returns a minimal HTML document with an empty body.
func GenerateEmptyHTML() string {
	return `<html><body></body></html>`
}

// GenerateInvalidShowTableHTML returns HTML that does not match the expected show table structure.
func GenerateInvalidShowTableHTML() string {
	return GenerateHTMLWithBody(`<table><tr><td>Invalid structure</td></tr></table>`)
}

// GenerateInvalidThirdPartyHTML returns HTML without third-party ID links.
func GenerateInvalidThirdPartyHTML() string {
	return GenerateHTMLWithBody(`<div>Invalid structure</div>`)
}

// GenerateHTMLWithBody wraps custom body content in a standard HTML shell.
func GenerateHTMLWithBody(bodyHTML string) string {
	return `<html><body>` + bodyHTML + `</body></html>`
}

// GenerateThirdPartyIDHTML generates a proper HTML structure for third-party ID details page
// based on the real feliratok.eu episode detail page structure
func GenerateThirdPartyIDHTML(imdbID string, tvdbID, tvmazeID, traktID int) string {
	var sb strings.Builder

	sb.WriteString(`<html>
<body>
	<div class="adatlapTabla">
		<div class="adatlapKep">
			<img src="img/sorozat_posterx/10665.jpg" width="124" height="182">
		</div>
		<div class="adatlapAdat">
			<div class="adatlapRow">
				<span>Fájlnév:</span>
				<span>Show.S01E01.srt</span>
			</div>
			<div class="adatlapRow">
				<span>Feltöltő:</span>
				<span>TestUser</span>
			</div>
			<div class="adatlapRow paddingb5">
				<span>Megjegyzés:</span>
				<span class="megjegyzes"></span>
			</div>
			<div class="adatlapRow">
`)

	if imdbID != "" {
		fmt.Fprintf(&sb, `				<a href="http://www.imdb.com/title/%s/" target="_blank" alt="iMDB" ><img src="img/adatlap/imdb.png" alt="iMDB" /></a><input type="hidden" id="imdb_adatlap" value="%s" />
`, imdbID, imdbID)
	}
	if tvdbID != 0 {
		fmt.Fprintf(&sb, `				<a href="http://thetvdb.com/?tab=series&id=%d" target="_blank" alt="TheTVDB"><img src="img/adatlap/tvdb.png" alt="TheTVDB"/></a>
`, tvdbID)
	}
	if tvmazeID != 0 {
		fmt.Fprintf(&sb, `				<a href="http://www.tvmaze.com/shows/%d" target="_blank" alt="TVMaze"><img src="img/adatlap/tvmaze.png" alt="TVMaze"/></a>
`, tvmazeID)
	}
	if traktID != 0 {
		fmt.Fprintf(&sb, `				<a href="http://trakt.tv/search/tvdb?utf8=%%E2%%9C%%93&query=%d" target="_blank" alt="trakt" ><img src="img/adatlap/trakt.png?v=20250411" alt="trakt" /></a>
`, traktID)
	}

	sb.WriteString(`			</div>
		</div>
	</div>
</body>
</html>`)

	return sb.String()
}

// GeneratePaginationHTML generates HTML with pagination elements matching the
// real feliratok.eu structure: div.pagination with <span class="current"> for
// the current page and <a href="...oldal=N&sorf=..."> links for other pages.
func GeneratePaginationHTML(currentPage, totalPages int, useOldalParam bool) string {
	if totalPages <= 1 {
		return ""
	}

	var sb strings.Builder

	pageParam := "page"
	sorfParam := "sorf=nem-all-forditas-alatt"
	if useOldalParam {
		pageParam = "oldal"
	}

	sb.WriteString(`<div class="tableTitle">
	<div class="pagination">
`)

	// Generate realistic pagination: show first few pages, ..., last few pages
	// Similar to the real site: 1 2 3 4 5 6 7 ... 41 42 >
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			fmt.Fprintf(&sb, `<span class="current">%d</span>`, i)
		} else {
			fmt.Fprintf(&sb, `<a href="/index.php?%s=%d&%s">%d</a>`, pageParam, i, sorfParam, i)
		}
	}

	// Add ">" next page link if not on last page
	if currentPage < totalPages {
		fmt.Fprintf(&sb, `<a href="/index.php?%s=%d&%s">></a>`, pageParam, currentPage+1, sorfParam)
	}

	sb.WriteString(`	</div>
</div>`)

	return sb.String()
}
