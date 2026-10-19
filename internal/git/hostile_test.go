package git

import (
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
