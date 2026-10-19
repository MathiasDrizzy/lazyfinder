package theme

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
)

var sgrSeq = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// PaintBackground paints with bg every cell of s that has no background of its own: each line starts with it and it
// is set again after every sequence that clears it (a reset or 49) without setting another. So no cell keeps the
// terminal's background: not the gaps between panels, the padding of the lines or what popups draw without one.
//
// With a text color, text that carries none takes it as well, the same way.
func PaintBackground(s string, bg color.Color, text ...color.Color) string {
	set, setFg := BackgroundSeq(bg), ""
	if len(text) > 0 {
		setFg = ForegroundSeq(text[0])
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = sgrSeq.ReplaceAllStringFunc(line, func(seq string) string {
			params := sgrSeq.FindStringSubmatch(seq)[1]
			if clearsBackground(params) {
				seq += set
			}
			if setFg != "" && clearsForeground(params) {
				seq += setFg
			}
			return seq
		})
		lines[i] = set + setFg + line + "\x1b[m"
	}
	return strings.Join(lines, "\n")
}

// clearsBackground says whether the parameters of an SGR leave the background without a color: a reset (empty, or 0
// first) or 49, without setting another background (40-47, 48, 100-107) in the same sequence.
func clearsBackground(params string) bool {
	parts := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(parts) == 0 {
		return true
	}
	clears := false
	for i := 0; i < len(parts); i++ {
		switch p := parts[i]; {
		case p == "38" || p == "58": // text or underline color: its parameters are not SGR codes of their own
			if i+1 < len(parts) && parts[i+1] == "5" {
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			}
		case p == "48":
			return false
		case len(p) == 2 && (p[0] == '4' && p[1] >= '0' && p[1] <= '7'), len(p) == 3 && p[:2] == "10" && p[2] >= '0' && p[2] <= '7':
			return false
		case p == "0" || p == "00" || p == "49":
			clears = true
		}
	}
	return clears
}

// BackgroundSeq is the escape sequence that sets bg as the background color.
func BackgroundSeq(bg color.Color) string {
	r, g, b, _ := bg.RGBA()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}

// ForegroundSeq is the escape sequence that sets c as the text color.
func ForegroundSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}

// clearsForeground says whether the parameters of an SGR leave the text without a color: a reset or 39, without
// setting another one (30-37, 38, 90-97) in the same sequence.
func clearsForeground(params string) bool {
	parts := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(parts) == 0 {
		return true
	}
	clears := false
	for i := 0; i < len(parts); i++ {
		switch p := parts[i]; {
		case p == "48" || p == "58":
			if i+1 < len(parts) && parts[i+1] == "5" {
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			}
		case p == "38":
			return false
		case len(p) == 2 && p[0] == '3' && p[1] >= '0' && p[1] <= '7', len(p) == 2 && p[0] == '9' && p[1] >= '0' && p[1] <= '7':
			return false
		case p == "0" || p == "00" || p == "39":
			clears = true
		}
	}
	return clears
}

// Mix is the color t of the way from a to b (0 is a, 1 is b).
func Mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	mix := func(x, y uint32) uint8 { return uint8((float64(x>>8)*(1-t) + float64(y>>8)*t) + 0.5) }
	return color.RGBA{R: mix(ar, br), G: mix(ag, bg), B: mix(ab, bb), A: 255}
}
