package fileops

import (
	"os"
	"testing"
)

// An undo that fails halfway (a folder that cannot be written for a moment) used to stay on the stack as it was, so
// the next try moved the items it had already put back again, failed on them, and the rest could never come back.
func TestAnUndoThatFailsHalfwayResumesWhereItStopped(t *testing.T) {
	w := newWorld(t)
	w.put("dst/b.txt", "beta")
	w.must(w.ops.Trash([]string{w.path("src/a.txt"), w.path("dst/b.txt")}))
	w.must(os.Chmod(w.path("src"), 0o500))
	_, err := w.ops.Undo()
	w.must(os.Chmod(w.path("src"), 0o755))
	if err == nil {
		t.Fatal("the undo into a read-only folder worked")
	}
	if _, err := w.ops.Undo(); err != nil {
		t.Fatalf("second undo: %v", err)
	}
	if w.read("src/a.txt") != "alpha" || w.read("dst/b.txt") != "beta" || w.exists("dst/b copy.txt") {
		t.Fatal("the items did not come back once each")
	}
	if w.ops.CanUndo() {
		t.Fatal("the finished undo is still on the stack")
	}
}
