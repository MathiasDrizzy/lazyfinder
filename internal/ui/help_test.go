package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Mathias, 2026-10-24: in the herdr sidebar the help showed only the keys and an ellipsis where the text goes.
func TestTheHelpOfANarrowSidebarShowsWhatEachKeyDoes(t *testing.T) {
	for _, w := range []int{30, 40, 60, 100} {
		e := newEnv(t)
		pluginPane(e)
		e.send(tea.WindowSizeMsg{Width: w, Height: 60})
		e.m.help.open = true
		text := e.text()
		for _, s := range []string{"Move the selection", "Go to a path"} {
			if !strings.Contains(strings.Join(strings.Fields(text), " "), s) {
				t.Fatalf("width %d: %q missing from the help:\n%s", w, s, text)
			}
		}
	}
}

func TestTheHelpOfANarrowSidebarScrollsToItsLastLine(t *testing.T) {
	e := newEnv(t)
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 30, Height: 20})
	e.m.help.open = true
	for range 200 {
		e.send(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if text := strings.Join(strings.Fields(e.text()), " "); !strings.Contains(text, "quit") {
		t.Fatalf("the last help line is out of reach:\n%s", e.text())
	}
}
