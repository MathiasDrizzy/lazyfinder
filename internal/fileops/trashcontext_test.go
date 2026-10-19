package fileops

import (
	"context"
	"errors"
	"testing"
)

// Trashing and undoing went through context.Background, so a trash or an undo that copies across volumes could not
// be stopped. A cancelled one stops, and an undo that stopped stays to undo.
func TestATrashOrAnUndoStopsWhenCancelled(t *testing.T) {
	w := newWorld(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.ops.TrashContext(cancelled, []string{w.path("src/a.txt")}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("trash = %v, want cancelled", err)
	}
	if !w.exists("src/a.txt") || w.ops.CanUndo() {
		t.Fatal("a cancelled trash moved the file or left a step to undo")
	}
	w.must(w.ops.TrashContext(context.Background(), []string{w.path("src/a.txt")}, nil))
	if _, err := w.ops.UndoContext(cancelled, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("undo = %v, want cancelled", err)
	}
	if w.exists("src/a.txt") || !w.ops.CanUndo() {
		t.Fatal("a cancelled undo put the file back or lost its step")
	}
	if _, err := w.ops.UndoContext(context.Background(), nil); err != nil || !w.exists("src/a.txt") {
		t.Fatalf("undo after the cancel = %v", err)
	}
}
