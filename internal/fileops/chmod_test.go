package fileops

import (
	"io/fs"
	"os"
	"testing"

	"golang.org/x/sys/unix"
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

// An item swapped for a link after a chmod must not let the undo change the mode of what the link points to.
func TestUndoOfAChmodDoesNotFollowALinkPutInPlace(t *testing.T) {
	w := newWorld(t)
	p, target := w.path("src/a.txt"), w.path("src/target")
	w.must(os.WriteFile(target, []byte("theirs"), 0o640))
	w.must(os.Chmod(target, 0o640))
	w.must(os.Chmod(p, 0o644))
	w.must(w.ops.Chmod([]string{p}, 0o600))
	w.must(os.Remove(p))
	w.must(os.Symlink(target, p))
	if _, err := w.ops.Undo(); err == nil {
		t.Fatal("the undo must fail on a link")
	}
	if got := mode(t, target); got != 0o640 {
		t.Fatalf("the undo changed the link's target to %o", got)
	}
}

// Where the system cannot set a mode without following links (Linux before fchmodat2, or a seccomp filter), chmod
// still works on a file and still refuses a link.
func TestChmodWithoutTheNoFollowFlag(t *testing.T) {
	for _, refusal := range []error{unix.EOPNOTSUPP, unix.EPERM, unix.ENOSYS} {
		saved := fchmodat
		fchmodat = func(int, string, uint32, int) error { return refusal }
		w := newWorld(t)
		p, target := w.path("src/a.txt"), w.path("src/target")
		w.must(os.WriteFile(target, []byte("theirs"), 0o640))
		w.must(os.Chmod(target, 0o640))
		w.must(os.Symlink(target, w.path("src/link")))
		if err := w.ops.Chmod([]string{p}, 0o600); err != nil || mode(t, p) != 0o600 {
			t.Errorf("%v: chmod of a file: %v, mode %o", refusal, err, mode(t, p))
		}
		if err := chmodNoFollow(w.path("src/link"), 0o600); err == nil || mode(t, target) != 0o640 {
			t.Errorf("%v: a link must be refused: %v, target %o", refusal, err, mode(t, target))
		}
		fchmodat = saved
	}
}
