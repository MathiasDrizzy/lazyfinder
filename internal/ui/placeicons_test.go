package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
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

// On Linux the Trash is ~/.local/share/Trash/files, which is not a folder of the home: it is still the Trash.
func TestTheConfiguredTrashGetsTheTrashGlyphAndTheComputerIsLocalized(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Ops.TrashDir = filepath.Join(e.home, ".local", "share", "Trash", "files")
	if g, ok := e.m.placeGlyph(place{name: "Trash", path: e.m.deps.Cfg.Ops.TrashDir}); !ok || g != "" {
		t.Fatalf("glyph = %q, %v", g, ok)
	}
	i18n.SetLanguage("es")
	t.Cleanup(func() { i18n.SetLanguage("en") })
	if got := localName("Computer"); got != "Equipo" {
		t.Fatalf("Computer reads %q in Spanish", got)
	}
}
