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

// copyXattrs read each attribute whole: a resource fork of hundreds of MB, which a tiny zip with an AppleDouble entry
// restores, sat in memory at once and could not be cancelled. An attribute above maxXattr is left behind.
func TestAHugeAttributeIsLeftBehind(t *testing.T) {
	w := newWorld(t)
	w.put("src/a.txt", "a")
	if err := unix.Setxattr(w.path("src/a.txt"), "user.lazyfinder.test", []byte("v"), 0); err != nil {
		t.Skipf("this filesystem keeps no extended attributes: %v", err)
	}
	old := fgetxattr
	fgetxattr = func(fd int, name string, dest []byte) (int, error) {
		if dest == nil {
			return 1 << 40, nil // a terabyte: reading it would never end
		}
		return old(fd, name, dest)
	}
	t.Cleanup(func() { fgetxattr = old })
	w.must(copyNew(context.Background(), w.path("src/a.txt"), w.path("dst/a.txt"), nil))
	if _, err := unix.Getxattr(w.path("dst/a.txt"), "user.lazyfinder.test", nil); err == nil {
		t.Fatal("an attribute above the limit was copied")
	}
}

// A move across volumes copies, then removes the source: an attribute the copy left behind (a resource fork above the
// limit, one the destination refused) was lost for good with it. A move keeps every attribute or does not happen.
func TestAMoveThatCannotKeepAnAttributeKeepsTheSource(t *testing.T) {
	w := newWorld(t)
	w.put("src/a.txt", "a")
	if err := unix.Setxattr(w.path("src/a.txt"), "user.lazyfinder.test", []byte("v"), 0); err != nil {
		t.Skipf("this filesystem keeps no extended attributes: %v", err)
	}
	old := fgetxattr
	fgetxattr = func(fd int, name string, dest []byte) (int, error) {
		if dest == nil {
			return maxXattr + 1, nil
		}
		return old(fd, name, dest)
	}
	t.Cleanup(func() { fgetxattr = old })
	if err := copyNew(wholeAttrs(context.Background()), w.path("src/a.txt"), w.path("dst/a.txt"), nil); err == nil {
		t.Fatal("a move that leaves an attribute behind must fail")
	}
	if _, err := os.Lstat(w.path("dst/a.txt")); !os.IsNotExist(err) {
		t.Fatalf("the copy stayed: %v", err)
	}
	if got := w.read("src/a.txt"); got != "a" {
		t.Fatal("the source is gone")
	}
}

// Esc while the attributes were copied ended copyXattrs quietly, so the copy counted as done and a move removed its
// source. The copy is cancelled like any other.
func TestEscWhileCopyingAttributesCancels(t *testing.T) {
	w := newWorld(t)
	w.put("src/a.txt", "a")
	if err := unix.Setxattr(w.path("src/a.txt"), "user.lazyfinder.test", []byte("v"), 0); err != nil {
		t.Skipf("this filesystem keeps no extended attributes: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	old := fgetxattr
	fgetxattr = func(fd int, name string, dest []byte) (int, error) {
		cancel()
		return old(fd, name, dest)
	}
	t.Cleanup(func() { fgetxattr = old })
	if err := copyNew(ctx, w.path("src/a.txt"), w.path("dst/a.txt"), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("copy = %v, want cancelled", err)
	}
	if _, err := os.Lstat(w.path("dst/a.txt")); !os.IsNotExist(err) {
		t.Fatalf("the cancelled copy stayed: %v", err)
	}
}

// An attribute the source would not give (EPERM on reading it) was taken for one the destination cannot hold, and a
// move left it behind. Only the destination's refusal is skipped.
func TestAMoveThatCannotReadAnAttributeKeepsTheSource(t *testing.T) {
	w := newWorld(t)
	w.put("src/a.txt", "a")
	if err := unix.Setxattr(w.path("src/a.txt"), "user.lazyfinder.test", []byte("v"), 0); err != nil {
		t.Skipf("this filesystem keeps no extended attributes: %v", err)
	}
	old := fgetxattr
	fgetxattr = func(int, string, []byte) (int, error) { return 0, unix.EPERM }
	t.Cleanup(func() { fgetxattr = old })
	if err := copyNew(wholeAttrs(context.Background()), w.path("src/a.txt"), w.path("dst/a.txt"), nil); err == nil {
		t.Fatal("a move that cannot read an attribute must fail")
	}
	if got := w.read("src/a.txt"); got != "a" {
		t.Fatal("the source is gone")
	}
}
