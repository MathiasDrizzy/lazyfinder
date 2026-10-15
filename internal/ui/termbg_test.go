package ui

import (
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
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
