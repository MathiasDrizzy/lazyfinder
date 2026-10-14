package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func names(cs []Change) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Path)
	}
	return out
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestReadSplitsStagedFromUnstaged(t *testing.T) {
	dir := fixture(t)
	st, ok, err := Read(dir)
	if err != nil || !ok {
		t.Fatalf("Read: %v %v", ok, err)
	}
	if got := names(st.Staged); len(got) != 1 || got[0] != "staged.txt" {
		t.Fatalf("staged = %v, want [staged.txt]", got)
	}
	un := names(st.Unstaged)
	for _, want := range []string{"edited.txt", "sub/deep.txt", "loose.txt", "fresh/a.txt"} {
		if !has(un, want) {
			t.Errorf("unstaged %v lacks %s", un, want)
		}
	}
	if has(un, "run.log") || has(un, "build/out.bin") {
		t.Errorf("ignored files must not be listed as changes: %v", un)
	}
}

func TestStageAndUnstageMoveAFileBetweenTheLists(t *testing.T) {
	dir := fixture(t)
	if err := Stage(dir, "edited.txt"); err != nil {
		t.Fatal(err)
	}
	st, _, _ := Read(dir)
	if !has(names(st.Staged), "edited.txt") || has(names(st.Unstaged), "edited.txt") {
		t.Fatalf("after Stage: staged=%v unstaged=%v", names(st.Staged), names(st.Unstaged))
	}
	if err := Unstage(dir, "edited.txt"); err != nil {
		t.Fatal(err)
	}
	st, _, _ = Read(dir)
	if has(names(st.Staged), "edited.txt") || !has(names(st.Unstaged), "edited.txt") {
		t.Fatalf("after Unstage: staged=%v unstaged=%v", names(st.Staged), names(st.Unstaged))
	}
}

func TestStageAllAndUnstageAll(t *testing.T) {
	dir := fixture(t)
	if err := StageAll(dir); err != nil {
		t.Fatal(err)
	}
	st, _, _ := Read(dir)
	if len(st.Unstaged) != 0 || len(st.Staged) < 4 {
		t.Fatalf("StageAll: staged=%v unstaged=%v", names(st.Staged), names(st.Unstaged))
	}
	if err := UnstageAll(dir); err != nil {
		t.Fatal(err)
	}
	st, _, _ = Read(dir)
	if len(st.Staged) != 0 {
		t.Fatalf("UnstageAll left %v", names(st.Staged))
	}
}

func TestUnstageWorksBeforeTheFirstCommit(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	run(t, dir, "init", "-q", "-b", "main")
	put(t, filepath.Join(dir, "a.txt"), "x\n")
	if err := Stage(dir, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := Unstage(dir, "a.txt"); err != nil {
		t.Fatalf("Unstage on an unborn branch: %v", err)
	}
	st, _, _ := Read(dir)
	if len(st.Staged) != 0 {
		t.Fatalf("still staged: %v", names(st.Staged))
	}
}

func TestDiscardRestoresATrackedFileAndRemovesAnUntrackedOne(t *testing.T) {
	dir := fixture(t)
	if err := Discard(dir, Change{Path: "edited.txt", Code: Modified}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "edited.txt"))
	if string(b) != "one\n" {
		t.Fatalf("edited.txt = %q, want the committed content", b)
	}
	if err := Discard(dir, Change{Path: "loose.txt", Code: Untracked}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "loose.txt")); !os.IsNotExist(err) {
		t.Fatalf("loose.txt still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "run.log")); err != nil {
		t.Fatalf("an ignored file must survive a discard: %v", err)
	}
}

func TestDiscardNeverEscapesTheRepository(t *testing.T) {
	dir := fixture(t)
	outside := filepath.Join(filepath.Dir(dir), "victim.txt")
	put(t, outside, "keep\n")
	t.Cleanup(func() { os.Remove(outside) })
	if err := Discard(dir, Change{Path: "../victim.txt", Code: Untracked}); err == nil {
		t.Fatal("a path outside the repository must be refused")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("the file outside was touched: %v", err)
	}
}

func TestDiffShowsTheWorkTreeAndTheIndexSides(t *testing.T) {
	dir := fixture(t)
	d, err := Diff(dir, Change{Path: "edited.txt", Code: Modified}, false)
	if err != nil || !strings.Contains(d, "-one") || !strings.Contains(d, "+two") {
		t.Fatalf("worktree diff = %q, %v", d, err)
	}
	d, err = Diff(dir, Change{Path: "staged.txt", Code: Added}, true)
	if err != nil || !strings.Contains(d, "+new") {
		t.Fatalf("staged diff = %q, %v", d, err)
	}
	d, err = Diff(dir, Change{Path: "loose.txt", Code: Untracked}, false)
	if err != nil || !strings.Contains(d, "+x") {
		t.Fatalf("untracked diff = %q, %v", d, err)
	}
}

func TestBranchesListsLocalAndMarksTheCurrent(t *testing.T) {
	dir := fixture(t)
	run(t, dir, "branch", "feature/a")
	bs, err := Branches(dir)
	if err != nil {
		t.Fatal(err)
	}
	var cur, other bool
	for _, b := range bs {
		if b.Name == "main" && b.Current {
			cur = true
		}
		if b.Name == "feature/a" && !b.Current {
			other = true
		}
	}
	if !cur || !other {
		t.Fatalf("branches = %+v", bs)
	}
}

func TestCheckoutSwitchesAndRefusesToLoseWork(t *testing.T) {
	dir := fixture(t)
	run(t, dir, "branch", "other")
	if err := Checkout(dir, "other"); err != nil {
		t.Fatalf("a switch that keeps the changes must work: %v", err)
	}
	st, _, _ := Read(dir)
	if st.Branch != "other" {
		t.Fatalf("branch = %s", st.Branch)
	}
	// a branch where edited.txt differs: git refuses, and the work stays
	run(t, dir, "stash", "-u", "-q")
	run(t, dir, "checkout", "-q", "-b", "diverged")
	put(t, filepath.Join(dir, "edited.txt"), "diverged\n")
	run(t, dir, "commit", "-q", "-am", "d")
	run(t, dir, "checkout", "-q", "main")
	put(t, filepath.Join(dir, "edited.txt"), "my work\n")
	if err := Checkout(dir, "diverged"); err == nil {
		t.Fatal("Checkout must refuse when it would overwrite local changes")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "edited.txt"))
	if string(b) != "my work\n" {
		t.Fatalf("local work was lost: %q", b)
	}
}

func TestCheckoutRejectsOptionLookingNames(t *testing.T) {
	dir := fixture(t)
	if err := Checkout(dir, "--orphan=x"); err == nil {
		t.Fatal("a name that looks like an option must be refused")
	}
	if err := CreateBranch(dir, "-bad"); err == nil {
		t.Fatal("an invalid branch name must be refused")
	}
	if err := CreateBranch(dir, "has space"); err == nil {
		t.Fatal("an invalid branch name must be refused")
	}
}

func TestCreateBranchSwitchesToIt(t *testing.T) {
	dir := fixture(t)
	if err := CreateBranch(dir, "topic/new"); err != nil {
		t.Fatal(err)
	}
	st, _, _ := Read(dir)
	if st.Branch != "topic/new" {
		t.Fatalf("branch = %s", st.Branch)
	}
	if err := CreateBranch(dir, "topic/new"); err == nil {
		t.Fatal("creating a branch that exists must fail")
	}
}

func TestStashPushListPopRoundTrip(t *testing.T) {
	dir := fixture(t)
	if err := StashPush(dir, "wip"); err != nil {
		t.Fatal(err)
	}
	st, _, _ := Read(dir)
	if st.Changed() != 0 {
		t.Fatalf("the stash left %d changes", st.Changed())
	}
	list, err := Stashes(dir)
	if err != nil || len(list) != 1 || !strings.Contains(list[0], "wip") {
		t.Fatalf("stashes = %v, %v", list, err)
	}
	if err := StashPop(dir); err != nil {
		t.Fatal(err)
	}
	st, _, _ = Read(dir)
	if st.Changed() == 0 {
		t.Fatal("pop did not bring the changes back")
	}
}

func TestSyncAgainstARemote(t *testing.T) {
	dir := fixture(t)
	remote, _ := filepath.EvalSymlinks(t.TempDir())
	run(t, remote, "init", "-q", "--bare", "-b", "main")
	run(t, dir, "stash", "-u", "-q")
	run(t, dir, "remote", "add", "origin", remote)
	if err := Push(dir); err == nil {
		t.Fatal("pushing without an upstream must say so, not silently create one")
	}
	if err := PushSetUpstream(dir); err != nil {
		t.Fatalf("PushSetUpstream: %v", err)
	}
	// someone else pushes a commit; fetch shows it behind, pull brings it
	other, _ := filepath.EvalSymlinks(t.TempDir())
	run(t, other, "clone", "-q", remote, "c")
	c := filepath.Join(other, "c")
	put(t, filepath.Join(c, "new.txt"), "n\n")
	run(t, c, "add", ".")
	run(t, c, "commit", "-q", "-m", "n")
	run(t, c, "push", "-q", "origin", "main")
	if err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	st, _, _ := Read(dir)
	if st.Behind != 1 {
		t.Fatalf("behind = %d, want 1", st.Behind)
	}
	if err := Pull(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatalf("pull did not bring the file: %v", err)
	}
	put(t, filepath.Join(dir, "mine.txt"), "m\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "m")
	if err := Push(dir); err != nil {
		t.Fatal(err)
	}
	st, _, _ = Read(dir)
	if st.Ahead != 0 {
		t.Fatalf("ahead = %d after push", st.Ahead)
	}
}

func TestConflictsAreListed(t *testing.T) {
	dir := fixture(t)
	run(t, dir, "stash", "-u", "-q")
	run(t, dir, "checkout", "-q", "-b", "side")
	put(t, filepath.Join(dir, "edited.txt"), "side\n")
	run(t, dir, "commit", "-q", "-am", "side")
	run(t, dir, "checkout", "-q", "main")
	put(t, filepath.Join(dir, "edited.txt"), "main\n")
	run(t, dir, "commit", "-q", "-am", "main")
	cmd := execIn(dir, "merge", "side") // fails with a conflict, on purpose
	_ = cmd
	st, _, _ := Read(dir)
	var found bool
	for _, c := range st.Unstaged {
		if c.Path == "edited.txt" && c.Code == Conflicted {
			found = true
		}
	}
	if !found {
		t.Fatalf("conflict not listed: %+v", st.Unstaged)
	}
}

// execIn runs git in dir and ignores its failure: used to leave a repository in a state that git reports as an error.
func execIn(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	return cmd.Run()
}

func TestStashApplyPopAndDropByIndex(t *testing.T) {
	dir := fixture(t)
	if err := StashPush(dir, "first"); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "edited.txt"), "three\n")
	if err := StashPush(dir, "second"); err != nil {
		t.Fatal(err)
	}
	list, _ := Stashes(dir)
	if len(list) != 2 || !strings.Contains(list[0], "second") || !strings.Contains(list[1], "first") {
		t.Fatalf("stashes = %v", list)
	}
	if err := StashApplyAt(dir, 1); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if list, _ = Stashes(dir); len(list) != 2 {
		t.Fatalf("apply must keep the stash, got %v", list)
	}
	if err := StashDropAt(dir, 0); err != nil {
		t.Fatal(err)
	}
	if list, _ = Stashes(dir); len(list) != 1 || !strings.Contains(list[0], "first") {
		t.Fatalf("after dropping the newest: %v", list)
	}
	run(t, dir, "checkout", "--", ".")
	run(t, dir, "clean", "-fdq")
	if err := StashPopAt(dir, 0); err != nil {
		t.Fatalf("pop: %v", err)
	}
	if list, _ = Stashes(dir); len(list) != 0 {
		t.Fatalf("pop must remove the stash, got %v", list)
	}
	if err := StashDropAt(dir, 5); err == nil {
		t.Fatal("dropping a stash that does not exist must fail")
	}
	if err := StashDropAt(dir, -1); err == nil {
		t.Fatal("a negative index must be refused")
	}
}

func TestChangesOfFindsBothSidesOfAFileByAbsolutePath(t *testing.T) {
	dir := fixture(t)
	put(t, filepath.Join(dir, "staged.txt"), "new\nmore\n") // staged as new, then edited again: both sides
	st, _, _ := Read(dir)
	staged, unstaged := st.ChangesOf(filepath.Join(dir, "staged.txt"))
	if len(staged) != 1 || len(unstaged) != 1 {
		t.Fatalf("staged=%v unstaged=%v, want one of each", staged, unstaged)
	}
	staged, unstaged = st.ChangesOf(filepath.Join(dir, "kept.txt"))
	if len(staged)+len(unstaged) != 0 {
		t.Fatalf("an unchanged file has no changes: %v %v", staged, unstaged)
	}
	staged, unstaged = st.ChangesOf(filepath.Join(filepath.Dir(dir), "elsewhere.txt"))
	if len(staged)+len(unstaged) != 0 {
		t.Fatal("a path outside the repository has no changes")
	}
}
