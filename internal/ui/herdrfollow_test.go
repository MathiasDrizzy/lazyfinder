package ui

import (
	"os"
	"path/filepath"
	"testing"
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
