package theme

import (
	"image/color"
	"testing"
)

// high-contrast is for low vision: every text color reads at 7:1 at least (WCAG AAA) on the background, the
// inactive selection and the selection, so no part of the interface needs the color to be lifted.
func TestTheHighContrastThemeReadsAtSevenToOneEverywhere(t *testing.T) {
	p, ok := Lookup("high-contrast")
	if !ok {
		t.Fatal("high-contrast is not a built-in theme")
	}
	texts := map[string]color.Color{"text": p.Text, "subtext": p.Subtext0, "overlay": p.Overlay0, "peach": p.Peach, "mauve": p.Mauve,
		"teal": p.Teal, "green": p.Green, "red": p.Red, "blue": p.Blue, "yellow": p.Yellow}
	for bgName, bg := range map[string]color.Color{"base": p.Base, "surface0": p.Surface0, "surface1": p.Surface1} {
		for name, fg := range texts {
			if r := Contrast(fg, bg); r < 7 {
				t.Errorf("%s on %s: %.1f:1", name, bgName, r)
			}
		}
	}
	if Contrast(p.Surface1, p.Base) < 1.5 {
		t.Error("the selection must stand apart from the background")
	}
}
