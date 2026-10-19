package preview

import "strings"

// Visible makes the text of a file safe to draw: every control character except the line break and the tab becomes
// its Unicode control picture (ESC shows as ␛, a carriage return as ␍), so a file cannot move the cursor, open a
// link or send the terminal any command. A CRLF line ending stays a plain line break.
func Visible(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		return Picture(r)
	}, strings.ReplaceAll(s, "\r\n", "\n"))
}

// VisibleLine is Visible for text that must stay on one row, such as a file name: the line break and the tab become
// pictures too.
func VisibleLine(s string) string { return strings.Map(Picture, s) }

// Picture is how a character of a file's name or text is drawn: a control character as its Unicode picture, and a
// bidi control, which makes the terminal reorder what follows (a name drawn as invoiceexe.pdf, code that reads other
// than it runs), as �. Any other character is itself.
func Picture(r rune) rune {
	switch {
	case r < 0x20:
		return 0x2400 + r // ␀ … ␟
	case r == 0x7f:
		return '␡'
	case r >= 0x80 && r < 0xa0: // C1 controls: some terminals read U+009B as CSI and U+009D as OSC
		return '�'
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069: // embeddings, overrides and isolates
		return '�'
	}
	return r
}
