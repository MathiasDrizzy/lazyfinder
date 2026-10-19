package preview

import (
	"encoding/csv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// maxCellWidth is the widest a column of a table preview gets, in cells; a longer value is cut with an ellipsis.
const maxCellWidth = 40

// tableSeparators are the extensions shown as a table and the separator of each.
var tableSeparators = map[string]rune{"csv": ',', "tsv": '\t'}

// renderTable lays out a CSV or TSV body as aligned columns with a rule under the header. ok is false when the body
// does not parse, and the caller shows it as text.
func renderTable(body string, sep rune, noColor bool) (lines []string, ok bool) {
	r := csv.NewReader(strings.NewReader(body))
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
			widths[i] = max(widths[i], min(ansi.StringWidth(cell), maxCellWidth))
		}
	}
	for n, row := range rows {
		lines = append(lines, tableRow(row, widths, n == 0 && !noColor))
		if n == 0 {
			rule := make([]string, len(widths))
			for i, w := range widths {
				rule[i] = strings.Repeat("─", w)
			}
			lines = append(lines, strings.Join(rule, "─┼─"))
		}
	}
	return lines, true
}

func tableRow(row []string, widths []int, bold bool) string {
	cells := make([]string, len(widths))
	for i, w := range widths {
		cell := ""
		if i < len(row) {
			cell = ansi.Truncate(strings.ReplaceAll(row[i], "\n", " "), w, "…")
		}
		cells[i] = cell + strings.Repeat(" ", w-ansi.StringWidth(cell))
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
