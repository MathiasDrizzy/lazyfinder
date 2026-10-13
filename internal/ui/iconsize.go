package ui

import (
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

// placeIcon is the icon of a sidebar line and the indent that goes before it. Small icons leave the sidebar plain.
func (m *Model) placeIcon(p place) (glyph, lead string) {
	lead = " "
	if m.iconW() == 0 || m.deps.Cfg.IconSize == iconSmall {
		return "", " "
	}
	ic := m.icons.Pick(icons.Item{Name: p.name, IsDir: true})
	if m.icons.Style == icons.Nerd {
		switch {
		case p.recents:
			ic.Glyph = ""
		case p.path == m.deps.Home:
			ic.Glyph = ""
		case p.path == "/":
			ic.Glyph = ""
		}
	}
	return m.styles.icon(ic.Role, ic.Glyph, p.missing), lead
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
