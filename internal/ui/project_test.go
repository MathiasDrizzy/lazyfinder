package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/charmbracelet/x/ansi"
)

// projectEnv is the project sidebar mode on a git repository with one modified and one new file, 36 columns wide.
func projectEnv(t *testing.T, w int) *env {
	t.Helper()
	e := projectEnvWithoutGit(t, w)
	e.git("init", "-q", "-b", "main")
	e.git("add", "-A")
	e.git("commit", "-q", "-m", "init")
	if err := os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.root, "alpha", "fresh.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.send(tea.WindowSizeMsg{Width: w, Height: 24})
	return e
}

// projectEnvWithoutGit is the sidebar mode on the fixture folder, which is not a repository yet.
func projectEnvWithoutGit(t *testing.T, w int) *env {
	t.Helper()
	e := newEnv(t)
	d := e.m.deps
	d.Follow = true
	d.Watcher = newFakeWatcher()
	d.Cfg.Density = config.DensityComfortable // the wide rows must not reach the sidebar
	d.Cfg.Columns = []string{"name"}
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: w, Height: 24})
	return e
}

func TestProjectModeHasClickableFilesSearchAndGitTabs(t *testing.T) {
	e := projectEnv(t, 36)
	top := e.screen()[0]
	for _, tab := range []string{"Files", "Search", "Git"} {
		if !strings.Contains(top, tab) {
			t.Fatalf("tab %q missing in %q", tab, top)
		}
	}
	x, _ := e.find("Search")
	e.click(x, 0)
	if !e.m.searchOn() {
		t.Fatal("the Search tab must open the search")
	}
}

func TestProjectModeShowsTheNameInCapitalsAndTheBranchAtTheBottom(t *testing.T) {
	e := projectEnv(t, 36)
	e.mustHave("PROJ")
	bottom := e.screen()[e.m.h-2]
	if !strings.Contains(bottom, "main") {
		t.Fatalf("the branch is missing in %q", bottom)
	}
}

func TestProjectModeKeepsOneLinePerRowAndTheGitLetterAtTheRight(t *testing.T) {
	e := projectEnv(t, 36)
	_, y := e.find("notes.txt")
	_, y2 := e.find("alpha")
	if y-y2 > 6 {
		t.Fatalf("rows are not one line each: alpha at %d, notes.txt at %d", y2, y)
	}
	line := strings.TrimRight(strings.Trim(e.screen()[y], "│ "), " ")
	if !strings.HasSuffix(line, "M") {
		t.Fatalf("the git letter belongs at the right edge: %q", line)
	}
}

func TestTheGitTabListsOnlyTheChangedFiles(t *testing.T) {
	e := projectEnv(t, 36)
	x, _ := e.find("Git")
	e.click(x, 0)
	e.mustHave("notes.txt")
	e.mustHave("fresh.txt")
	e.mustNotHave("data.xlsx")
	x, _ = e.find("Files")
	e.click(x, 0)
	e.mustHave("data.xlsx")
}

func TestProjectModeFitsThirtyColumns(t *testing.T) {
	e := projectEnv(t, 30)
	e.mustNotHave("too small")
	for i, l := range e.screen() {
		if w := len([]rune(l)); w > 400 {
			t.Fatalf("line %d is absurd", i)
		}
	}
	e.mustHave("Git")
}

func TestSidebarModeStartsCollapsedAndKeepsWhatTheUserOpened(t *testing.T) {
	e := projectEnv(t, 36)
	if e.m.expanded[filepath.Join(e.root, "alpha")] {
		t.Fatal("entering the sidebar mode must not expand every folder (order 07, point M3)")
	}
	e.m.deps.Follow = false
	e.m.toggleProject()
	if len(e.m.expanded) != 0 {
		t.Fatalf("nothing opens when the mode changes: %v", e.m.expanded)
	}
}

// order 07, point M4: git is visible in the sidebar, with the color of the state on the name, a dot on the folders
// that hold changes and a counter next to the branch.
func TestGitColorsTheNamesAndMarksFoldersWithADotAndACounter(t *testing.T) {
	e := projectEnv(t, 36)
	e.key("down") // the selected row is drawn in the selection color, so the checks look at the other rows
	e.key("down")
	yellow := "38;2;249;226;175" // catppuccin-mocha yellow: modified
	green := "38;2;166;227;161"  // catppuccin-mocha green: new
	l := rawAt(e, lineOf(t, e, "notes.txt"))
	if i := strings.Index(l, "notes.txt"); !strings.Contains(l[:i], yellow) || !strings.Contains(l[i:], yellow) {
		t.Fatalf("a modified file has the name and the letter in yellow: %q", l)
	}
	l = rawAt(e, lineOf(t, e, "alpha"))
	if !strings.Contains(ansi.Strip(l), "●") || !strings.Contains(l, green) {
		t.Fatalf("a folder with a new file has a green dot: %q", l)
	}
	if foot := e.screen()[len(e.screen())-2]; !strings.Contains(foot, "2") || !strings.Contains(foot, "●") {
		t.Fatalf("the footer counts the changes: %q", foot)
	}
}
