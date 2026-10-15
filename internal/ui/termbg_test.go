package ui

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// The terminal's own background shows around the cells lazyfinder paints: the window padding of Ghostty, a cell left
// at the edge. With the theme's background on the screen, lazyfinder sets the terminal background to the same color
// (Bubble Tea resets it on exit), so changing the theme changes the whole window, not only the cells.
func TestTheTerminalBackgroundFollowsTheTheme(t *testing.T) {
	e := newEnv(t)
	got := e.m.View().BackgroundColor
	if got == nil || got != e.m.styles.bg {
		t.Fatalf("the terminal background is %v, want the theme background %v", got, e.m.styles.bg)
	}
	before := got
	e.key(",")
	e.pickSetting("Theme")
	e.key("right")
	e.key("esc")
	if after := e.m.View().BackgroundColor; after == nil || after == before {
		t.Fatalf("after a new theme the terminal background is %v, still %v", after, before)
	}
}

func TestTheTerminalBackgroundIsLeftAloneWithTheTerminalBackgroundSetting(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.ScreenBackground = config.ScreenBackgroundTerminal
	e.m.restyle()
	if got := e.m.View().BackgroundColor; got != nil {
		t.Fatalf("with screen_background terminal the terminal background must stay as it is, got %v", got)
	}
}

// The kind of each file is worked out when the folder is read, so after switching the language in Settings the
// Kind column kept the old language ("Folder" in a Spanish window) until the folder was read again.
func TestChangingTheLanguageTranslatesTheKindColumn(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	e := newEnv(t)
	e.mustHave("Folder")
	e.key(",")
	e.pickSetting("Language")
	for range len(languageCodes()) {
		if e.m.deps.Cfg.Language == "es" {
			break
		}
		e.key("right")
	}
	e.key("esc")
	if e.m.deps.Cfg.Language != "es" {
		t.Fatalf("could not pick Spanish, language is %q", e.m.deps.Cfg.Language)
	}
	e.mustHave("Carpeta")
	if strings.Contains(e.text(), "Folder") {
		t.Fatalf("the Kind column is still in English:\n%s", e.text())
	}
}
