package fileops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// swapForLink renames path away and puts a link to target in its place, as someone with write access to a shared
// folder could while a copy runs.
func swapForLink(t *testing.T, path, target string) {
	t.Helper()
	if err := os.Rename(path, path+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

// A copied folder got its mode by path once its content was copied, so a folder swapped for a link in that time
// handed the source's mode to whatever the link named: a 0777 source opened ~/.ssh to everyone. The mode goes to the
// folder the copy made.
func TestACopyGivesItsModeOnlyToTheFolderItMade(t *testing.T) {
	w := newWorld(t)
	w.must(os.MkdirAll(w.path("src/open"), 0o755))
	w.put("src/open/big.bin", "data")
	w.must(os.Chmod(w.path("src/open"), 0o777))
	victim := w.path("private")
	w.must(os.Mkdir(victim, 0o700))
	dst := w.path("dst/open")
	swapped := false
	err := copyNew(context.Background(), w.path("src/open"), dst, func(int64) {
		if !swapped {
			swapped = true
			swapForLink(t, dst, victim)
		}
	})
	w.must(err)
	if fi, err := os.Stat(victim); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the folder behind the link changed: %v, %v", fi.Mode(), err)
	}
	if fi, err := os.Lstat(dst + ".moved"); err != nil || fi.Mode().Perm() != 0o777 {
		t.Fatalf("the folder the copy made did not get the mode: %v", err)
	}
}

// The items of a folder were opened by path one by one, so a source folder swapped for a link halfway made the copy
// read the rest from wherever the link pointed, a folder with keys in it. The copy reads the folder it started on.
func TestACopyReadsTheFolderItStartedOn(t *testing.T) {
	w := newWorld(t)
	w.must(os.MkdirAll(w.path("src/shared"), 0o755))
	w.must(os.MkdirAll(w.path("secret"), 0o755))
	w.put("src/shared/1.txt", "one")
	w.put("src/shared/2.txt", "two")
	w.put("secret/1.txt", "SECRET")
	w.put("secret/2.txt", "SECRET")
	src := w.path("src/shared")
	swapped := false
	err := copyNew(context.Background(), src, w.path("dst/shared"), func(int64) {
		if !swapped {
			swapped = true
			swapForLink(t, src, w.path("secret"))
		}
	})
	w.must(err)
	for _, n := range []string{"1.txt", "2.txt"} {
		b, err := os.ReadFile(filepath.Join(w.path("dst/shared"), n))
		if err != nil || string(b) == "SECRET" {
			t.Fatalf("%s: %q, %v: the copy followed the link swapped in for its source", n, b, err)
		}
	}
}

// A copy that failed on its source before it made anything removed what stood at dst all the same: an unreadable
// source and a name someone else took in the meantime lost their file for good. Only what the copy made goes.
func TestACopyThatFailsOnItsSourceLeavesWhatIsAtTheDestination(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a file with no permissions")
	}
	w := newWorld(t)
	w.must(os.MkdirAll(w.path("src/locked"), 0o755))
	w.put("src/locked.txt", "x")
	w.must(os.Chmod(w.path("src/locked.txt"), 0))
	w.must(os.Chmod(w.path("src/locked"), 0))
	t.Cleanup(func() { _ = os.Chmod(w.path("src/locked"), 0o755) })
	for _, n := range []string{"locked.txt", "locked"} {
		w.put("dst/"+n, "theirs")
		if err := copyNew(context.Background(), w.path("src/"+n), w.path("dst/"+n), nil); err == nil {
			t.Fatalf("%s: copying an unreadable source must fail", n)
		}
		if got := w.read("dst/" + n); got != "theirs" {
			t.Fatalf("%s: the file at the destination was removed", n)
		}
	}
}
