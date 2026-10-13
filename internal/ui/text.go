package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// fit truncates s to w cells (with an ellipsis) and pads it with spaces to exactly w cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "…")
	}
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// overlay draws box (lines of equal width bw) over base lines at column x, row y.
func overlay(base []string, box []string, x, y int) []string {
	return overlayAfter(base, box, x, y, "")
}

// overlayAfter is overlay; after is written right behind the box, to give the rest of the line back its background.
func overlayAfter(base []string, box []string, x, y int, after string) []string {
	out := make([]string, len(base))
	copy(out, base)
	for i, b := range box {
		row := y + i
		if row < 0 || row >= len(out) {
			continue
		}
		bw := ansi.StringWidth(b)
		left := ansi.Truncate(out[row], x, "")
		left += strings.Repeat(" ", max(0, x-ansi.StringWidth(left)))
		right := ansi.TruncateLeft(out[row], x+bw, "")
		out[row] = left + "\x1b[0m" + b + "\x1b[0m" + after + right
	}
	return out
}

// box wraps content lines (each already inner wide) in a rounded popup frame with the title in the top edge.
func (m *Model) box(title string, content []string, inner int) []string {
	title = " " + ansi.Truncate(title, inner-4, "…") + " "
	dashes := strings.Repeat("─", max(0, inner+1-ansi.StringWidth(title)))
	top := m.styles.popupBorder("╭─") + m.styles.popupTitle(title) + m.styles.popupBorder(dashes+"╮")
	out := []string{top}
	for _, l := range content {
		out = append(out, m.styles.popupBorder("│")+" "+fit(l, inner)+" "+m.styles.popupBorder("│"))
	}
	return append(out, m.styles.popupBorder("╰"+strings.Repeat("─", inner+2)+"╯"))
}

// panel draws a rounded panel of exactly w x h cells. lines are the inner rows (already inner wide or shorter); the
// title sits in the top border, the footer (a counter) in the bottom one. The active panel gets the accent border.
func (m *Model) panel(title, footer string, lines []string, w, h int, active bool) []string {
	inner := w - 2
	border, titleStyle := m.styles.border, m.styles.title
	if active {
		border, titleStyle = m.styles.borderActive, m.styles.titleActive
	}
	edge := func(label string, right bool) string {
		if label == "" || inner < 6 {
			return border(strings.Repeat("─", inner))
		}
		label = " " + ansi.Truncate(label, inner-4, "…") + " "
		dashes := border(strings.Repeat("─", inner-1-ansi.StringWidth(label)))
		if right {
			return dashes + titleStyle(label) + border("─")
		}
		return border("─") + titleStyle(label) + dashes
	}
	rows := []string{border("╭") + edge(title, false) + border("╮")}
	for i := 0; i < h-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows = append(rows, border("│")+fit(line, inner)+"\x1b[0m"+border("│"))
	}
	return append(rows, border("╰")+edge(footer, true)+border("╯"))
}
