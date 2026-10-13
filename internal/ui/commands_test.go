package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

func (e *env) withCommands(c map[string]config.Opener) {
	d := e.m.deps
	d.Cfg.Commands = c
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
}

func TestCommandsPickerRunsTheChosenCommandOnTheSelectionInItsFolder(t *testing.T) {
	e := newEnv(t)
	e.withCommands(map[string]config.Opener{
		"count": {Argv: []string{"wc", "-l", "{file}"}, Mode: "gui"},
		"build": {Argv: []string{"make"}, Mode: "gui"},
	})
	e.clickText("notes.txt")
	e.key("!")
	if !e.m.picker.open {
		t.Fatal("the picker must open")
	}
	e.mustHave("build")
	e.key("down") // build, count: count second
	e.key("enter")
	if len(e.run.calls) != 1 || e.run.calls[0][0] != "wc" || e.run.calls[0][2] != filepath.Join(e.root, "notes.txt") {
		t.Fatalf("calls = %v", e.run.calls)
	}
	if len(e.run.dirs) != 1 || e.run.dirs[0] != e.root {
		t.Fatalf("dirs = %v, want the current folder", e.run.dirs)
	}
	if e.m.picker.open {
		t.Fatal("choosing closes the picker")
	}
}

func TestCommandsWithoutAnyExplainsHowToAddOne(t *testing.T) {
	e := newEnv(t)
	e.key("!")
	if !strings.Contains(e.text(), "No custom commands") {
		t.Fatalf("status:\n%s", e.text())
	}
}

func TestACommandThatNeedsAFileIsRefusedOnAnEmptyFolder(t *testing.T) {
	e := newEnv(t)
	e.withCommands(map[string]config.Opener{"count": {Argv: []string{"wc", "{file}"}, Mode: "gui"}})
	e.m.listing.Entries = nil
	e.key("!")
	e.key("enter")
	if len(e.run.calls) != 0 {
		t.Fatalf("nothing is selected, calls = %v", e.run.calls)
	}
	e.mustHave("Select a file")
}
