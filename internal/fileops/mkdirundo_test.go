package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Undoing a new folder removed whatever had its name by then, and os.Remove takes a file too: a folder made in
// lazyfinder, replaced outside it by a file with data, was deleted for good on U. Only the empty folder it made goes;
// anything else under that name is kept, in the trash.
func TestUndoingANewFolderNeverDeletesAFileThatTookItsName(t *testing.T) {
	w := newWorld(t)
	p, err := w.ops.Mkdir(w.path("src"), "fresh")
	w.must(err)
	w.must(os.Remove(p))
	w.put("src/fresh", "data that matters")
	if _, err := w.ops.Undo(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(w.trash)
	w.must(err)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "fresh") {
			if b, err := os.ReadFile(filepath.Join(w.trash, e.Name())); err == nil && string(b) == "data that matters" {
				return
			}
		}
	}
	t.Fatalf("the file that took the folder's name is gone; trash has %v", entries)
}
