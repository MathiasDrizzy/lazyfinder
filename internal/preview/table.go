package preview

import (
	"encoding/csv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// maxCellWidth is the widest a column of a table preview gets, in cells; a longer value is cut with an ellipsis.
const maxCellWidth = 40

// maxTableColumns is how many columns a table preview shows. Every row is padded to the widest one, so without a
// cap a single row of many thousands of separators grew the preview of a small file into gigabytes.
const maxTableColumns = 64

// tableSeparators are the extensions shown as a table and the separator of each.
var tableSeparators = map[string]rune{"csv": ',', "tsv": '\t'}

// renderTable lays out a CSV or TSV body as aligned columns with a rule under the header. ok is false when the body
// does not parse, and the caller shows it as text.
func renderTable(body string, sep rune, noColor bool) (lines []string, ok bool) {
	// A spreadsheet's UTF-8 export starts with a byte order mark, which encoding/csv takes for part of the first field.
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\ufeff")))
	r.Comma, r.FieldsPerRecord, r.LazyQuotes = sep, -1, false
	rows, err := r.ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	var widths []int
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], min(ansi.StringWidth(cellText(cell)), maxCellWidth))
		}
	}
	cut := len(widths) > maxTableColumns
	widths = widths[:min(len(widths), maxTableColumns)]
	budget, size := tableOutputBudget(len(body)), 0
	for n, row := range rows {
		line := tableRow(row, widths, cut, n == 0 && !noColor)
		if size += len(line) + 1; size > budget {
			return append(lines, "…"), true
		}
		lines = append(lines, line)
		if n == 0 {
			rule := make([]string, len(widths))
			for i, w := range widths {
				rule[i] = strings.Repeat("─", w)
			}
			lines = append(lines, strings.Join(rule, "─┼─"))
			size += len(lines[len(lines)-1]) + 1
		}
	}
	return lines, true
}

// tableRow lays out one row in the columns of widths; cut adds a last "…" column for those left out.
func tableRow(row []string, widths []int, cut, bold bool) string {
	cells := make([]string, len(widths))
	for i, w := range widths {
		cell := ""
		if i < len(row) {
			cell = ansi.Truncate(cellText(row[i]), w, "…")
		}
		cells[i] = cell + strings.Repeat(" ", max(0, w-ansi.StringWidth(cell)))
	}
	if cut {
		cells = append(cells, "…")
	}
	line := strings.TrimRight(strings.Join(cells, " │ "), " ")
	if bold {
		return "\x1b[1m" + line + "\x1b[22m"
	}
	return line
}

// wholeRows drops the last, cut row of a body the budget truncated.
func wholeRows(body string, truncated bool) string {
	if !truncated {
		return body
	}
	if i := strings.LastIndexByte(body, '\n'); i >= 0 {
		return body[:i+1]
	}
	return ""
}

// tableOutputBudget is the most bytes a table of a body of n bytes lays out: padding the columns may grow the text,
// but only by a bounded factor.
func tableOutputBudget(n int) int { return 4*n + 64<<10 }

// cellText is a cell as one line of valid text: a quoted cell may hold line breaks, and a file any bytes.
func cellText(cell string) string {
	cell = strings.ToValidUTF8(cell, "\uFFFD")
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(cell)
}
