package fileops

import (
	"context"
	"errors"
	"testing"
)

// Get Info walked a folder inside the UI's update: on / or a home folder that froze the program for minutes. The
// walk is now Count, which runs apart and stops when its context is cancelled.
func TestCountWalksAFolderAndStopsWhenCancelled(t *testing.T) {
	w := newWorld(t)
	files, total, err := Count(context.Background(), w.path("src/tree"))
	if err != nil || files != 2 || total != int64(len("inner")+len("deepest")) {
		t.Fatalf("files=%d total=%d err=%v", files, total, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Count(ctx, w.path("src")); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if in, err := Info(w.path("src/tree")); err != nil || !in.IsDir || in.Files != 0 {
		t.Fatalf("Info walked the folder: %+v %v", in, err)
	}
}
