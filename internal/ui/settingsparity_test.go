package ui

import "testing"

// Mathias, 2026-10-13: "porque en la configuracion dice kitty? ... eso deberia ser automatico como lazymark" and
// "porque dice raton? el raton va con la opcion de atajos ... como esta en lazymark". lazymark has neither row: the
// terminal is asked for Kitty support by itself, and the mouse is `mouse_click` in the file and `--no-mouse`.
func TestSettingsListHasNoTerminalProbeRows(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	e.mustHave("Theme")
	e.mustNotHave("Kitty")
	e.mustNotHave("Mouse")
	e.mustNotHave("Images")
}

func TestImagesStayAutomaticAndMouseKeepsItsConfig(t *testing.T) {
	e := newEnv(t)
	if e.m.deps.Cfg.Preview.Images != "auto" {
		t.Fatalf("images default to auto, got %q", e.m.deps.Cfg.Preview.Images)
	}
	if !e.m.mouse {
		t.Fatal("the mouse is on by default")
	}
}
