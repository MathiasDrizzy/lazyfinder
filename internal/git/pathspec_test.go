package git

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Paths went to git as pathspecs, where *, ? and [..] are patterns: discarding the untracked file "[ab].txt" ran
// git clean on a.txt and b.txt too, deleting them for good. A path names one file.
func TestAPathWithPatternCharactersNamesOnlyThatFile(t *testing.T) {
	repo := fixture(t)
	for _, n := range []string{"a.txt", "b.txt", "[ab].txt"} {
		put(t, filepath.Join(repo, n), "x\n")
	}
	if err := Discard(repo, Change{Path: "[ab].txt", Code: Untracked}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "[ab].txt")); !os.IsNotExist(err) {
		t.Fatalf("[ab].txt was not discarded: %v", err)
	}
	for _, n := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(repo, n)); err != nil {
			t.Fatalf("discarding [ab].txt took %s too: %v", n, err)
		}
	}
	if err := Stage(repo, "[ab].txt"); err == nil {
		t.Fatal("staging a file that is gone must fail, not match a.txt and b.txt")
	}
}

// The diff of an untracked named pipe ran git diff --no-index on it, which waits for a writer: the UI hung until the
// 15 s timeout killed git, on every try. Only a regular file or a link is compared.
func TestTheDiffOfAnUntrackedPipeDoesNotWait(t *testing.T) {
	repo := fixture(t)
	if err := syscall.Mkfifo(filepath.Join(repo, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := Diff(repo, Change{Path: "pipe", Code: Untracked}, false); err == nil {
		t.Fatal("the diff of a pipe must say it cannot be shown")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("the diff of a pipe took %v", d)
	}
}
