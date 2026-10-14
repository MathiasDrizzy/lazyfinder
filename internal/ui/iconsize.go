package ui

import (
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
)

// There is no icon size: a terminal cell cannot hold a bigger glyph, and the boxes and blank lines that stood in for
// one looked wrong (docs/adr/0015-no-icon-size.md). An icon is a glyph and a space on the plain background.

// iconW is the cells an icon takes in front of a name, or 0 when icons are off.
func (m *Model) iconW() int {
	return m.icons.Width()
}

// iconCell draws an icon; it is iconW cells wide.
func (m *Model) iconCell(ic icons.Icon) string {
	return m.styles.icon(ic, ic.Dim) + " "
}

// placeIcon is the icon of a sidebar line and the indent that goes before it. Like Finder, every entry has its own
// glyph at every size; the size only changes the room around it.
func (m *Model) placeIcon(p place) (glyph, lead string) {
	lead = " "
	if m.iconW() == 0 {
		return "", " "
	}
	ic := m.icons.Pick(icons.Item{Name: p.name, IsDir: true})
	if m.icons.Style == icons.Nerd {
		if g, ok := m.placeGlyph(p); ok {
			ic.Glyph, ic.Role = g, icons.Blue
		}
	}
	return m.styles.icon(ic, p.missing), lead
}

// userFolderGlyphs are the glyphs of the standard folders of a home, found by their place on disk and not by the name
// the favorite shows, which may be translated or renamed.
var userFolderGlyphs = map[string]string{
	"Desktop": "\uf108", "Documents": "\uf0f6", "Downloads": "\uf019", "Pictures": "\uf03e",
	"Music": "\uf001", "Movies": "\uf008", "Library": "\uf02d", ".Trash": "\uf1f8",
}

// placeGlyph picks the Nerd Font glyph of a sidebar entry by what it points to.
func (m *Model) placeGlyph(p place) (string, bool) {
	home := m.deps.Home
	switch {
	case p.recents:
		return "\uf017", true // clock
	case p.path == home:
		return "\uf015", true // house
	case p.path == "/" || p.path == "/Volumes" || strings.HasPrefix(p.path, "/Volumes/"):
		return "\uf0a0", true // disk
	case p.path == "/Applications":
		return "\uf135", true // rocket
	case home != "" && filepath.Dir(p.path) == home:
		g, ok := userFolderGlyphs[filepath.Base(p.path)]
		return g, ok
	}
	return "", false
}

// gearText is the glyph of the Settings button: the Nerd Font gear when the icons are Nerd Font, else the symbol.
func (m *Model) gearText() string {
	if m.icons.Style == icons.Nerd {
		return "\uf013"
	}
	return gear
}

// gearW is the cells of the Settings button, the same three as every other button: the whole of it is clickable.
func (m *Model) gearW() int { return 3 }

// gearButton draws the gear in the accent color, with a space on each side.
func (m *Model) gearButton() string { return m.styles.accent(" " + m.gearText() + " ") }
