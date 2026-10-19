package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

// Mathias, 2026-10-14: "no se puede centrar el icono con el recuadro gris? … el icono está más a la derecha y la
// izquierda está más grande". Ghostty draws a Nerd Font glyph over its cell and part of the space after it, so its
// ink is wider than one cell and narrower than two, and no box of whole or half cells is centered on it. nerdInkPx
// is the ink width of each glyph of a button, measured in Ghostty with 17-pixel cells (JetBrainsMono Nerd Font).
// chipMargins models the two margins from that width; the model was checked against Ghostty screenshots of every
// button in both modes (order 07, revision 3: margins at most 2 pixels apart).
var nerdInkPx = map[string]int{
	"\uea7f": 21, // new file
	"\uea80": 25, // new folder
	"\ueac5": 21, // collapse all
	"\uf013": 23, // gear
	"\uf0db": 25, // mode
}

// inkLeftPx is where the ink of every one of those glyphs starts, in pixels from the left edge of its cell.
const inkLeftPx = 1

// inkCellPx is the cell width the ink widths were measured with.
const inkCellPx = 17

// leftEighths are the blocks that fill the left n eighths of a cell, n from 1 to 7.
var leftEighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// rightEighths are the blocks that fill the right n eighths of a cell, n from 1 to 7 (the ones from 2 to 7, but 4,
// are in Symbols for Legacy Computing; Ghostty draws them itself).
var rightEighths = []string{"", "▕", "🮇", "🮈", "▐", "🮉", "🮊", "🮋"}

// chipBoxPx is the width every lit box aims at, two cells and a half, so the boxes look alike whatever the ink of
// their glyph (Mathias, 2026-10-14: "revisa también los tamaños de los recuadros gris … este es mucho más grande que
// los otros").
const chipBoxPx = 2.5 * inkCellPx

// chipBox is how the lit box of a chip x-1, x (the glyph), x+1 is drawn, in eighths of a cell: fill eighths (1 to 8) of
// the cell before the glyph are lit, on its right side, with a block in the box color (never with reversed colors:
// with the terminal's own background the unlit part showed as a white strip, Mathias 2026-10-14), and tail eighths
// of the cell after the space that follows the glyph. That space is lit whole: any other character there makes
// Ghostty draw the glyph in one cell, smaller than its neighbours.
type chipBox struct{ fill, tail int }

// nerdChipBox picks, for a measured glyph, among the boxes whose margins on the two sides of the ink differ by 2
// pixels at most, the one closest to chipBoxPx wide.
func nerdChipBox(g string) (chipBox, bool) {
	ink, ok := nerdInkPx[g]
	if !ok {
		return chipBox{}, false
	}
	eighth := float64(inkCellPx) / 8
	best, bestScore := chipBox{8, 0}, -1.0
	for fill := 1; fill <= 8; fill++ {
		for tail := 0; tail < 8; tail++ {
			left := inkLeftPx + float64(fill)*eighth
			right := float64(2*inkCellPx-inkLeftPx-ink) + float64(tail)*eighth
			if abs(left-right) > 2 {
				continue
			}
			score := abs(float64(fill+16+tail)*eighth - chipBoxPx)
			if bestScore < 0 || score < bestScore {
				best, bestScore = chipBox{fill, tail}, score
			}
		}
	}
	return best, true
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// chipGlyphUnderPointer is the glyph of the button of the top rows under the pointer, in the Nerd Font style.
func (m *Model) chipGlyphUnderPointer() (string, bool) {
	if m.icons.Style != icons.Nerd {
		return "", false
	}
	switch z := m.hit(m.hover.x, m.hover.y); z.kind {
	case zoneAction:
		return m.actionGlyph(z.idx), true
	case zoneSettings:
		return m.gearText(), true
	case zoneMode:
		return m.modeGlyph(), true
	}
	return "", false
}

// paintChip lights the chip r (three cells, the glyph in the middle) as box says: the right
// fill eighths of the cell before the glyph, the glyph's cell and the space after it whole, and the left tail eighths
// of the next cell.
func paintChip(lines []string, r hoverRect, box chipBox, bg color.Color) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	if r.y < 0 || r.y >= len(out) {
		return out
	}
	line := out[r.y]
	want := ansi.StringWidth(line)
	edge := lipgloss.NewStyle().Foreground(bg).Render
	seq := theme.BackgroundSeq(bg)
	from, head := r.x, ""
	if box.fill < 8 {
		from, head = r.x+1, edge(rightEighths[box.fill])
	}
	left := ansi.Truncate(line, r.x, "")
	left += strings.Repeat(" ", max(0, r.x-ansi.StringWidth(left)))
	mid := ansi.Cut(line, from, r.x+r.w)
	mid += strings.Repeat(" ", max(0, r.x+r.w-from-ansi.StringWidth(mid)))
	end, tail := r.x+r.w, ""
	if box.tail > 0 {
		end, tail = end+1, edge(leftEighths[box.tail])
	}
	right := ansi.TruncateLeft(line, end, "")
	painted := left + "\x1b[0m" + head + seq + dropBackgrounds(mid, seq, bg) + "\x1b[0m" + tail + right
	if ansi.StringWidth(painted) > want {
		painted = ansi.Truncate(painted, want, "")
	}
	if got := ansi.StringWidth(painted); got < want {
		painted += strings.Repeat(" ", want-got)
	}
	out[r.y] = painted
	return out
}
