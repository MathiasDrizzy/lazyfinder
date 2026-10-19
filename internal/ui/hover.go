package ui

import (
	"charm.land/lipgloss/v2"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

// hoverState is where the pointer rests; it is off after a key press and with --no-mouse.
type hoverState struct {
	x, y int
	on   bool
	at   time.Time // the last time the pointer moved over the screen
}

type hoverRect struct{ x, y, w, h int }

// hoverBg is the soft background under the pointer: between the stripes and the selection, so it reads as neither.
func hoverBg(p theme.Palette) color.Color { return theme.Mix(p.Base, p.Surface1, 0.6) }

// chipHoverBg is the box the pointer lights on a button, lighter than the hover of a row, as the keycap of the herdr
// sidebar; chipActiveBg is the chip of the active tab, a step lighter still. Both come from the theme (S3).
func chipHoverBg(p theme.Palette) color.Color  { return p.Surface1 }
func chipActiveBg(p theme.Palette) color.Color { return theme.Mix(p.Surface1, p.Overlay0, 0.4) }

// activeChip draws the chip of the active tab on its background, bold.
func (m *Model) activeChip(s string) string {
	p, ok := theme.Lookup(m.deps.Cfg.Theme)
	if !ok || m.deps.NoColor {
		return m.styles.selected(s)
	}
	bg := chipActiveBg(p)
	return lipgloss.NewStyle().Background(bg).Foreground(theme.Readable(p.Text, bg, minReadable)).Bold(true).Render(s)
}

// onChip says whether the pointer is on a button, whose box is lit with chipHoverBg instead of the row hover.
func (m *Model) onChip() bool {
	switch m.hit(m.hover.x, m.hover.y).kind {
	case zoneAction, zoneSettings, zoneMode, zoneProjectTab, zoneScmBtn, zoneBack, zoneForward:
		return true
	}
	return false
}

// hoverRect is the area of whatever is clickable under the pointer, if anything.
func (m *Model) hoverRect() (hoverRect, bool) {
	if !m.mouse || !m.hover.on || m.tooSmallNow() {
		return hoverRect{}, false
	}
	x, y := m.hover.x, m.hover.y
	switch {
	case m.set.open:
		mx, my, bx := m.settingsBox()
		_, rows, _ := m.settingsView()
		return popupRow(x, y, mx, my, bx, rows)
	case m.menu.open:
		mx, my, bx := m.menuBox()
		return popupRow(x, y, mx, my, bx, len(m.menu.items))
	case m.ctx.open:
		mx, my, bx := m.ctxBox()
		return popupRow(x, y, mx, my, bx, len(m.ctx.items))
	case m.diff.open:
		return hoverRect{}, false
	case m.gm.open:
		mx, my, bx := m.gitMenuBox()
		if !inBox(x, y, mx, my, bx) || y-my-3 < 0 || y-my-3 >= m.gmRows() {
			return hoverRect{}, false
		}
		return hoverRect{mx + 1, y, boxWidth(bx) - 2, 1}, true
	case m.picker.open:
		mx, my, bx := m.pickerBox()
		return popupRow(x, y, mx, my, bx, m.pickerRows())
	case m.help.open, m.info.open, m.pv.open:
		return hoverRect{}, false
	}
	return m.zoneRectAt(m.hit(x, y), x, y)
}

// zoneRect is the area a button of the top rows takes: what its hover paints and its click hits.
func (m *Model) zoneRect(z zone) (hoverRect, bool) {
	return m.zoneRectAt(z, 0, 0)
}

// zoneRectAt is the area of the zone z under the pointer at x, y.
func (m *Model) zoneRectAt(z zone, x, y int) (hoverRect, bool) {
	switch z.kind {
	case zoneRow:
		if z.idx == m.sel {
			return hoverRect{}, false
		}
		return hoverRect{m.listX(), m.firstRow() + (z.idx-m.top)*m.rowH(), m.listW(), m.rowH()}, true
	case zoneScmRow:
		if z.idx == m.scm.sel {
			return hoverRect{}, false
		}
		return hoverRect{m.listX(), y, m.listW(), 1}, true
	case zoneScmBtn:
		for _, b := range m.scmButtons(z.idx) {
			if b.act == z.act {
				return hoverRect{b.x, y, b.w, 1}, true
			}
		}
	case zoneBranch:
		if y != 0 {
			return hoverRect{0, y, ansi.StringWidth(strings.TrimRight(ansi.Strip(m.projectFooter()), " ")), 1}, true
		}
		if bx, bw, ok := m.branchSpan(); ok {
			return hoverRect{bx, 0, bw, 1}, true
		}
	case zoneToggle:
		return hoverRect{m.listX(), m.firstRow() + (z.idx-m.top)*m.rowH(), m.listW(), m.rowH()}, true
	case zonePlace:
		if z.idx == m.sideSel && m.focus == focusSidebar {
			return hoverRect{}, false
		}
		return hoverRect{1, y, m.sidebarW() - 2, 1}, true
	case zoneHeader:
		for _, s := range m.columns() {
			if z.col == s.col {
				return hoverRect{m.listX() + s.x, m.headerRow(), s.w, 1}, true
			}
		}
	case zoneProjectTab:
		if projectTab(z.idx) == m.activeTab() {
			return hoverRect{}, false // the active tab keeps its selected chip
		}
		_, spans := m.projectTabsLine()
		s := spans[z.idx]
		return hoverRect{s.x, 1, s.w, 1}, true // the tab bar draws the rows above and below
	case zoneSettings:
		return hoverRect{m.gearX(), m.toolbarRow(), m.gearW(), 1}, true
	case zoneMode:
		return hoverRect{m.modeX(), m.toolbarRow(), m.chipW(), 1}, true
	case zoneSearch:
		if y == 0 {
			return hoverRect{m.fieldX(), 0, m.fieldW(), 1}, true
		}
	case zoneAction:
		for _, a := range m.actionSpans() {
			if a.idx == z.idx {
				return hoverRect{a.x, a.y, a.w, 1}, true
			}
		}
	case zoneBack:
		return hoverRect{0, 0, 3, 1}, true
	case zoneForward:
		return hoverRect{3, 0, 3, 1}, true
	case zoneCrumb:
		for _, c := range m.crumbs() {
			if c.path == z.path && x >= c.x && x < c.x+c.w {
				return hoverRect{c.x, y, c.w, 1}, true
			}
		}
	case zoneChip:
		_, spans := m.filterBar()
		s := spans[z.idx]
		return hoverRect{s.x, y, s.w, 1}, true
	case zoneHint:
		_, spans := m.footerHints()
		for _, s := range spans {
			if x >= s.x && x < s.x+s.w {
				return hoverRect{s.x, y, s.w, 1}, true
			}
		}
	}
	return hoverRect{}, false
}

// popupRow is the row of a popup under the pointer, inside its frame.
func popupRow(x, y, bx, by int, box []string, rows int) (hoverRect, bool) {
	if !inBox(x, y, bx, by, box) || y-by-1 < 0 || y-by-1 >= rows {
		return hoverRect{}, false
	}
	return hoverRect{bx + 1, y, boxWidth(box) - 2, 1}, true
}

// withHover paints the hovered area with the hover background, keeping the colors of the text on it.
func (m *Model) withHover(lines []string) []string {
	r, ok := m.hoverRect()
	if !ok {
		return lines
	}
	r.w = min(r.w, m.w-r.x) // a chip cut off by a narrow terminal still has its full span
	if r.w <= 0 {
		return lines
	}
	return m.paintHover(lines, r)
}

// paintHover paints the area r of lines with the hover background.
func (m *Model) paintHover(lines []string, r hoverRect) []string {
	seq := "\x1b[4m" // without a palette the pointer underlines what it is on
	var bg color.Color
	if p, found := theme.Lookup(m.deps.Cfg.Theme); found {
		bg = hoverBg(p)
		if m.onChip() {
			bg = chipHoverBg(p)
		}
		seq = theme.BackgroundSeq(bg)
	}
	if p, found := theme.Lookup(m.deps.Cfg.Theme); found {
		if g, ok := m.chipGlyphUnderPointer(); ok {
			if box, ok := nerdChipBox(g); ok {
				return paintChip(lines, r, box, chipHoverBg(p))
			}
		}
	}
	out := make([]string, len(lines))
	copy(out, lines)
	for y := r.y; y < r.y+r.h && y < len(out); y++ {
		if y < 0 {
			continue
		}
		line := out[y]
		left := ansi.Truncate(line, r.x, "")
		left += strings.Repeat(" ", max(0, r.x-ansi.StringWidth(left)))
		mid := ansi.Cut(line, r.x, r.x+r.w)
		mid += strings.Repeat(" ", max(0, r.w-ansi.StringWidth(mid)))
		right := ansi.TruncateLeft(line, r.x+r.w, "")
		painted := left + "\x1b[0m" + seq + dropBackgrounds(mid, seq, bg) + "\x1b[0m" + right
		// a wide glyph cut by the edge of the area changes the width by a cell either way
		want := ansi.StringWidth(line)
		if ansi.StringWidth(painted) > want {
			painted = ansi.Truncate(painted, want, "")
		}
		if got := ansi.StringWidth(painted); got < want {
			painted += strings.Repeat(" ", want-got)
		}
		out[y] = painted
	}
	return out
}

// dropBackgrounds removes the background colors set inside s and puts seq after every reset, so the hover color
// reaches the whole area whatever stripe or selection was under it.
func dropBackgrounds(s, seq string, bg color.Color) string {
	var b strings.Builder
	for len(s) > 0 {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		s = s[i:]
		end := strings.IndexByte(s, 'm')
		if end < 0 || strings.ContainsAny(s[2:end], "ABCDEFGHJKSTfhlsu") {
			b.WriteString(s[:2])
			s = s[2:]
			continue
		}
		params := s[2:end]
		seqText := s[:end+1]
		s = s[end+1:]
		if params == "" || params == "0" {
			b.WriteString(seqText + seq)
			continue
		}
		if kept := readableOn(stripBackground(params), bg); kept != "" {
			b.WriteString("\x1b[" + kept + "m")
		}
	}
	return b.String()
}

// stripBackground drops the 40-49, 100-107 and 48;... parameters of an SGR sequence.
func stripBackground(params string) string {
	parts := strings.Split(params, ";")
	var keep []string
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		switch {
		case err != nil:
			keep = append(keep, parts[i])
		case n == 38:
			keep = append(keep, parts[i])
			if i+1 < len(parts) && parts[i+1] == "2" {
				keep = append(keep, parts[i+1:min(len(parts), i+5)]...)
				i += 4
			} else if i+1 < len(parts) && parts[i+1] == "5" {
				keep = append(keep, parts[i+1:min(len(parts), i+3)]...)
				i += 2
			}
		case n == 48:
			if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			} else {
				i += 2
			}
		case (n >= 40 && n <= 49) || (n >= 100 && n <= 107):
		default:
			keep = append(keep, parts[i])
		}
	}
	return strings.Join(keep, ";")
}

// readableOn lifts every true color text color of the SGR parameters params that would read under minReadable on bg,
// the background the text is now drawn on: a folder blue on the selection of a dark theme. A nil bg keeps them.
func readableOn(params string, bg color.Color) string {
	if bg == nil {
		return params
	}
	parts := strings.Split(params, ";")
	for i := 0; i+4 < len(parts); i++ {
		if parts[i] != "38" || parts[i+1] != "2" {
			continue
		}
		var rgb [3]uint8
		ok := true
		for j := range rgb {
			n, err := strconv.Atoi(parts[i+2+j])
			ok = ok && err == nil && n >= 0 && n <= 255
			rgb[j] = uint8(n) //nolint:gosec // checked just above
		}
		if ok {
			r, g, b, _ := theme.Readable(color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, bg, minReadable).RGBA()
			parts[i+2], parts[i+3], parts[i+4] = strconv.Itoa(int(r>>8)), strconv.Itoa(int(g>>8)), strconv.Itoa(int(b>>8))
		}
		i += 4
	}
	return strings.Join(parts, ";")
}
