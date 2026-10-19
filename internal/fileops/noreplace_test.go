package fileops

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

// move checked for a free name and renamed afterwards: a file created at that name in between was replaced for good.
// The rename itself now refuses to replace anything.
func TestMoveNeverReplacesAFileThatAppearedAtTheTarget(t *testing.T) {
	w := newWorld(t)
	w.put("dst/a.txt", "theirs")
	err := moveContext(context.Background(), w.path("src/a.txt"), w.path("dst/a.txt"), nil)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("err = %v, want fs.ErrExist", err)
	}
	if w.read("dst/a.txt") != "theirs" || w.read("src/a.txt") != "alpha" {
		t.Fatal("the file at the target was replaced")
	}
}
