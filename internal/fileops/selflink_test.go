package fileops

import (
	"os"
	"strings"
	"testing"
)

// The check that keeps a folder out of itself compared the paths as text, so a destination reached through a link
// to the folder passed it: the copy then went on copying itself and left a deep half copy inside the source.
func TestAFolderCannotBeCopiedIntoItselfThroughALink(t *testing.T) {
	w := newWorld(t)
	w.must(os.Symlink(w.path("src/tree"), w.path("link")))
	_, err := w.ops.Copy([]string{w.path("src/tree")}, w.path("link"))
	if err == nil || !strings.Contains(err.Error(), "inside itself") {
		t.Fatalf("err = %v, want the inside itself refusal", err)
	}
	if w.exists("src/tree/tree") {
		t.Fatal("a half copy was left inside the source")
	}
}
