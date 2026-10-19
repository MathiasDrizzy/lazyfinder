package fileops

import (
	"context"
	"errors"
	"os"
	"testing"
)

// A big copy ran inside the UI with no way to stop it. CopyContext reports the bytes it writes and stops when its
// context is cancelled, leaving no half copy behind and nothing to undo.
func TestACopyReportsItsBytesAndStopsWhenCancelled(t *testing.T) {
	w := newWorld(t)
	var got int64
	made, err := w.ops.CopyContext(context.Background(), []string{w.path("src/tree")}, w.dst, func(n int64) { got += n })
	if err != nil || len(made) != 1 || got != int64(len("inner")+len("deepest")) {
		t.Fatalf("made=%v err=%v bytes=%d", made, err, got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err = w.ops.CopyContext(ctx, []string{w.path("src/a.txt"), w.path("src/tree")}, w.dst, func(int64) {
		if calls++; calls == 1 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if w.exists("dst/tree copy") {
		t.Fatal("the cancelled copy left a half copy")
	}
	if _, err := w.ops.Undo(); err != nil || w.exists("dst/a.txt") {
		t.Fatalf("the item copied before the cancel must be undoable: %v", err)
	}
}

func TestAMoveCanBeCancelledBeforeItStarts(t *testing.T) {
	w := newWorld(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := w.ops.MoveContext(ctx, []string{w.path("src/a.txt")}, w.dst, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if !w.exists("src/a.txt") || w.exists("dst/a.txt") {
		t.Fatal("a cancelled move must leave the source where it was")
	}
}

// A copy whose target appears between choosing the name and making it fails, and must not remove what appeared
// there: move() removed it as its own half copy.
func TestAFailedCopyNeverRemovesATargetItDidNotMake(t *testing.T) {
	w := newWorld(t)
	w.put("dst/theirs.txt", "keep me")
	if err := copyNew(context.Background(), w.path("src/a.txt"), w.path("dst/theirs.txt"), nil); !errors.Is(err, os.ErrExist) {
		t.Fatalf("err = %v, want ErrExist", err)
	}
	if w.read("dst/theirs.txt") != "keep me" {
		t.Fatal("the file that was already there changed")
	}
	if err := os.Mkdir(w.path("dst/theirdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	w.put("dst/theirdir/f", "keep")
	if err := copyNew(context.Background(), w.path("src/tree"), w.path("dst/theirdir"), nil); !errors.Is(err, os.ErrExist) {
		t.Fatalf("folder: err = %v", err)
	}
	if w.read("dst/theirdir/f") != "keep" {
		t.Fatal("the folder that was already there lost its content")
	}
}
