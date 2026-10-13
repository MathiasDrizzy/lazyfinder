package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// followEnv is the model started the way `lazyfinder --follow <dir>` starts it.
func followEnv(t *testing.T, w, h int) (*env, *fakeWatcher) {
	t.Helper()
	e := newEnv(t)
	fw := newFakeWatcher()
	d := e.m.deps
	d.Watcher, d.Follow = fw, true
	d.Cfg.Columns = []string{"name"}
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: w, Height: h})
	return e, fw
}

func TestFollowModeHasNoSidebarAndStartsCollapsed(t *testing.T) {
	e, _ := followEnv(t, 40, 20)
	e.mustNotHave("Favorites")
	e.mustNotHave("inner.txt") // point M3: nothing opens by itself
	e.mustHave("alpha")
}

func TestFollowModeFitsANarrowTerminal(t *testing.T) {
	e, _ := followEnv(t, 28, 14)
	e.mustNotHave("too small")
	e.mustHave("notes.txt")
}

func TestEnterOnAFolderTogglesItInsteadOfLeavingTheProject(t *testing.T) {
	e, _ := followEnv(t, 40, 20)
	e.clickText("alpha")
	e.key("enter")
	e.mustHave("inner.txt")
	if e.m.dir() != e.root {
		t.Fatalf("dir = %s, the project root must stay", e.m.dir())
	}
	e.key("enter")
	e.mustNotHave("inner.txt")
}

func TestBackspaceNeverLeavesTheProjectRoot(t *testing.T) {
	e, _ := followEnv(t, 40, 20)
	e.key("backspace")
	if e.m.dir() != e.root {
		t.Fatalf("dir = %s", e.m.dir())
	}
}

func TestANewFolderOpensByItselfAndItsFilesAreMarked(t *testing.T) {
	e, _ := followEnv(t, 40, 20)
	made := filepath.Join(e.root, "agent-made")
	if err := os.MkdirAll(made, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(made, "main.rs"), []byte("fn main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.send(dirChangedMsg{dir: e.root})
	e.mustHave("agent-made")
	e.mustHave("main.rs")
	if !e.m.isFresh(made) {
		t.Fatal("the new folder is marked as new")
	}
	e.send(dirChangedMsg{dir: made})
	if !e.m.isFresh(filepath.Join(made, "main.rs")) {
		t.Fatal("what appears inside an open folder is marked as new too")
	}
}

func TestNewMarksSurviveCollapsingAndOpeningFolders(t *testing.T) {
	e, _ := followEnv(t, 40, 20)
	if err := os.WriteFile(filepath.Join(e.root, "late.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.send(dirChangedMsg{dir: e.root})
	e.clickText("alpha")
	e.key("enter")
	e.key("enter")
	if !e.m.isFresh(filepath.Join(e.root, "late.txt")) {
		t.Fatal("the marks stay for the whole session of a followed project")
	}
}
