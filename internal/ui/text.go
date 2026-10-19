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
	s = showControls(s)
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "…")
	}
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// showControls turns the control characters of a cell's text into their pictures (a line break in a file name shows
// as ␊), so a name cannot break the row or move the cursor. ESC stays: the cells carry lazyfinder's own colors, and
// View drops any sequence a name could add with it.
func showControls(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\x1b':
			return r
		case r < 0x20:
			return 0x2400 + r
		case r == 0x7f:
			return '␡'
		case r >= 0x80 && r < 0xa0:
			return '�'
		}
		return r
	}, s)
}

// safeFrame drops from the frame every escape sequence other than a color or a text style (SGR), and every control
// character other than the line break. lazyfinder draws nothing else in the frame (images go out on their own with
// tea.Raw), so any other sequence came from a file name or a file's text: a string sequence (an OSC 8 link, an OSC 52
// clipboard write), a CSI that clears the screen, moves the cursor or leaves the alternate screen, or a two-byte one
// such as ESC c (reset) or ESC ( 0 (line drawing characters).
func safeFrame(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\x1b' {
			end, keep := escapeEnd(s, i)
			if keep {
				b.WriteString(s[i : end+1])
			}
			i = end
			continue
		}
		if c < 0x20 && c != '\n' || c == 0x7f {
			continue
		}
		if c == 0xc2 && i+1 < len(s) && s[i+1] >= 0x80 && s[i+1] < 0xa0 { // a C1 control in UTF-8
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// escapeEnd returns the index of the last byte of the escape sequence that starts at i, and whether it is a color or
// a text style (CSI with only digits, ; and : before its final m).
func escapeEnd(s string, i int) (int, bool) {
	if i+1 >= len(s) {
		return i, false
	}
	switch next := s[i+1]; {
	case strings.IndexByte("]P_^X", next) >= 0:
		return stringSequenceEnd(s, i+2), false
	case next == '[':
		j, sgr := i+2, true
		for ; j < len(s) && s[j] >= 0x20 && s[j] <= 0x3f; j++ { // parameters and intermediates
			sgr = sgr && (s[j] >= '0' && s[j] <= '9' || s[j] == ';' || s[j] == ':')
		}
		if j >= len(s) || s[j] < 0x40 || s[j] > 0x7e { // cut short: drop what there is
			return j - 1, false
		}
		return j, sgr && s[j] == 'm'
	default:
		j := i + 1
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f { // intermediates, as the ( of ESC ( 0
			j++
		}
		if j >= len(s) || s[j] < 0x30 || s[j] > 0x7e {
			return j - 1, false
		}
		return j, false
	}
}

// stringSequenceEnd returns the index of the last byte of a string sequence whose body starts at i: its BEL or the
// backslash of its ESC \. A sequence without an end runs to the end of the line.
func stringSequenceEnd(s string, i int) int {
	for ; i < len(s); i++ {
		switch {
		case s[i] == '\a':
			return i
		case s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\':
			return i + 1
		case s[i] == '\n':
			return i - 1
		}
	}
	return len(s) - 1
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
		if want := ansi.StringWidth(out[row]) - x - bw; ansi.StringWidth(right) > want && want >= 0 {
			// a wide glyph straddles the edge of the box: drop it whole and fill its other half
			right = " " + ansi.TruncateLeft(out[row], x+bw+1, "")
		}
		out[row] = left + "\x1b[0m" + b + "\x1b[0m" + after + right
	}
	return out
}

// box wraps content lines (each already inner wide) in a rounded popup frame with the title in the top edge.
func (m *Model) box(title string, content []string, inner int) []string {
	inner = max(1, min(inner, m.w-4)) // a popup never outgrows the terminal, however small the sidebar mode is
	title = " " + ansi.Truncate(title, inner-4, "…") + " "
	dashes := strings.Repeat("─", max(0, inner+1-ansi.StringWidth(title)))
	top := m.styles.popupBorder("╭─") + m.styles.popupTitle(title) + m.styles.popupBorder(dashes+"╮")
	out := []string{top}
	for _, l := range content {
		out = append(out, m.styles.popupBorder("│")+" "+fit(l, inner)+" "+m.styles.popupBorder("│"))
	}
	bottom := "╰" + strings.Repeat("─", inner+2) + "╯"
	if inner+2 >= len(escHint)+4 { // like lazymark, the way out is written on the frame
		bottom = "╰" + strings.Repeat("─", inner+2-len(escHint)-1) + escHint + "─╯"
	}
	return append(out, m.styles.popupBorder(bottom))
}

// escHint is written on the bottom border of every popup: the popups close with Esc and only with it.
const escHint = "[Esc]"

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
