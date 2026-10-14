package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// The recent files, saved searches and tags say what you opened and where: only you may read them.
func TestStateFilesAreReadableOnlyByTheUser(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "lazyfinder")
	path := filepath.Join(dir, recentsFile)
	saveState(path, []string{"/Users/someone/Documents/secret plan.txt"})
	for p, want := range map[string]os.FileMode{dir: 0o700, path: 0o600} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s has mode %v, want %v", filepath.Base(p), got, want)
		}
	}
}
