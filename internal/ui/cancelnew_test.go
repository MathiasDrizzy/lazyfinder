package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Esc on the name of a new folder takes the folder back, and that is not something to redo: U made again the folder
// the user had cancelled.
func TestACancelledNewFolderCannotBeRedone(t *testing.T) {
	e := newEnv(t)
	e.key("N")
	e.key("esc")
	if _, err := os.Stat(filepath.Join(e.root, "untitled folder")); err == nil {
		t.Fatal("esc left the new folder")
	}
	e.key("U")
	if _, err := os.Stat(filepath.Join(e.root, "untitled folder")); err == nil || !strings.Contains(e.m.status, "Nothing to redo") {
		t.Fatalf("U made the cancelled folder again: %v, status %q", err, e.m.status)
	}
}
