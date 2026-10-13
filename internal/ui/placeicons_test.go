package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
)

// Mathias, 2026-10-13: "ubicaciones y favoritos también tiene que tener iconos como está en Finder". Every entry of
// the sidebar has its own glyph, chosen by where it points (a translated or renamed favorite keeps it).
func TestEverySidebarEntryHasItsOwnGlyph(t *testing.T) {
	e := newEnv(t)
	e.m.icons = icons.Set{Style: icons.Nerd}
	home := e.m.deps.Home
	cases := []struct {
		name string
		p    place
		want string
	}{
		{"Recents", place{name: "Recents", recents: true}, ""},
		{"Home", place{name: "Home", path: home}, ""},
		{"Desktop", place{name: "Escritorio", path: filepath.Join(home, "Desktop")}, ""},
		{"Documents", place{name: "Docs", path: filepath.Join(home, "Documents")}, ""},
		{"Downloads", place{name: "Descargas", path: filepath.Join(home, "Downloads")}, ""},
		{"Applications", place{name: "Apps", path: "/Applications"}, ""},
		{"Pictures", place{name: "Pictures", path: filepath.Join(home, "Pictures")}, ""},
		{"Music", place{name: "Music", path: filepath.Join(home, "Music")}, ""},
		{"Movies", place{name: "Movies", path: filepath.Join(home, "Movies")}, ""},
		{"Trash", place{name: "Papelera", path: filepath.Join(home, ".Trash")}, ""},
		{"Disk", place{name: "Macintosh HD", path: "/"}, ""},
		{"Volumes", place{name: "Volumes", path: "/Volumes"}, ""},
		{"a folder of the user", place{name: "proj", path: filepath.Join(home, "proj")}, ""},
	}
	for _, c := range cases {
		glyph, _ := e.m.placeIcon(c.p)
		if !strings.Contains(glyph, c.want) {
			t.Errorf("%s: glyph %q, want U+%04X", c.name, glyph, []rune(c.want)[0])
		}
	}
}

func TestSidebarIconsAlsoShowAtTheSmallSize(t *testing.T) {
	e := newEnv(t)
	e.m.icons = icons.Set{Style: icons.Nerd}
	e.m.deps.Cfg.IconSize = "small"
	if glyph, _ := e.m.placeIcon(place{name: "Home", path: e.m.deps.Home}); glyph == "" {
		t.Fatal("Finder shows an icon on every entry, whatever the size")
	}
}
