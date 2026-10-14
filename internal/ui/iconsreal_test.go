package ui

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
)

// Mathias, 2026-10-13, after trying v0.11.0: "los iconos se agrandan mal, ademas dejan como un cuadrado alrededor del
// icono". A terminal cell cannot hold a bigger glyph, so there is no icon size: every icon is a glyph and a space, on
// the plain background, and the gear is a bare glyph. A file that still has "icon_size" (any value) changes nothing.
func TestIconsHaveNoBoxAndNoSize(t *testing.T) {
	for _, legacy := range []string{"", "small", "medium", "large"} {
		e := newEnv(t)
		e.m.icons = icons.Set{Style: icons.Nerd}
		e.m.deps.Cfg.IconSize = legacy
		if w := e.m.iconW(); w != 2 {
			t.Errorf("icon_size %q: an icon is a glyph and a space (2 cells), got %d", legacy, w)
		}
		if h := e.m.rowH(); h != 1 {
			t.Errorf("icon_size %q: rows stay one line in the compact density, got %d", legacy, h)
		}
		if strings.Contains(e.m.gearButton(), "48;") {
			t.Errorf("icon_size %q: the gear has a background box: %q", legacy, e.m.gearButton())
		}
		glyph, _ := e.m.placeIcon(place{name: "Home", path: e.m.deps.Home})
		if strings.Contains(glyph, "48;") {
			t.Errorf("icon_size %q: a sidebar icon has a background box: %q", legacy, glyph)
		}
		if strings.Contains(e.m.iconCell(icons.Icon{Glyph: ""}), "48;") {
			t.Errorf("icon_size %q: a list icon has a background box", legacy)
		}
	}
}

func TestSettingsHasNoIconSizeRowAndTheGearStillOpensThem(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	e.mustNotHave("Icon size")
	e.key("esc")
	e.send(sizeMsg())
	e.click(e.m.w-1, 0)
	if !e.m.set.open {
		t.Fatal("a click on the gear opens Settings")
	}
}

func TestSidebarIconsShowOnEveryEntry(t *testing.T) {
	e := newEnv(t)
	e.m.icons = icons.Set{Style: icons.Nerd}
	if glyph, _ := e.m.placeIcon(place{name: "Home", path: e.m.deps.Home}); glyph == "" {
		t.Fatal("Finder shows an icon on every entry")
	}
}
