package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// order 07, point M6: at any size the popups fit the window and Settings scrolls to every row.
func TestSettingsFitsAndScrollsAtSmallSizes(t *testing.T) {
	for _, sz := range [][2]int{{20, 10}, {24, 15}, {30, 25}, {40, 25}} {
		e := projectEnv(t, sz[0])
		e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		e.key(",")
		for i := 0; i < settingCount; i++ {
			for y, l := range e.screen() {
				if w := ansi.StringWidth(l); w > sz[0] {
					t.Fatalf("%dx%d row %d is %d cells wide", sz[0], sz[1], y, w)
				}
			}
			if len(e.screen()) > sz[1] {
				t.Fatalf("%dx%d: %d rows", sz[0], sz[1], len(e.screen()))
			}
			label := e.m.settingLabel(e.m.set.sel)
			if !strings.Contains(e.text(), ansi.Truncate(label, 3, "")) {
				t.Fatalf("%dx%d: the selected row %q is not visible:\n%s", sz[0], sz[1], label, e.text())
			}
			e.key("down")
		}
	}
}
