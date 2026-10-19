package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// xdgWorld is newWorld with an XDG trash (…/Trash/files), the one Linux desktops show.
func xdgWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	w.trash = filepath.Join(w.root, "share", "Trash", "files")
	w.ops = New(w.trash)
	return w
}

// A desktop shows an item of its trash, and puts it back, only with the .trashinfo that says where it came from.
func TestTheXDGTrashGetsAnInfoFileForEachItem(t *testing.T) {
	w := xdgWorld(t)
	w.put("src/my file%.txt", "odd")
	w.must(w.ops.Trash([]string{w.path("src/my file%.txt")}))
	if !w.exists("share/Trash/files/my file%.txt") {
		t.Fatal("the file is not in Trash/files")
	}
	info := w.read("share/Trash/info/my file%.txt.trashinfo")
	lines := strings.Split(strings.TrimSpace(info), "\n")
	want := "Path=" + strings.ReplaceAll(strings.ReplaceAll(w.path("src/my file%.txt"), "%", "%25"), " ", "%20")
	if len(lines) != 3 || lines[0] != "[Trash Info]" || lines[1] != want || !strings.HasPrefix(lines[2], "DeletionDate=") || len(lines[2]) != len("DeletionDate=2006-01-02T15:04:05") {
		t.Fatalf("trashinfo:\n%s\nwant %s", info, want)
	}
	if _, err := w.ops.Undo(); err != nil {
		t.Fatal(err)
	}
	if w.read("src/my file%.txt") != "odd" || w.exists("share/Trash/info/my file%.txt.trashinfo") {
		t.Fatal("the undo did not put the file back and drop its info")
	}
}

// The name of an item and of its info file are taken together: an info file left behind takes its name.
func TestAnOrphanTrashInfoKeepsItsName(t *testing.T) {
	w := xdgWorld(t)
	w.must(os.MkdirAll(w.path("share/Trash/info"), 0o700))
	w.put("share/Trash/info/a.txt.trashinfo", "[Trash Info]\nPath=/elsewhere/a.txt\nDeletionDate=2020-01-01T00:00:00\n")
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	if !w.exists("share/Trash/files/a copy.txt") || !w.exists("share/Trash/info/a copy.txt.trashinfo") {
		t.Fatal("the item did not take the next free name")
	}
	if !strings.Contains(w.read("share/Trash/info/a.txt.trashinfo"), "/elsewhere/a.txt") {
		t.Fatal("the orphan info file was overwritten")
	}
}

// Any other trash folder (the macOS ~/.Trash) gets no info files.
func TestATrashThatIsNotXDGGetsNoInfoFiles(t *testing.T) {
	w := newWorld(t)
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	if w.exists("info") || w.exists("trash/info") {
		t.Fatal("info files outside an XDG trash")
	}
}
