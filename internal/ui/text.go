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
		out[row] = left + "\x1b[0m" + b + "\x1b[0m" + right
	}
	return out
}

// box wraps content lines (each already inner wide) in a border with a title in the top edge.
func box(title string, content []string, inner int, style func(string) string) []string {
	title = " " + ansi.Truncate(title, inner-4, "…") + " "
	top := "┌─" + title + strings.Repeat("─", max(0, inner+1-ansi.StringWidth(title))) + "┐"
	out := []string{style(top)}
	for _, l := range content {
		out = append(out, style("│")+" "+fit(l, inner)+" "+style("│"))
	}
	return append(out, style("└"+strings.Repeat("─", inner+2)+"┘"))
}
