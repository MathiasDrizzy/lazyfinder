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

func (e *env) staged() []string {
	var out []string
	for _, c := range e.status().Staged {
		out = append(out, c.Path)
	}
	return out
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

func TestStageUnstageWithTheKeyAndTheButtons(t *testing.T) {
	e := gitEnv(t, 40)
	e.clickText("notes.txt")
	e.key("s")
	if got := e.staged(); len(got) != 1 || got[0] != "notes.txt" {
		t.Fatalf("s stages the selected file, staged = %v", got)
	}
	e.mustHave("Staged Changes")
	e.key("s")
	if got := e.staged(); len(got) != 0 {
		t.Fatalf("s on a staged file unstages it, staged = %v", got)
	}
	x, y := e.button("fresh.txt", scmActStage)
	e.click(x, y)
	if got := e.staged(); len(got) != 1 || got[0] != "alpha/fresh.txt" {
		t.Fatalf("the + button stages the row it is on, staged = %v", got)
	}
	x, y = e.button("fresh.txt", scmActUnstage)
	e.click(x, y)
	if got := e.staged(); len(got) != 0 {
		t.Fatalf("the − button unstages, staged = %v", got)
	}
}

func TestSectionButtonsStageAndUnstageEverything(t *testing.T) {
	e := gitEnv(t, 40)
	y := e.rowY("Changes")
	e.move(5, y)
	var hit bool
	for _, b := range e.m.scmButtons(e.m.scm.top + y - e.m.firstRow()) {
		if b.act == scmActStageAll {
			e.click(b.x, y)
			hit = true
		}
	}
	if !hit {
		t.Fatalf("the Changes header has a + button:\n%s", e.text())
	}
	if len(e.status().Unstaged) != 0 || len(e.staged()) != 2 {
		t.Fatalf("stage all: staged=%v unstaged=%v", e.staged(), e.status().Unstaged)
	}
	y = e.rowY("Staged Changes")
	e.move(5, y)
	for _, b := range e.m.scmButtons(e.m.scm.top + y - e.m.firstRow()) {
		if b.act == scmActUnstageAll {
			e.click(b.x, y)
		}
	}
	if len(e.staged()) != 0 {
		t.Fatalf("unstage all left %v", e.staged())
	}
}

func TestDiscardAsksFirstAndOnlyThenRestoresTheFile(t *testing.T) {
	e := gitEnv(t, 40)
	e.clickText("notes.txt")
	e.key("d")
	if e.m.prompt.kind != promptConfirm {
		t.Fatal("d asks before throwing work away")
	}
	e.mustHave("notes.txt")
	e.key("n")
	if b, _ := os.ReadFile(filepath.Join(e.root, "notes.txt")); string(b) != "changed\n" {
		t.Fatalf("answering no must keep the file, got %q", b)
	}
	e.key("d")
	e.key("y")
	if b, _ := os.ReadFile(filepath.Join(e.root, "notes.txt")); string(b) != "first line\nsecond line\n" {
		t.Fatalf("discard must restore the committed text, got %q", b)
	}
	e.mustNotHave("notes.txt")
}

func TestDiscardingAnUntrackedFileDeletesIt(t *testing.T) {
	e := gitEnv(t, 40)
	x, y := e.button("fresh.txt", scmActDiscard)
	e.click(x, y)
	e.key("y")
	if _, err := os.Stat(filepath.Join(e.root, "alpha", "fresh.txt")); !os.IsNotExist(err) {
		t.Fatalf("the untracked file should be gone: %v", err)
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

func TestStashShelvesTheChangesAndTheTabSaysThereIsNothing(t *testing.T) {
	e := gitEnv(t, 40)
	e.key("z")
	if e.status().Changed() != 0 {
		t.Fatal("z stashes everything")
	}
	e.mustHave("No changes")
	if l, _ := git.Stashes(e.root); len(l) != 1 {
		t.Fatalf("stashes = %v", l)
	}
}

func TestTheBranchMenuSwitchesCreatesAndWarnsAboutChanges(t *testing.T) {
	e := gitEnv(t, 60)
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
	e := gitEnv(t, 60)
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

func TestFetchPullAndPushFromTheTab(t *testing.T) {
	e := gitEnv(t, 60)
	remote := t.TempDir()
	g := func(dir string, args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@e.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	g(remote, "init", "-q", "--bare", "-b", "main")
	g(e.root, "remote", "add", "origin", remote)
	g(e.root, "push", "-q", "-u", "origin", "main")
	other := t.TempDir()
	g(other, "clone", "-q", remote, "c")
	c := filepath.Join(other, "c")
	if err := os.WriteFile(filepath.Join(c, "remote.txt"), []byte("r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g(c, "add", ".")
	g(c, "commit", "-q", "-m", "r")
	g(c, "push", "-q", "origin", "main")
	e.key("f")
	e.mustHave("↓1")
	e.key("p")
	if _, err := os.Stat(filepath.Join(e.root, "remote.txt")); err != nil {
		t.Fatalf("p pulls: %v", err)
	}
	g(e.root, "add", "notes.txt")
	g(e.root, "commit", "-q", "-m", "mine")
	e.key("r")
	e.mustHave("↑1")
	e.key("P")
	e.mustHave("↑0")
}

func TestPushWithoutUpstreamExplainsInsteadOfGuessing(t *testing.T) {
	e := gitEnv(t, 60)
	e.key("P")
	if !e.m.statusErr || !strings.Contains(e.m.status, "upstream") {
		t.Fatalf("status = %q (err=%v)", e.m.status, e.m.statusErr)
	}
}

func TestConflictsAreMarkedAndCannotBeDiscardedByMistake(t *testing.T) {
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
	e.clickText("notes.txt")
	e.key("d")
	e.key("y")
	if !e.m.statusErr {
		t.Fatal("discarding a conflicted file must say to resolve it first")
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
	if !strings.Contains(e.screen()[selY], "↶") {
		t.Fatalf("the selected row shows its buttons:\n%s", e.text())
	}
	if strings.Contains(e.screen()[freshY], "↶") {
		t.Fatalf("a row nobody points at shows its letter, not buttons:\n%s", e.text())
	}
	e.move(5, freshY)
	if !strings.Contains(e.screen()[freshY], "↶") {
		t.Fatalf("hovering shows the buttons:\n%s", e.text())
	}
	x, y := e.button("fresh.txt", scmActStage)
	e.move(x, y)
	if !strings.Contains(e.screen()[e.m.h-1], "Stage") {
		t.Fatalf("the bottom row names the button:\n%s", e.text())
	}
}

func TestTheStashesSectionListsPopsAndDropsWithAConfirmation(t *testing.T) {
	e := gitEnv(t, 50)
	e.key("z")
	e.mustHave("Stashes")
	e.mustNotHave("WIP on main") // folded until opened
	e.clickText("Stashes")
	e.mustHave("WIP on main")
	x, y := e.button("WIP on main", scmActDrop)
	e.click(x, y)
	if e.m.prompt.kind != promptConfirm {
		t.Fatal("dropping a stash loses it for good: it asks first")
	}
	e.key("n")
	if l, _ := git.Stashes(e.root); len(l) != 1 {
		t.Fatalf("answering no must keep it: %v", l)
	}
	x, y = e.button("WIP on main", scmActPop)
	e.click(x, y)
	if l, _ := git.Stashes(e.root); len(l) != 0 {
		t.Fatalf("pop removes the stash: %v", l)
	}
	if e.status().Changed() == 0 {
		t.Fatal("pop brings the changes back")
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

// R2 of cerebro's review: a detached HEAD says where it is, like git and the herdr sidebar.
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

// R4: with a conflict, discarding or staging everything names the files in conflict and waits for a yes.
func TestDiscardAllAndStageAllNameTheConflictsFirst(t *testing.T) {
	for _, tc := range []struct{ key, word string }{{"d", "Discard"}, {"s", "Stage"}} {
		e := projectEnv(t, 80)
		e.git("stash", "-u")
		e.git("checkout", "-q", "-b", "side")
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("side\n"), 0o644)
		e.git("commit", "-q", "-am", "s")
		e.git("checkout", "-q", "main")
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("main\n"), 0o644)
		e.git("commit", "-q", "-am", "m")
		_ = e.gitMayFail("merge", "side")
		_ = os.WriteFile(filepath.Join(e.root, "loose.txt"), []byte("x\n"), 0o644)
		e.clickTab(tabGit)
		e.clickText("Changes")
		e.key(tc.key)
		e.mustHave(tc.word)
		if ask := e.m.prompt.ask; e.m.prompt.kind != promptConfirm || !strings.Contains(ask, "conflict") || !strings.Contains(ask, "notes.txt") {
			t.Fatalf("%s with a conflict must ask first and name it: %+v", tc.word, e.m.prompt.ask)
		}
		e.key("n")
		if got := e.staged(); len(got) != 0 {
			t.Fatalf("no answers yes, nothing changes: staged = %v", got)
		}
		if st := e.status(); len(st.Unstaged) < 2 {
			t.Fatalf("no answers yes, nothing is discarded: %+v", st.Unstaged)
		}
	}
}
