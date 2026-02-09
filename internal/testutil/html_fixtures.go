package testutil

import (
	"fmt"
	"strings"
)

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
	SubtitleID       string
	BackgroundColor  string // Default alternates
	Status           string // Optional status like "fordítás alatt (Alice)"
}

// ShowRowOptions contains options for generating a show row
type ShowRowOptions struct {
	ShowID          int
	ShowName        string
	Year            int
	BackgroundColor string
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
		if row.SubtitleID == "" {
			row.SubtitleID = fmt.Sprintf("%d", 1737439811+i)
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

		sb.WriteString(fmt.Sprintf(`
		<tr id="vilagit" style="background-color: %s;">
			<td align="left">
				<a href="index.php?sid=%d"> <img class="kategk" src="img/sorozat_cat/%d.jpg"></a>
			</td>
			<td align="center" class="lang" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				<small><img src="img/flags/%s" alt="%s" border="0" width="30" title="%s"></small>
				%s
			</td>
			<td align="left" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')" style="cursor: pointer;">
					<div class="magyar">%s</div>
					<div class="eredeti">%s</div>%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				%s
			</td>
			<td align="center">
				<a href="/index.php?action=%s&amp;fnev=%s&amp;felirat=%s">
				<img src="img/download.png" border="0" alt="Letöltés" width="20"></a>
			</td>
		</tr>
		
		<tr><td colspan="7" id="adatlap" style="background-color: %s;">
				<div class="0" style="display:none;" id="a_%s">
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
		))
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
		if row.SubtitleID == "" {
			row.SubtitleID = fmt.Sprintf("%d", 1737439811+i)
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

		sb.WriteString(fmt.Sprintf(`
		<tr id="vilagit" style="background-color: %s;">
			<td align="left">
				<a href="index.php?sid=%d"> <img class="kategk" src="img/sorozat_cat/%d.jpg"></a>
			</td>
			<td align="center" class="lang" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				<small><img src="img/flags/%s" alt="%s" border="0" width="30" title="%s"></small>
				%s
			</td>
			<td align="left" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')" style="cursor: pointer;">
					<div class="magyar">%s</div>
					<div class="eredeti">%s</div>%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				%s
			</td>
			<td align="center" onmouseover="this.style.cursor='pointer';" onclick="adatlapnyitas('a_%s')">
				%s
			</td>
			<td align="center">
				<a href="/index.php?action=%s&amp;fnev=%s&amp;felirat=%s">
				<img src="img/download.png" border="0" alt="Letöltés" width="20"></a>
			</td>
		</tr>
		
		<tr><td colspan="7" id="adatlap" style="background-color: %s;">
				<div class="0" style="display:none;" id="a_%s">
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
		))
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
			sb.WriteString(fmt.Sprintf(`
		<tr>
			<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
				%d
			</td>
		</tr>`, show.Year))
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

		sb.WriteString(fmt.Sprintf(`
		<tr style="background-color: %s">
			<td style="padding: 5px;">
				<a href="index.php?sid=%d"><img class="kategk" src="sorozat_cat.php?kep=%d"/></a>
			</td>
			<td class="sangol">
				<div>%s</div>
				<div class="sev"></div>
			</td>
		</tr>`, bgColor, show.ShowID, show.ShowID, show.ShowName))

		rowIndex++
	}

	sb.WriteString(`	</tbody>
</table>
</body>
</html>`)

	return sb.String()
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
		sb.WriteString(fmt.Sprintf(`				<a href="http://www.imdb.com/title/%s/" target="_blank" alt="iMDB" ><img src="img/adatlap/imdb.png" alt="iMDB" /></a><input type="hidden" id="imdb_adatlap" value="%s" />
`, imdbID, imdbID))
	}
	if tvdbID != 0 {
		sb.WriteString(fmt.Sprintf(`				<a href="http://thetvdb.com/?tab=series&id=%d" target="_blank" alt="TheTVDB"><img src="img/adatlap/tvdb.png" alt="TheTVDB"/></a>
`, tvdbID))
	}
	if tvmazeID != 0 {
		sb.WriteString(fmt.Sprintf(`				<a href="http://www.tvmaze.com/shows/%d" target="_blank" alt="TVMaze"><img src="img/adatlap/tvmaze.png" alt="TVMaze"/></a>
`, tvmazeID))
	}
	if traktID != 0 {
		sb.WriteString(fmt.Sprintf(`				<a href="http://trakt.tv/search/tvdb?utf8=%%E2%%9C%%93&query=%d" target="_blank" alt="trakt" ><img src="img/adatlap/trakt.png?v=20250411" alt="trakt" /></a>
`, traktID))
	}

	sb.WriteString(`			</div>
		</div>
	</div>
</body>
</html>`)

	return sb.String()
}

// GeneratePaginationHTML generates HTML with pagination elements
func GeneratePaginationHTML(currentPage, totalPages int, useOldalParam bool) string {
	var sb strings.Builder

	sb.WriteString(`<div class="pagination">
		<span class="current">` + fmt.Sprintf("%d", currentPage) + `</span>
`)

	pageParam := "page"
	if useOldalParam {
		pageParam = "oldal"
	}

	for i := currentPage + 1; i <= totalPages; i++ {
		sb.WriteString(fmt.Sprintf(`		<a href="/index.php?%s=%d">%d</a>
`, pageParam, i, i))
	}

	sb.WriteString(`	</div>`)

	return sb.String()
}
