package ui

import (
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
)

// Icon sizes (config "icon_size"). A terminal cell cannot hold a bigger glyph, so the size is the room around it:
// small is the glyph and a space, medium adds air and a framed gear, large puts the glyph on a tile with roomy rows
// (docs/adr/0014-icon-size-setting.md).
const (
	iconSmall  = "small"
	iconMedium = "medium"
	iconLarge  = "large"
)

// iconW is the cells an icon takes in front of a name, or 0 when icons are off.
func (m *Model) iconW() int {
	if m.icons.Width() == 0 {
		return 0
	}
	switch m.deps.Cfg.IconSize {
	case iconSmall:
		return 2
	case iconLarge:
		return 4 // a tile of three cells and a space
	}
	return 3
}

// iconCell draws an icon with the room its size gives it; it is iconW cells wide.
func (m *Model) iconCell(ic icons.Icon) string {
	switch m.deps.Cfg.IconSize {
	case iconSmall:
		return m.styles.icon(ic.Role, ic.Glyph, ic.Dim) + " "
	case iconLarge:
		return m.styles.iconTile(ic.Role, ic.Glyph, ic.Dim) + " "
	}
	return m.styles.icon(ic.Role, ic.Glyph, ic.Dim) + "  "
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
	return m.styles.icon(ic.Role, ic.Glyph, p.missing), lead
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
	if m.icons.Style == icons.Nerd && m.deps.Cfg.IconSize != iconSmall {
		return ""
	}
	return gear
}

// gearW is the cells the Settings button takes from the right edge, margin included: the whole of it is clickable.
func (m *Model) gearW() int {
	switch m.deps.Cfg.IconSize {
	case iconSmall:
		return 4
	case iconLarge:
		return 8
	}
	return 6
}

// gearButton draws the gear: bare when small, on a tile when bigger.
func (m *Model) gearButton() string {
	switch m.deps.Cfg.IconSize {
	case iconSmall:
		return m.styles.accent(m.gearText())
	case iconLarge:
		return m.styles.tile("  "+m.gearText()+"  ", true)
	}
	return m.styles.tile(" "+m.gearText()+" ", false)
}
