package fileops

import (
	"io/fs"
	"os"
	"testing"
)

func mode(t *testing.T, p string) fs.FileMode {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// Chmod sets the permission bits of each item, not of what a folder holds, and an undo puts the old ones back.
func TestChmodSetsThePermissionsAndUndoRestoresThem(t *testing.T) {
	w := newWorld(t)
	a, dir := w.path("src/a.txt"), w.path("src")
	w.must(os.Chmod(a, 0o644))
	w.must(os.Chmod(dir, 0o755))
	o := New(w.path("trash"))
	if err := o.Chmod([]string{a, dir}, 0o700); err != nil {
		t.Fatal(err)
	}
	if mode(t, a) != 0o700 || mode(t, dir) != 0o700 {
		t.Fatalf("modes %v %v, want 0700", mode(t, a), mode(t, dir))
	}
	if _, err := o.Undo(); err != nil {
		t.Fatal(err)
	}
	if mode(t, a) != 0o644 || mode(t, dir) != 0o755 {
		t.Fatalf("undo left %v %v", mode(t, a), mode(t, dir))
	}
}

// A link is refused: chmod would change the item it points to, which may be anywhere.
func TestChmodRefusesALink(t *testing.T) {
	w := newWorld(t)
	link := w.path("src/link")
	w.must(os.Symlink(w.path("src/a.txt"), link))
	before := mode(t, w.path("src/a.txt"))
	if err := New(w.path("trash")).Chmod([]string{link}, 0o777); err == nil {
		t.Fatal("a link was changed")
	}
	if mode(t, w.path("src/a.txt")) != before {
		t.Fatal("the target of the link changed")
	}
}

func TestParseModeTakesOctalPermissionsOnly(t *testing.T) {
	for in, want := range map[string]fs.FileMode{"644": 0o644, "0755": 0o755, " 700 ": 0o700} {
		if got, err := ParseMode(in); err != nil || got != want {
			t.Errorf("ParseMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "8", "64", "6444", "4755", "rwx", "-1", "77777"} {
		if _, err := ParseMode(in); err == nil {
			t.Errorf("ParseMode(%q) took it", in)
		}
	}
}
