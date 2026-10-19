package fileops

import (
	"context"
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// A copy, and so a move or a Trash across volumes, dropped the extended attributes: Finder tags, a resource fork,
// and com.apple.quarantine, so a downloaded program moved to another disk opened without Gatekeeper's question.
// Files and folders keep them.
func TestACopyKeepsExtendedAttributes(t *testing.T) {
	w := newWorld(t)
	w.must(os.MkdirAll(w.path("src/dir"), 0o755))
	w.put("src/dir/a.txt", "a")
	const name, value = "user.lazyfinder.test", "kept"
	for _, p := range []string{"src/dir", "src/dir/a.txt"} {
		if err := unix.Setxattr(w.path(p), name, []byte(value), 0); err != nil {
			if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EPERM) {
				t.Skipf("this filesystem keeps no extended attributes: %v", err)
			}
			t.Fatal(err)
		}
	}
	w.must(copyNew(context.Background(), w.path("src/dir"), w.path("dst/dir"), nil))
	for _, p := range []string{"dst/dir", "dst/dir/a.txt"} {
		buf := make([]byte, 64)
		n, err := unix.Getxattr(w.path(p), name, buf)
		if err != nil || string(buf[:n]) != value {
			t.Errorf("%s lost its extended attribute: %q, %v", p, buf[:max(n, 0)], err)
		}
	}
}

// Only three errors of setxattr were skipped: any other one (EINVAL on a partial SMB or FUSE volume, ENOSPC when the
// attribute block is full) failed the whole copy and removed it, for an attribute the files do not need.
func TestAnAttributeTheDestinationRefusesDoesNotFailTheCopy(t *testing.T) {
	w := newWorld(t)
	w.put("src/a.txt", "a")
	if err := unix.Setxattr(w.path("src/a.txt"), "user.lazyfinder.test", []byte("v"), 0); err != nil {
		t.Skipf("this filesystem keeps no extended attributes: %v", err)
	}
	old := fsetxattr
	fsetxattr = func(int, string, []byte, int) error { return unix.EINVAL }
	t.Cleanup(func() { fsetxattr = old })
	w.must(copyNew(context.Background(), w.path("src/a.txt"), w.path("dst/a.txt"), nil))
	if got := w.read("dst/a.txt"); got != "a" {
		t.Fatalf("copy = %q", got)
	}
}
