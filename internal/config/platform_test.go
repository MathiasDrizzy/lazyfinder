package config

import (
	"strings"
	"testing"
)

// On Linux the defaults were the macOS ones: the Trash was ~/.Trash, which no Linux desktop shows, and the sidebar
// offered Applications, Macintosh HD and /Volumes, which do not exist there.
func TestTheLinuxDefaultsUseTheXDGTrashAndLinuxPlaces(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "") // the default without it; xdgdata_test.go covers it set
	c := DefaultsFor("linux", "/h")
	if c.Ops.TrashDir != "/h/.local/share/Trash/files" {
		t.Fatalf("trash_dir = %q", c.Ops.TrashDir)
	}
	for _, p := range append(append([]Place{}, c.Sidebar.Favorites...), c.Sidebar.Locations...) {
		if strings.HasPrefix(p.Path, "/Applications") || strings.HasPrefix(p.Path, "/Volumes") || p.Name == "Macintosh HD" || p.Path == "/h/.Trash" {
			t.Errorf("a macOS place on Linux: %+v", p)
		}
	}
	trash := false
	for _, p := range c.Sidebar.Locations {
		trash = trash || p.Path == c.Ops.TrashDir
	}
	if !trash {
		t.Errorf("the sidebar does not lead to the trash: %+v", c.Sidebar.Locations)
	}
}

func TestTheMacOSDefaultsStayAsTheyWere(t *testing.T) {
	c := DefaultsFor("darwin", "/h")
	if c.Ops.TrashDir != "/h/.Trash" || c.Sidebar.Locations[0].Name != "Macintosh HD" {
		t.Fatalf("trash_dir = %q, locations = %+v", c.Ops.TrashDir, c.Sidebar.Locations)
	}
}
