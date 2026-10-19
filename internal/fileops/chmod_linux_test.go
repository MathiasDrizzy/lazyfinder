package fileops

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// Without fchmodat2 the mode was set by name after checking the name was no link, so a link swapped in between (by
// someone who can write to the folder) got the new mode on the item it points to. The item checked is the one changed.
func TestALinkSwappedInBeforeTheChangeIsNotFollowed(t *testing.T) {
	saved, savedHook := fchmodat, beforeChange
	t.Cleanup(func() { fchmodat, beforeChange = saved, savedHook })
	fchmodat = func(int, string, uint32, int) error { return unix.ENOSYS }
	w := newWorld(t)
	p, target := w.path("src/a.txt"), w.path("src/target")
	w.must(os.WriteFile(target, []byte("theirs"), 0o640))
	w.must(os.Chmod(target, 0o640))
	beforeChange = func(path string) {
		w.must(os.Rename(path, path+".moved"))
		w.must(os.Symlink(target, path))
	}
	_ = chmodNoFollow(p, 0o600)
	if mode(t, target) != 0o640 {
		t.Fatalf("the mode reached the item the link points to: %o", mode(t, target))
	}
	if mode(t, p+".moved") != 0o600 {
		t.Fatalf("the item checked did not change: %o", mode(t, p+".moved"))
	}
}
