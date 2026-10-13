package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

func (e *env) rightClick(x, y int) {
	e.t.Helper()
	e.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
}

func (e *env) favNames() []string {
	var out []string
	for _, p := range e.m.deps.Cfg.Sidebar.Favorites {
		out = append(out, p.Name)
	}
	return out
}

func TestBookmarkKeyAddsTheCurrentFolderFromTheSidebarToo(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key("tab")
	e.key("b")
	if got := e.favNames(); len(got) != 3 || got[2] != "proj" {
		t.Fatalf("favorites %v", got)
	}
	if cfg := e.savedConfig(path); len(cfg.Sidebar.Favorites) != 3 {
		t.Fatalf("saved %v", cfg.Sidebar.Favorites)
	}
	e.key("b") // again: removes it
	if len(e.favNames()) != 2 {
		t.Fatalf("toggled off: %v", e.favNames())
	}
}

func TestSidebarKeysRemoveRenameAndReorderAFavorite(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key("tab")
	e.key("down") // Home (Recents is first)
	e.key("J")
	if got := e.favNames(); got[0] != "Gone" || got[1] != "Home" {
		t.Fatalf("J moves down: %v", got)
	}
	e.key("K")
	if got := e.favNames(); got[0] != "Home" {
		t.Fatalf("K moves up: %v", got)
	}
	e.key("f2")
	for range 4 {
		e.key("backspace")
	}
	for _, r := range "Casa" {
		e.key(string(r))
	}
	e.key("enter")
	if got := e.favNames(); got[0] != "Casa" {
		t.Fatalf("renamed: %v", got)
	}
	e.mustHave("Casa")
	e.key("delete")
	if got := e.favNames(); len(got) != 1 || got[0] != "Gone" {
		t.Fatalf("removed: %v", got)
	}
	if cfg := e.savedConfig(path); len(cfg.Sidebar.Favorites) != 1 {
		t.Fatalf("saved %v", cfg.Sidebar.Favorites)
	}
}

func TestRightClickOnAFavoriteOffersRemoveAndOnAFolderOffersAdd(t *testing.T) {
	e := newEnv(t)
	e.withConfigFile()
	x, y := e.find("Gone")
	e.rightClick(x, y)
	e.mustHave("Remove from Favorites")
	e.clickText("Remove from Favorites")
	if got := e.favNames(); len(got) != 1 || got[0] != "Home" {
		t.Fatalf("removed by menu: %v", got)
	}
	x, y = e.find("alpha")
	e.rightClick(x, y)
	e.mustHave("Add to Favorites")
	e.clickText("Add to Favorites")
	if got := e.favNames(); got[len(got)-1] != "alpha" {
		t.Fatalf("added by menu: %v", got)
	}
}

func TestDefaultSidebarHasTheTrashAsALocation(t *testing.T) {
	d := config.Defaults("/home/u")
	last := d.Sidebar.Locations[len(d.Sidebar.Locations)-1]
	if last.Name != "Trash" || last.Path != filepath.Join("/home/u", ".Trash") {
		t.Fatalf("locations %v", d.Sidebar.Locations)
	}
	if localName("Trash") == "" {
		t.Fatal("Trash has a local name")
	}
}

func TestQuestionMarkOpensHelpThatExplainsFavorites(t *testing.T) {
	e := newEnv(t)
	e.key("?")
	e.mustHave("Favorites")
	e.mustHave("Add the folder to Favorites")
	e.key("esc")
	if e.has("Add the folder to Favorites") {
		t.Fatal("esc closes the help")
	}
}

func TestSettingsKeybindingsRowTurnsTheVimLettersOff(t *testing.T) {
	e := newEnv(t)
	path := e.withConfigFile()
	e.key("j")
	if e.m.sel != 1 {
		t.Fatalf("dual: j moves down, sel=%d", e.m.sel)
	}
	e.m.set = settingsState{open: true, sel: setKeys}
	e.mustHave("Lazy + Vim (hjkl)")
	e.key("right")
	e.mustHave("Lazy ")
	e.key("esc")
	e.key("j")
	if e.m.sel != 1 {
		t.Fatalf("lazy: j does nothing, sel=%d", e.m.sel)
	}
	if cfg := e.savedConfig(path); cfg.KeybindingMode != "lazy" {
		t.Fatalf("saved %q", cfg.KeybindingMode)
	}
}
