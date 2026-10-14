package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
)

// windowGitEnv is the normal window on the repository of projectEnv.
func windowGitEnv(t *testing.T) *env {
	t.Helper()
	e := projectEnv(t, 60)
	e.m.toggleProject()
	e.key("r")
	return e
}

// gitEnv is the sidebar mode on a repository (notes.txt modified, alpha/fresh.txt new), with the Git tab open.
func gitEnv(t *testing.T, w int) *env {
	t.Helper()
	e := projectEnv(t, w)
	e.clickTab(tabGit)
	return e
}

func (e *env) status() git.Status {
	e.t.Helper()
	st, ok, err := git.Read(e.root)
	if err != nil || !ok {
		e.t.Fatalf("git.Read: %v %v", ok, err)
	}
	return st
}

// rowY is the screen line of the row of the Git tab that shows name.
func (e *env) rowY(name string) int {
	e.t.Helper()
	_, y := e.find(name)
	return y
}

// button is the cell of a row's button, found by asking the model where it drew it.
func (e *env) button(name string, act scmAct) (x, y int) {
	e.t.Helper()
	y = e.rowY(name)
	e.move(5, y) // the pointer on the row shows its buttons
	rows := e.m.scmRows()
	idx := e.m.scm.top + y - e.m.firstRow()
	if idx < 0 || idx >= len(rows) {
		e.t.Fatalf("no row at y=%d", y)
	}
	for _, b := range e.m.scmButtons(idx) {
		if b.act == act {
			return b.x, y
		}
	}
	e.t.Fatalf("row %q has no button %v: %+v\n%s", name, act, e.m.scmButtons(idx), e.text())
	return 0, 0
}

func TestTheGitTabSplitsStagedFromChangesWithCounts(t *testing.T) {
	e := gitEnv(t, 40)
	e.mustHave("Changes")
	e.mustHave("notes.txt")
	e.mustHave("fresh.txt")
	e.mustNotHave("Staged Changes")
	if err := git.Stage(e.root, "notes.txt"); err != nil {
		t.Fatal(err)
	}
	e.key("r")
	e.mustHave("Staged Changes")
	if e.rowY("Staged Changes") > e.rowY("notes.txt") || e.rowY("notes.txt") > e.rowY("fresh.txt") {
		t.Fatalf("staged section first, then the changes:\n%s", e.text())
	}
}

func TestEnterShowsTheDiffAndEscClosesIt(t *testing.T) {
	e := gitEnv(t, 80)
	e.clickText("notes.txt")
	e.key("enter")
	e.mustHave("+changed")
	e.mustHave("-first line")
	e.key("esc")
	e.mustNotHave("+changed")
}

// The branch menu is the window's (the sidebar only looks).
func TestTheBranchMenuSwitchesCreatesAndWarnsAboutChanges(t *testing.T) {
	e := windowGitEnv(t)
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@e.com"}, args...)...)
		cmd.Dir = e.root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("branch", "feature/login")
	e.key("B")
	e.mustHave("feature/login")
	e.mustHave("main")
	e.mustHave("2 uncommitted changes")
	typeText(e, "login")
	for _, it := range e.m.gmItems() {
		if it.label == "main" {
			t.Fatal("the query login must filter main out")
		}
	}
	e.key("enter")
	if e.status().Branch != "feature/login" {
		t.Fatalf("branch = %s", e.status().Branch)
	}
	e.key("B")
	typeText(e, "brand-new")
	e.mustHave("Create branch")
	e.key("enter")
	if e.status().Branch != "brand-new" {
		t.Fatalf("branch = %s", e.status().Branch)
	}
	e.key("B")
	typeText(e, "bad name")
	e.mustNotHave("Create branch")
	e.key("esc")
}

func TestTheBranchMenuRefusesToLoseWorkAndSaysWhy(t *testing.T) {
	e := windowGitEnv(t)
	// notes.txt differs on "other" and is modified here: git must refuse the switch
	g := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@e.com"}, args...)...)
		cmd.Dir = e.root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	g("stash", "-u")
	g("checkout", "-q", "-b", "other")
	if err := os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g("commit", "-q", "-am", "o")
	g("checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.key("r")
	e.key("B")
	typeText(e, "other")
	e.key("enter")
	if e.status().Branch != "main" {
		t.Fatalf("the switch must not happen, branch = %s", e.status().Branch)
	}
	if b, _ := os.ReadFile(filepath.Join(e.root, "notes.txt")); string(b) != "mine\n" {
		t.Fatalf("work was lost: %q", b)
	}
	if !e.m.statusErr || !strings.Contains(e.m.status, "notes.txt") {
		t.Fatalf("the status line should say which file blocks the switch, got %q", e.m.status)
	}
}

func TestConflictsAreMarked(t *testing.T) {
	e := gitEnv(t, 60)
	g := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@e.com"}, args...)...)
		cmd.Dir = e.root
		_ = cmd.Run()
	}
	g("stash", "-u")
	g("checkout", "-q", "-b", "side")
	_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("side\n"), 0o644)
	g("commit", "-q", "-am", "s")
	g("checkout", "-q", "main")
	_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("main\n"), 0o644)
	g("commit", "-q", "-am", "m")
	g("merge", "side")
	e.key("r")
	if !strings.Contains(e.row("notes.txt"), "!") {
		t.Fatalf("a conflict wears the ! letter:\n%s", e.text())
	}
}

func TestOutsideARepositoryTheGitTabSaysSo(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.Follow = true
	d.Watcher = newFakeWatcher()
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 40, Height: 24})
	e.clickTab(tabGit)
	e.mustHave("Not a git repository")
}

func TestButtonsShowOnlyOnTheSelectedOrHoveredRowAndHaveTooltips(t *testing.T) {
	e := gitEnv(t, 40)
	e.clickText("notes.txt")
	selY := e.rowY("notes.txt")
	freshY := e.rowY("fresh.txt")
	if !strings.Contains(e.screen()[selY], "↗") {
		t.Fatalf("the selected row shows its button:\n%s", e.text())
	}
	if strings.Contains(e.screen()[freshY], "↗") {
		t.Fatalf("a row nobody points at shows its letter, not buttons:\n%s", e.text())
	}
	e.move(5, freshY)
	if !strings.Contains(e.screen()[freshY], "↗") {
		t.Fatalf("hovering shows the button:\n%s", e.text())
	}
	x, y := e.button("fresh.txt", scmActOpen)
	e.move(x, y)
	if !strings.Contains(e.screen()[e.m.h-1], "Open changes") {
		t.Fatalf("the bottom row names the button:\n%s", e.text())
	}
}

func TestTheOpenButtonShowsTheDiffAndOShortcutOpensTheFile(t *testing.T) {
	e := gitEnv(t, 80)
	x, y := e.button("notes.txt", scmActOpen)
	e.click(x, y)
	e.mustHave("+changed")
	e.key("esc")
	e.key("o")
	if len(e.run.calls) == 0 {
		t.Fatalf("o opens the file with its program, calls = %v", e.run.calls)
	}
}

// order 07, point O3: what the Git tab does is also in the window, through the right-click menu of a changed file.
func TestRightClickOnAChangedFileOffersTheGitActions(t *testing.T) {
	e := newEnv(t)
	repo(t, e)
	e.key("r")
	x, y := e.find("notes.txt")
	e.rightClick(x, y)
	for _, want := range []string{"Show changes", "Stage", "Discard changes", "Open with"} {
		e.mustHave(want)
	}
	e.clickText("Stage")
	if got := e.stagedNames(); len(got) != 1 || got[0] != "notes.txt" {
		t.Fatalf("staged = %v", got)
	}
	x, y = e.find("notes.txt")
	e.rightClick(x, y)
	e.mustHave("Unstage")
	e.mustNotHave("Discard changes")
	e.clickText("Show staged changes")
	e.mustHave("+changed")
}

func TestRightClickOnACleanFileKeepsTheOpenWithMenu(t *testing.T) {
	e := newEnv(t)
	repo(t, e)
	e.key("r")
	x, y := e.find("data.xlsx")
	e.rightClick(x, y)
	e.mustNotHave("Stage")
	if !e.m.menu.open {
		t.Fatal("a file with no changes opens the Open with menu as before")
	}
}

func (e *env) stagedNames() []string {
	st, _, _ := git.Read(e.root)
	var out []string
	for _, c := range st.Staged {
		out = append(out, c.Path)
	}
	return out
}

func TestTheCounterOfTheGitTabCountsItsOwnRows(t *testing.T) {
	e := projectEnv(t, 50)
	e.clickTab(tabGit)
	rows := len(e.m.scmRows())
	// the sidebar draws no frame (herdr frames the pane), so the counter is read where the frame would put it
	if got, want := e.m.counter(), fmt.Sprintf("1 of %d", rows); got != want {
		t.Fatalf("counter = %q, want %q", got, want)
	}
	e.key("down")
	if got, want := e.m.counter(), fmt.Sprintf("2 of %d", rows); got != want {
		t.Fatalf("counter = %q, want %q", got, want)
	}
}

// R2 of the review: a detached HEAD says where it is, like git and the herdr sidebar.
func TestADetachedHeadShowsTheShortHash(t *testing.T) {
	e := gitEnv(t, 60)
	e.git("checkout", "-q", "--detach")
	e.key("r")
	out, err := exec.Command("git", "-C", e.root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	e.mustHave("HEAD (detached at " + strings.TrimSpace(string(out)) + ")")
}

// R3: outside a repository the Git tab has no git buttons to click.
func TestOutsideARepositoryTheGitTabHasNoGitButtons(t *testing.T) {
	e := projectEnvWithoutGit(t, 50)
	e.clickTab(tabGit)
	e.move(10, 6)
	if n := e.m.actionsShown(); n != 0 {
		t.Fatalf("no git buttons outside a repository, got %d", n)
	}
	for _, g := range []string{"↻", "↓", "↑"} {
		if strings.Contains(e.screen()[e.m.headerRow()], g) {
			t.Fatalf("the header shows %q outside a repository: %q", g, e.screen()[e.m.headerRow()])
		}
	}
}
