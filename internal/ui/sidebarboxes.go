package ui

import (
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/charmbracelet/x/ansi"
)

// minBoxH is the least a box can be: its two borders and one line.
const minBoxH = 3

// sbBox is one framed section of the sidebar: Favorites or Locations, each with its own border and title, like
// Folders, Tasks and Tags in lazymark.
type sbBox struct {
	title string
	items []int // indexes into places()
	y, h  int   // first row inside the sidebar column, and height with the borders
}

// sidebarSections splits places() into the items of Favorites (with Recents) and of Locations.
func (m *Model) sidebarSections() (favs, locs []int) {
	into := &favs
	for i, p := range m.places() {
		switch {
		case p.divider:
			into = &locs
		case p.title != "":
		default:
			*into = append(*into, i)
		}
	}
	return favs, locs
}

// sidebarBoxLayout stacks the two boxes in the panel height: Favorites takes what its items need and Locations the
// rest. When the window is too short for both, Favorites alone fills the column.
func (m *Model) sidebarBoxLayout() []sbBox {
	favs, locs := m.sidebarSections()
	total := m.panelH()
	if total < 2*minBoxH {
		return []sbBox{{title: i18n.T("Favoritos", "Favorites"), items: favs, y: 0, h: total}}
	}
	hFav := max(minBoxH, min(len(favs)+2, total-minBoxH))
	return []sbBox{
		{title: i18n.T("Favoritos", "Favorites"), items: favs, y: 0, h: hFav},
		{title: i18n.T("Ubicaciones", "Locations"), items: locs, y: hFav, h: total - hFav},
	}
}

// sidebarItemAt returns the place on a row of the sidebar column (0 is the top border); borders and empty space
// have none.
func (m *Model) sidebarItemAt(row int) (int, bool) {
	for _, b := range m.sidebarBoxLayout() {
		if k := row - b.y - 1; row > b.y && row < b.y+b.h-1 && k < len(b.items) {
			return b.items[k], true
		}
	}
	return 0, false
}

// sidebarBoxes draws the sidebar column: the boxes one over the other, the one holding the selection in the focus
// color while the sidebar has the focus.
func (m *Model) sidebarBoxes() []string {
	var out []string
	ps := m.places()
	dests := destinations(ps)
	selected := -1
	if len(dests) > 0 {
		selected = dests[min(m.sideSel, len(dests)-1)]
	}
	w := m.sidebarW() - 2
	for _, b := range m.sidebarBoxLayout() {
		lines := make([]string, 0, len(b.items))
		active := false
		for _, pi := range b.items {
			lines = append(lines, m.placeLine(ps[pi], pi == selected && m.focus == focusSidebar, w))
			active = active || (pi == selected && m.focus == focusSidebar)
		}
		out = append(out, m.panel(b.title, "", lines, m.sidebarW(), b.h, active)...)
	}
	return out
}

// placeLine is one destination of the sidebar.
func (m *Model) placeLine(p place, focused bool, w int) string {
	glyph, lead := m.placeIcon(p)
	label := fit(lead+" "+p.name, w)
	if glyph != "" {
		label = fit(lead+glyph+" "+p.name, w)
	}
	switch {
	case focused:
		plain := "▸ " + p.name
		if glyph != "" {
			plain = "▸ " + ansi.Strip(glyph) + " " + p.name
		}
		return m.styles.selected(ansi.Strip(fit(plain, w)))
	case p.missing:
		return m.styles.dim(ansi.Strip(label))
	case p.path == m.dir():
		return m.styles.bold(label)
	}
	return label
}
