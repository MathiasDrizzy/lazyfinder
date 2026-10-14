package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mathias, 2026-10-14 (order 07, revision 2, S2): "con el click se queda pegado el texto que dice carpeta nueva N y
// confunde … no aparece visualmente como si se creó la carpeta y le estoy cambiando el nombre, además tengo que
// borrar siempre el texto que dice carpeta nueva". Like the Finder, a new folder or file is made at once with the
// default name, its row shows the name being edited, and typing replaces the whole name.

func TestANewFolderAppearsAtOnceAndTypingReplacesItsName(t *testing.T) {
	e := newEnv(t)
	e.key("N")
	if !e.onDisk("untitled folder") || e.selectedName() != "untitled folder" {
		t.Fatalf("the folder is made and selected at once:\n%s", e.text())
	}
	_, y := e.find("untitled folder")
	if !strings.Contains(e.screen()[y], "▏") {
		t.Fatalf("the row of the new folder shows its name being edited: %q", e.screen()[y])
	}
	e.typeText("fresh")
	if e.m.prompt.input != "fresh" {
		t.Fatalf("typing replaces the default name, got %q", e.m.prompt.input)
	}
	e.key("enter")
	if !e.onDisk("fresh") || e.onDisk("untitled folder") {
		t.Fatalf("enter gives the folder the typed name:\n%s", e.text())
	}
	if e.selectedName() != "fresh" {
		t.Fatalf("the renamed folder stays selected, got %q", e.selectedName())
	}
}

func TestEscapeTakesBackTheNewFolderOrFile(t *testing.T) {
	for _, k := range []string{"N", "n"} {
		e := newEnv(t)
		e.key(k)
		e.key("esc")
		if e.onDisk("untitled folder") || e.onDisk("untitled file") {
			t.Fatalf("%s then esc leaves nothing behind", k)
		}
	}
}

func TestEnterKeepsTheDefaultName(t *testing.T) {
	e := newEnv(t)
	e.key("n")
	e.key("enter")
	if fi, err := os.Stat(filepath.Join(e.root, "untitled file")); err != nil || fi.IsDir() {
		t.Fatalf("enter keeps the new file with its default name: %v", err)
	}
}

// After a key that is not text the name is edited, not replaced: backspace takes one letter of the default name.
func TestAnArrowKeyKeepsTheDefaultNameToEdit(t *testing.T) {
	e := newEnv(t)
	e.key("N")
	e.key("right")
	e.typeText(" 2")
	if e.m.prompt.input != "untitled folder 2" {
		t.Fatalf("after an arrow typing adds to the name, got %q", e.m.prompt.input)
	}
}

// A click on the icon leaves its tooltip out of the bottom row, which shows what is being made.
func TestAClickOnTheNewFolderIconShowsThePromptNotTheTooltip(t *testing.T) {
	for _, sidebar := range []bool{false, true} {
		var e *env
		if sidebar {
			e = projectEnv(t, 50)
			e.move(5, 5)
		} else {
			e = newEnv(t)
		}
		a := e.m.actionSpans()[actionNewFolder]
		e.move(a.x, a.y)
		e.click(a.x, a.y)
		last := e.screen()[e.m.h-1]
		if strings.Contains(last, ": N") || !strings.Contains(last, "New folder") || !strings.Contains(last, "untitled folder") {
			t.Fatalf("sidebar=%v: the bottom row asks for the new folder's name, not the tooltip: %q", sidebar, last)
		}
		e.typeText("aas")
		e.key("enter")
		if !strings.Contains(strings.Join(e.screen(), "\n"), "aas") || strings.Contains(e.text(), "folderaas") {
			t.Fatalf("sidebar=%v: the typed name replaces the default one:\n%s", sidebar, e.text())
		}
	}
}

// The bottom row says whether a file or a folder is being made, with its icon.
func TestTheNewFileAndNewFolderPromptsDiffer(t *testing.T) {
	e := newEnv(t)
	e.key("n")
	file := e.screen()[e.m.h-1]
	e.key("esc")
	e.key("N")
	folder := e.screen()[e.m.h-1]
	if !strings.Contains(file, "New file") || !strings.Contains(folder, "New folder") {
		t.Fatalf("labels: %q / %q", file, folder)
	}
	if fi, fo := strings.Fields(file)[0], strings.Fields(folder)[0]; fi == fo {
		t.Fatalf("each prompt starts with its own icon: %q / %q", file, folder)
	}
}
