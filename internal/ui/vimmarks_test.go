package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// m and a letter mark the folder; ' and that letter go back to it from anywhere, like Vim's marks. The marks survive a
// restart and ' lists them, so none is hidden.
func TestVimMarksGoBackToAFolder(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.ConfigPath = filepath.Join(t.TempDir(), "config.json")
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.pick("alpha")
	e.key("enter")
	alpha := e.m.dir()
	e.key("m")
	e.key("a")
	if !strings.Contains(e.m.status, "a") || e.m.vimMarks["a"] != alpha {
		t.Fatalf("mark a = %q, status %q", e.m.vimMarks["a"], e.m.status)
	}
	e.key("backspace")
	e.key("'")
	if !e.m.picker.open || !strings.Contains(e.m.picker.items[0].label, "a") {
		t.Fatalf("' must list the marks: open=%v", e.m.picker.open)
	}
	e.key("a")
	if e.m.dir() != alpha || e.m.picker.open {
		t.Fatalf("' a went to %q, want %q", e.m.dir(), alpha)
	}
	if again := New(d); again.vimMarks["a"] != alpha {
		t.Fatalf("marks after a restart: %v", again.vimMarks)
	}
}

// After m, a key that is not a letter or a digit cancels without marking, and does nothing else.
func TestMarkCancelledByAnotherKey(t *testing.T) {
	e := newEnv(t)
	e.key("m")
	e.key("esc")
	e.key("j")
	if len(e.m.vimMarks) != 0 || e.m.awaitMark {
		t.Fatalf("marks %v, waiting %v", e.m.vimMarks, e.m.awaitMark)
	}
	e.key("'")
	if e.m.picker.open || !strings.Contains(e.m.status, "No marks") {
		t.Fatalf("' without marks: open %v, status %q", e.m.picker.open, e.m.status)
	}
}
