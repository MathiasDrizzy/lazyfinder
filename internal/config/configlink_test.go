package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Saving a setting replaced a config.json that is a link (dotfiles kept in a repository) with a plain file, so the
// link to the user's repository was lost. The file the link points to is written instead.
func TestSavingASettingKeepsAConfigLink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "dotfiles", "config.json")
	if err := os.MkdirAll(filepath.Dir(real), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := Update(link, map[string]any{"theme": "nord"}); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config.json is no longer a link: %v", err)
	}
	data, err := os.ReadFile(real)
	if err != nil || !strings.Contains(string(data), `"nord"`) {
		t.Fatalf("the linked file was not updated: %s %v", data, err)
	}
}
