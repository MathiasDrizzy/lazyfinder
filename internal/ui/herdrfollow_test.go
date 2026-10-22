package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

// Order 2026-10-15-16: as the pane of the herdr plugin the sidebar «siga la carpeta donde esta el space»: it goes to
// the folder of the pane next to it when that folder changes, and a folder picked by hand stays until it does.
func TestTheSidebarFollowsTheFolderOfTheHerdrPane(t *testing.T) {
	e := newEnv(t)
	other := filepath.Join(e.root, "sub")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	next := e.root
	e.m.deps.HerdrDir = func() string { return next }
	if e.m.Init() == nil {
		t.Fatal("no command to poll herdr")
	}
	// Update, not e.send: e.send runs every command, and the poll asks again forever
	follow := func(dir string) { e.m.Update(herdrDirMsg{dir: dir}) }
	follow(e.root)
	if e.m.dir() != e.root {
		t.Fatalf("dir %q", e.m.dir())
	}
	follow(other) // cd in the pane next to it
	if e.m.dir() != other {
		t.Fatalf("did not follow: %q", e.m.dir())
	}
	e.m.enter(e.root, "") // the user goes back by hand
	follow(other)
	if e.m.dir() != e.root {
		t.Fatalf("the same folder again must not undo a choice by hand: %q", e.m.dir())
	}
	follow("") // the sidebar has the focus
	follow(filepath.Join(e.root, "gone"))
	if e.m.dir() != e.root {
		t.Fatalf("a folder that does not exist must not be followed: %q", e.m.dir())
	}
}

func TestWithoutHerdrNothingIsPolled(t *testing.T) {
	e := newEnv(t)
	if _, cmd := e.m.Update(herdrDirMsg{dir: e.root}); cmd != nil {
		t.Fatal("no poll outside herdr")
	}
}

// Mathias, 2026-10-22: "si salgo con q me cambio de space y despues vuelvo al mismo space se abre automaticamente,
// esto no deberia pasar porque me sali". As the pane of the herdr plugin, q tells herdr the sidebar was closed in its
// tab before it quits.
func TestQuittingThePluginPaneHidesTheSidebarInItsTab(t *testing.T) {
	e := newEnv(t)
	hidden := 0
	e.m.deps.HerdrHide = func() { hidden++ }
	e.key("q")
	if hidden != 1 || !e.m.quitting {
		t.Fatalf("hidden %d, quitting %v", hidden, e.m.quitting)
	}
}

// Mathias, 2026-10-22: the sidebar keeps its width when the window changes ("debe ser automatico"), and leaving the
// sidebar mode widens it to what the normal mode needs ("que el recuadro se acomode con el minimo para funcionar").
func TestThePluginPaneKeepsItsWidth(t *testing.T) {
	e := newEnv(t)
	type fit struct{ normal, force bool }
	var fits []fit
	e.m.deps.HerdrFit = func(_ config.Herdr, normal, force bool) error {
		fits = append(fits, fit{normal, force})
		return nil
	}
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	if len(fits) != 1 || fits[0] != (fit{false, false}) {
		t.Fatalf("on a new size: %+v", fits)
	}
	e.key("ctrl+b")
	if len(fits) != 2 || fits[1] != (fit{true, true}) {
		t.Fatalf("leaving the sidebar mode: %+v", fits)
	}
	e.key("ctrl+b")
	if len(fits) != 3 || fits[2] != (fit{false, true}) {
		t.Fatalf("back in the sidebar mode: %+v", fits)
	}
}

// In the pane of the herdr plugin the width follows the mode, not the other way round: herdr widens the pane when the
// screen grows (another monitor), and that must not switch it to the normal mode; ctrl+b does.
func TestThePluginPaneKeepsItsModeWhenHerdrWidensIt(t *testing.T) {
	e := newEnv(t)
	e.m.deps.HerdrFit = func(config.Herdr, bool, bool) error { return nil }
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	e.send(tea.WindowSizeMsg{Width: 67, Height: 30})
	if !e.m.projectMode() {
		t.Fatal("a wider screen switched the plugin pane to the normal mode")
	}
}

// pluginPane starts e as the pane of the herdr plugin does: in the sidebar mode.
func pluginPane(e *env) {
	if !e.m.projectMode() {
		e.m.toggleProject()
	}
}
