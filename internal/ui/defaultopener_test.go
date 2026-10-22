package ui

import (
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
)

// menuTo selects the row of the Open With menu called name.
func (e *env) menuTo(name string) {
	e.t.Helper()
	for range e.m.menu.items {
		if e.m.menu.items[e.m.menu.sel].Name == name {
			return
		}
		e.key("down")
	}
	e.t.Fatalf("no %q in Open With", name)
}

// Mathias, 2026-10-22: "cuando ponga O en donde dice default ponga el programa que sea default y la opcion de
// cambiarlo, asi por ejemplo si quiero abrir un .md y esta en default textedit, pueda cambiarlo a vscode, a micro".
// The system row names the application the system opens the file with, ✓ marks the program Enter uses, and d makes
// the selected one the program of that file type, or gives it back to the system.
func TestOpenWithShowsAndChangesTheDefaultProgram(t *testing.T) {
	old := opener.DefaultApp
	t.Cleanup(func() { opener.DefaultApp = old })
	opener.DefaultApp = func(string) string { return "TextEdit" }
	e := newEnv(t)
	path := e.withConfigFile()
	e.clickText("notes.txt")
	e.key("O")
	e.mustHave("System: TextEdit")
	e.mustHave("edit ✓") // open_with has txt: edit
	e.menuTo("vim")
	e.key("d")
	if e.m.deps.Cfg.OpenWith["txt"] != "vim" || e.savedConfig(path).OpenWith["txt"] != "vim" {
		t.Fatalf("open_with = %v", e.m.deps.Cfg.OpenWith)
	}
	e.mustHave("vim ✓")
	e.mustNotHave("edit ✓")
	if c := e.m.resolve.Resolve(e.root + "/notes.txt"); c.Name != "vim" {
		t.Fatalf("Enter opens with %q", c.Name)
	}
	e.menuTo(opener.DefaultName)
	e.key("d")
	if _, ok := e.m.deps.Cfg.OpenWith["txt"]; ok {
		t.Fatalf("open_with = %v: the system takes .txt back", e.m.deps.Cfg.OpenWith)
	}
	e.mustHave("System: TextEdit ✓")
}
