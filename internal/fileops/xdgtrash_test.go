package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
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

// An undo that put the file back but could not drop its .trashinfo stopped there and kept the item to be put back, so
// every later undo failed on a file no longer in the Trash and the whole undo list was stuck. The item now counts as
// put back once it is; the stale .trashinfo is reported, and the next undo only tries to remove it again.
func TestAnUndoThatCannotDropTheInfoDoesNotGetStuck(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read-only folder")
	}
	w := xdgWorld(t)
	w.put("src/a.txt", "a")
	w.put("src/b.txt", "b")
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	w.must(w.ops.Trash([]string{w.path("src/b.txt")}))
	info := filepath.Join(w.root, "share", "Trash", "info")
	w.must(os.Chmod(info, 0o500))
	t.Cleanup(func() { _ = os.Chmod(info, 0o700) })
	if _, err := w.ops.Undo(); err == nil || !w.exists("src/b.txt") {
		t.Fatalf("the undo must put b back and say the info stayed: %v", err)
	}
	w.must(os.Chmod(info, 0o700))
	if _, err := w.ops.Undo(); err != nil || w.exists("share/Trash/info/b.txt.trashinfo") || !w.exists("src/b.txt") || w.exists("src/b copy.txt") {
		t.Fatalf("the next undo must only drop the stale info: %v", err)
	}
	if _, err := w.ops.Undo(); err != nil || !w.exists("src/a.txt") {
		t.Fatalf("the undo list is stuck: %v", err)
	}
}

// A name of 246 to 255 bytes, the most a file system takes, could not go to an XDG trash: its .trashinfo name is ten
// bytes longer. The name in the trash is shortened instead; the .trashinfo keeps the whole path, and undo puts the
// item back under its own name.
func TestALongNameGoesToTheXDGTrash(t *testing.T) {
	w := xdgWorld(t)
	long := strings.Repeat("é", 123) + ".txt" // 250 bytes
	w.put("src/"+long, "x")
	w.must(w.ops.Trash([]string{w.path("src/" + long)}))
	if w.exists("src/" + long) {
		t.Fatal("the file did not go to the trash")
	}
	infos, _ := os.ReadDir(filepath.Join(w.root, "share", "Trash", "info"))
	if len(infos) != 1 || len(infos[0].Name()) > 255 || !strings.HasSuffix(infos[0].Name(), ".txt.trashinfo") {
		t.Fatalf("info files: %v", infos)
	}
	if _, err := w.ops.Undo(); err != nil || w.read("src/"+long) != "x" {
		t.Fatalf("undo: %v", err)
	}
}

// "name copy.ext" of a name near 255 bytes is too long for the file system, so a duplicate failed; the stem is cut
// to make room for the suffix, at a whole character.
func TestACopyOfALongNameFits(t *testing.T) {
	long := strings.Repeat("é", 125) + ".txt" // 254 bytes
	for n := range 3 {
		c := candidate(long, n)
		if len(c) > 255 || !utf8.ValidString(c) || !strings.HasSuffix(c, ".txt") {
			t.Fatalf("candidate %d = %d bytes %q", n, len(c), c)
		}
	}
	if c := candidate("short.txt", 1); c != "short copy.txt" {
		t.Fatalf("a short name changed: %q", c)
	}
}
