package config

import (
	"os"
	"path/filepath"
	"testing"
)

// config.json lists your folders and programs: a save leaves it, and a folder it creates, readable only by you.
func TestUpdateWritesAConfigOnlyTheUserCanRead(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "lazyfinder")
	path := filepath.Join(dir, "config.json")
	if err := Update(path, map[string]any{"theme": "nord"}); err != nil {
		t.Fatal(err)
	}
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
