package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestColumnLayoutShowsParentCurrentAndAPreviewOfTheSelection(t *testing.T) {
	e := newEnv(t)
	if err := os.Mkdir(filepath.Join(e.home, "sibling"), 0o755); err != nil {
		t.Fatal(err)
	}
	e.key("M")
	e.mustNotHave("Date Modified") // no detail columns in this layout
	e.mustHave("sibling")          // parent column: the folders next to this one
	e.clickText("alpha")
	e.mustHave("inner.txt") // right column: what is inside the selected folder
	e.clickText("notes.txt")
	e.mustHave("first line") // right column: the start of a text file
}

func TestColumnLayoutToggleReturnsToTheList(t *testing.T) {
	e := newEnv(t)
	e.key("M")
	e.key("M")
	e.mustHave("Date Modified")
}

func TestClickingTheParentColumnGoesThere(t *testing.T) {
	e := newEnv(t)
	if err := os.Mkdir(filepath.Join(e.home, "sibling"), 0o755); err != nil {
		t.Fatal(err)
	}
	e.key("M")
	e.clickText("sibling")
	if filepath.Base(e.m.dir()) != "sibling" {
		t.Fatalf("dir = %s", e.m.dir())
	}
}

func TestColumnLayoutKeepsKeyboardNavigation(t *testing.T) {
	e := newEnv(t)
	e.key("M")
	e.key("enter") // alpha
	e.mustHave("inner.txt")
	e.key("backspace")
	if filepath.Base(e.m.dir()) != "proj" {
		t.Fatalf("dir = %s", e.m.dir())
	}
}

func TestLayoutFromTheConfig(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.Cfg.Layout = "columns"
	e.m = New(d)
	e.mustNotHave("Date Modified")
}
