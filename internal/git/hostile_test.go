package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// hostile builds a repository whose own .git/config runs a command from every key git runs while it only reads:
// core.fsmonitor on status, a clean filter on a stat-dirty file, and a textconv driver on diff. Each command leaves a
// marker file in the returned folder, outside the repository.
func hostile(t *testing.T) (repo, marks string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo = fixture(t)
	marks = t.TempDir()
	put(t, filepath.Join(repo, ".gitattributes"), "*.txt filter=evil diff=evil\n")
	for key, cmd := range map[string]string{
		"core.fsmonitor":     "touch " + filepath.Join(marks, "fsmonitor") + "; false",
		"filter.evil.clean":  "touch " + filepath.Join(marks, "clean") + "; cat",
		"filter.evil.smudge": "touch " + filepath.Join(marks, "smudge") + "; cat",
		"diff.evil.textconv": "touch " + filepath.Join(marks, "textconv") + "; cat",
	} {
		run(t, repo, "config", key, cmd)
	}
	// An older mtime than the index makes the file stat-dirty, so git has to run the clean filter to compare it.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(repo, "kept.txt"), old, old); err != nil {
		t.Fatal(err)
	}
	return repo, marks
}

func ran(t *testing.T, marks string) []string {
	t.Helper()
	entries, err := os.ReadDir(marks)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestReadingAHostileRepositoryRunsNothingFromItsConfig(t *testing.T) {
	repo, marks := hostile(t)
	s, ok, err := Read(repo)
	if err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if got := s.Of(filepath.Join(repo, "edited.txt"), false); got != Modified {
		t.Errorf("edited.txt = %v, want Modified: the status must still work", got)
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("reading the repository ran commands from its config: %v", got)
	}
}

func TestADiffOfAHostileRepositoryRunsNothingFromItsConfig(t *testing.T) {
	repo, marks := hostile(t)
	d, err := Diff(repo, Change{Path: "edited.txt", Code: Modified}, false)
	if err != nil || !strings.Contains(d, "+two") {
		t.Fatalf("diff = %q, %v", d, err)
	}
	if _, err := Diff(repo, Change{Path: "staged.txt", Code: Added}, true); err != nil {
		t.Fatal(err)
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("the diff ran commands from the repository config: %v", got)
	}
}

// readOnly switched settings off with -c key=, which git splits at the first =: for a filter named "x=y" it set
// filter.x to "y.clean=" and left filter.x=y.clean on, so showing the folder ran the repository's clean filter.
func TestAFilterWithAnEqualsSignInItsNameRunsNothing(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := fixture(t)
	marks := t.TempDir()
	put(t, filepath.Join(repo, ".gitattributes"), "*.txt filter=x=y\n")
	run(t, repo, "config", "filter.x=y.clean", "touch "+filepath.Join(marks, "clean")+"; cat")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(repo, "kept.txt"), old, old); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := Read(repo); err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if _, err := Diff(repo, Change{Path: "edited.txt", Code: Modified}, false); err != nil {
		t.Fatal(err)
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("a filter named x=y ran: %v", got)
	}
}

// GIT_CONFIG_KEY_n only exists since git 2.31; an older git ignores it, so row 73 left every hostile setting on there.
// Settings whose name has no = go as -c, which every git reads, and a repository with a filter named with = is not
// read at all by a git that cannot switch it off.
func TestAnOldGitSwitchesHostileSettingsOffOrRefuses(t *testing.T) {
	old := configEnvWorks
	configEnvWorks = func() bool { return false }
	t.Cleanup(func() { configEnvWorks = old })
	repo, marks := hostile(t)
	t.Setenv("GIT_CONFIG_COUNT", "")
	if _, ok, err := Read(repo); err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if _, err := Diff(repo, Change{Path: "edited.txt", Code: Modified}, false); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(repo, ".gitattributes"), "*.txt filter=x=y\n")
	run(t, repo, "config", "filter.x=y.clean", "touch "+filepath.Join(marks, "eq")+"; cat")
	if _, _, err := Read(repo); err == nil {
		t.Fatal("a git that cannot switch a filter named x=y off must not read the repository")
	}
	if _, err := Diff(repo, Change{Path: "edited.txt", Code: Modified}, false); err == nil {
		t.Fatal("nor diff it")
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("an old git ran commands from the repository config: %v", got)
	}
}

// readOnly took any failure of git config for "no such setting" and switched nothing off: a git older than 2.26,
// which does not know --show-scope, exits 129, and status then ran the repository's fsmonitor. Only exit 1 means
// there is nothing to switch off; any other failure stops the read.
func TestReadOnlyFailsClosed(t *testing.T) {
	repo, _ := hostile(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := readOnly(ctx, repo); err == nil {
		t.Fatal("a git config that did not run must not mean there is nothing to switch off")
	}
	if args, env, err := readOnly(context.Background(), fixture(t)); err != nil || args != nil || env != nil {
		t.Fatalf("a repository without such settings: %v, %v, %v", args, env, err)
	}
}

