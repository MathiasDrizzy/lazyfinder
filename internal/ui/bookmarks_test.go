package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestBookmarkAddsTheSelectedFolderToFavoritesAndTogglesItOff(t *testing.T) {
	e := newEnv(t)
	e.clickText("alpha")
	e.key("b")
	e.mustHave("Bookmarked")
	favs := e.m.deps.Cfg.Sidebar.Favorites
	if last := favs[len(favs)-1]; last.Name != "alpha" || last.Path != filepath.Join(e.root, "alpha") {
		t.Fatalf("last favorite = %+v", last)
	}
	e.key("b")
	e.mustHave("Bookmark removed")
	if got := e.m.deps.Cfg.Sidebar.Favorites; len(got) != len(favs)-1 {
		t.Fatalf("favorites = %d, want %d", len(got), len(favs)-1)
	}
}

func TestBookmarkOnAFileBookmarksTheCurrentFolder(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	e.key("b")
	favs := e.m.deps.Cfg.Sidebar.Favorites
	if last := favs[len(favs)-1]; last.Path != e.root {
		t.Fatalf("last favorite = %+v, want the current folder", last)
	}
}

func TestBookmarksAreSavedWithHomeShorthand(t *testing.T) {
	e := newEnv(t)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	d := e.m.deps
	d.ConfigPath = cfgPath
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.clickText("alpha")
	e.key("b")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"path": "~/proj/alpha"`) {
		t.Fatalf("config:\n%s", data)
	}
}
