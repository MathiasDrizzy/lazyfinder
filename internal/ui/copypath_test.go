package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	sysclip "github.com/MathiasDrizzy/lazyfinder/internal/clipboard"
)

// Mathias, 2026-10-15: «quiero que al apretar click derecho o algo asi poder copiar la ruta de la carpeta en
// cualquiera que aprete», pointing at the breadcrumb of the bottom bar.

// copied makes the model copy into a slice instead of the clipboard and returns it.
func (e *env) copied() *[]string {
	var got []string
	e.m.deps.Copy = func(s string) error { got = append(got, s); return nil }
	return &got
}

func TestRightClickOnABreadcrumbCopiesThatFolder(t *testing.T) {
	e := newEnv(t)
	got := e.copied()
	cs := e.m.crumbs()
	if len(cs) < 2 {
		t.Fatalf("the breadcrumb needs two folders: %+v", cs)
	}
	parent := cs[len(cs)-2]
	e.rightClick(parent.x, e.m.h-2)
	if len(*got) != 1 || (*got)[0] != filepath.Dir(e.root) {
		t.Fatalf("copied %q, want the folder of that crumb %q", *got, filepath.Dir(e.root))
	}
	if e.m.dir() != e.root {
		t.Fatalf("a right click copies and stays: now in %q", e.m.dir())
	}
	e.mustHave("Copied " + filepath.Dir(e.root))
}

func TestRightClickOnTheLastBreadcrumbCopiesTheCurrentFolder(t *testing.T) {
	e := newEnv(t)
	got := e.copied()
	cs := e.m.crumbs()
	last := cs[len(cs)-1]
	e.rightClick(last.x+last.w-1, e.m.h-2)
	if len(*got) != 1 || (*got)[0] != e.root {
		t.Fatalf("copied %q, want %q", *got, e.root)
	}
}

func TestACopyThatFailsSaysSo(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Copy = func(string) error { return errors.New("xclip: exit status 1") }
	cs := e.m.crumbs()
	e.rightClick(cs[len(cs)-1].x, e.m.h-2)
	e.mustHave("Cannot copy the path: xclip: exit status 1")
}

// Without a clipboard tool (over SSH) the terminal sets the clipboard: lazyfinder sends OSC 52.
func TestWithoutAClipboardToolTheTerminalTakesTheCopy(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Copy = func(string) error { return sysclip.ErrNoTool }
	cs := e.m.crumbs()
	_, cmd := e.m.Update(tea.MouseClickMsg{X: cs[len(cs)-1].x, Y: e.m.h - 2, Button: tea.MouseRight})
	if cmd == nil {
		t.Fatal("no OSC 52 command")
	}
	if got := fmt.Sprintf("%T", cmd()); got != "tea.setClipboardMsg" {
		t.Fatalf("the command sends %s, want the clipboard message of Bubble Tea", got)
	}
	if e.m.status != "Copied "+e.root {
		t.Fatalf("status %q", e.m.status)
	}
}

// y copies the path of the selected row, or of every marked row one per line, like yy in ranger and lf.
func TestYCopiesThePathOfTheSelectionOrOfTheMarkedRows(t *testing.T) {
	e := newEnv(t)
	got := e.copied()
	e.pick("notes.txt")
	e.key("y")
	if len(*got) != 1 || (*got)[0] != filepath.Join(e.root, "notes.txt") {
		t.Fatalf("copied %q", *got)
	}
	e.pick("alpha")
	e.key("v")
	e.pick("notes.txt")
	e.key("v")
	e.key("y")
	want := filepath.Join(e.root, "alpha") + "\n" + filepath.Join(e.root, "notes.txt")
	if len(*got) != 2 || (*got)[1] != want {
		t.Fatalf("copied %q, want %q", (*got)[len(*got)-1], want)
	}
	e.mustHave("Copied 2 items")
}
