package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func ctrl(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl} }

func TestCtrlGOpensAPathPromptAndEnterGoesThere(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.mustHave("Go to")
	e.typeText(filepath.Join(e.root, "alpha", "deep"))
	e.key("enter")
	if e.m.dir() != filepath.Join(e.root, "alpha", "deep") {
		t.Fatalf("dir = %s", e.m.dir())
	}
	e.mustNotHave("Go to")
}

func TestTildeAndRelativePathsAreUnderstood(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText("~")
	e.key("enter")
	if e.m.dir() != e.home {
		t.Fatalf("dir = %s, want the home %s", e.m.dir(), e.home)
	}
	e.send(ctrl('g'))
	e.typeText("proj/alpha")
	e.key("enter")
	if e.m.dir() != filepath.Join(e.root, "alpha") {
		t.Fatalf("a relative path starts from the current folder, dir = %s", e.m.dir())
	}
}

func TestTabCompletesAFolderName(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText(e.root + "/al")
	e.key("tab")
	if got, want := e.m.goto_.input, e.root+"/alpha/"; got != want {
		t.Fatalf("input = %q, want %q", got, want)
	}
	e.typeText("de")
	e.key("tab")
	if got, want := e.m.goto_.input, e.root+"/alpha/deep/"; got != want {
		t.Fatalf("input = %q, want %q", got, want)
	}
}

func TestTabCompletesTheCommonPrefixOfSeveralFolders(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText(e.root + "/")
	e.key("tab") // alpha and beta share nothing: nothing is added
	if got, want := e.m.goto_.input, e.root+"/"; got != want {
		t.Fatalf("input = %q, want %q", got, want)
	}
}

func TestANonexistentPathShowsAnErrorAndKeepsThePrompt(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText(filepath.Join(e.root, "nope"))
	e.key("enter")
	e.mustHave("Error")
	if !e.m.goto_.open || e.m.dir() != e.root {
		t.Fatal("a bad path keeps the prompt open and the folder unchanged")
	}
}

func TestAFilePathGoesToItsFolderAndSelectsIt(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText(filepath.Join(e.root, "alpha", "inner.txt"))
	e.key("enter")
	sel, _ := e.m.selected()
	if e.m.dir() != filepath.Join(e.root, "alpha") || sel.Name != "inner.txt" {
		t.Fatalf("dir %s selected %s", e.m.dir(), sel.Name)
	}
}

func TestEscCancelsTheGoToPrompt(t *testing.T) {
	e := newEnv(t)
	e.send(ctrl('g'))
	e.typeText("whatever")
	e.key("esc")
	if e.m.goto_.open {
		t.Fatal("esc closes the prompt")
	}
}
