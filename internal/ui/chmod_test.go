package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// P asks for the permissions of the selection in octal, starting from the current ones; Enter sets them, u undoes.
func TestPSetsThePermissionsOfTheSelection(t *testing.T) {
	e := newEnv(t)
	p := filepath.Join(e.root, "notes.txt")
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	e.pick("notes.txt")
	e.key("P")
	if e.m.prompt.kind != promptChmod || e.m.prompt.input != "644" {
		t.Fatalf("prompt %v %q, want the chmod prompt with 644", e.m.prompt.kind, e.m.prompt.input)
	}
	e.key("backspace")
	e.key("backspace")
	e.typeText("00")
	e.key("enter")
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 || !strings.Contains(e.m.status, "600") {
		t.Fatalf("mode %v, status %q", fi.Mode().Perm(), e.m.status)
	}
	e.key("u")
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o644 {
		t.Fatalf("undo left %v", fi.Mode().Perm())
	}
}

// A mode that is not octal keeps the prompt open with an error, and changes nothing.
func TestABadModeKeepsThePromptOpen(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("P")
	e.typeText("9")
	e.key("enter")
	if e.m.prompt.kind != promptChmod || !e.m.statusErr {
		t.Fatalf("prompt %v, error %v %q", e.m.prompt.kind, e.m.statusErr, e.m.status)
	}
}
