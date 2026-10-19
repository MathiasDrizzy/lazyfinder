package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// U redoes what u undid, and H lists what can be undone and redone.
func TestURedoesAndHListsTheHistory(t *testing.T) {
	e := newEnv(t)
	e.key("U")
	if !strings.Contains(e.m.status, "Nothing to redo") {
		t.Fatalf("U with nothing to redo: %q", e.m.status)
	}
	e.key("N")
	e.typeText("made")
	e.key("enter")
	made := filepath.Join(e.root, "made")
	e.key("u") // N made "untitled folder" and then renamed it: u undoes the rename
	if _, err := os.Stat(made); err == nil {
		t.Fatal("undo left the folder")
	}
	e.key("H")
	items := e.m.picker.items
	if !e.m.picker.open || len(items) != 2 || !strings.HasPrefix(items[0].label, "New folder") || items[0].hint != "undo" ||
		!strings.HasPrefix(items[1].label, "Rename of") || items[1].hint != "redo" {
		t.Fatalf("history: open %v, items %+v", e.m.picker.open, items)
	}
	e.key("esc")
	e.key("U")
	if _, err := os.Stat(made); err != nil || !strings.Contains(e.m.status, "Redone") {
		t.Fatalf("redo: %v, status %q", err, e.m.status)
	}
	e.key("ctrl+r")
	if !strings.Contains(e.m.status, "Nothing to redo") {
		t.Fatalf("second redo: %q", e.m.status)
	}
}

// H with no history says so instead of opening an empty list.
func TestHWithoutHistory(t *testing.T) {
	e := newEnv(t)
	e.key("H")
	if e.m.picker.open || !strings.Contains(e.m.status, "Nothing to undo") {
		t.Fatalf("open %v, status %q", e.m.picker.open, e.m.status)
	}
}

// A redo can copy as much as a paste, so it runs apart from the UI the same way: the status line shows it, Esc can
// cancel it, and no other change to the files starts until it ends.
func TestARedoRunsApartFromTheUI(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("c")
	e.key("p")
	e.key("u")
	cmd := e.m.onKey(keyMsg("U"))
	if !e.m.op.busy || !strings.Contains(e.m.status, "Redoing") || cmd == nil {
		t.Fatalf("redo: busy %v, status %q", e.m.op.busy, e.m.status)
	}
	if e.m.onKey(keyMsg("N")); !strings.Contains(e.m.status, "Wait") {
		t.Fatalf("a change during the redo: %q", e.m.status)
	}
	for _, msg := range flatten(cmd) {
		if done, ok := msg.(opDoneMsg); ok {
			e.send(done)
		}
	}
	if e.m.op.busy || !strings.Contains(e.m.status, "Redone") {
		t.Fatalf("after the redo: busy %v, status %q", e.m.op.busy, e.m.status)
	}
	if _, err := os.Stat(filepath.Join(e.root, "notes copy.txt")); err != nil {
		t.Fatal("the redo did not copy again:", err)
	}
}
