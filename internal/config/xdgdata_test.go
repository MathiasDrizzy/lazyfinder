package config

import (
	"path/filepath"
	"testing"
)

// The Linux trash was always ~/.local/share/Trash, but the XDG spec puts it in $XDG_DATA_HOME/Trash when that is set:
// with XDG_DATA_HOME=/data/xdg the desktop and trash-cli showed another folder, and what lazyfinder trashed could not
// be restored from them. A relative value is invalid by the spec and ignored.
func TestTheLinuxTrashFollowsXDGDataHome(t *testing.T) {
	home := t.TempDir()
	for value, want := range map[string]string{
		"/data/xdg": "/data/xdg/Trash/files",
		"relative":  filepath.Join(home, ".local", "share", "Trash", "files"),
		"":          filepath.Join(home, ".local", "share", "Trash", "files"),
	} {
		t.Setenv("XDG_DATA_HOME", value)
		c := DefaultsFor("linux", home)
		if c.Ops.TrashDir != want {
			t.Errorf("XDG_DATA_HOME=%q: trash %s, want %s", value, c.Ops.TrashDir, want)
		}
		if got := c.Sidebar.Locations[1].Path; got != want {
			t.Errorf("XDG_DATA_HOME=%q: sidebar Trash %s, want %s", value, got, want)
		}
	}
	t.Setenv("XDG_DATA_HOME", "/data/xdg")
	if got := DefaultsFor("darwin", home).Ops.TrashDir; got != filepath.Join(home, ".Trash") {
		t.Errorf("macOS trash %s", got)
	}
}
