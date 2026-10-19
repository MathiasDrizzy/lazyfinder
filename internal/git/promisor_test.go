package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A repository that says it is a partial clone fetches a missing object from its promisor remote while git only
// reads, and that remote's uploadpack is a command from the repository's config: a staged rename whose old blob is
// missing ran it as soon as lazyfinder showed the folder. Reads never fetch.
func TestReadingAPartialCloneFetchesNothing(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := fixture(t)
	marks := t.TempDir()
	body := strings.Repeat("a line that stays the same\n", 40)
	put(t, filepath.Join(repo, "old.txt"), body)
	run(t, repo, "add", "old.txt")
	run(t, repo, "commit", "-q", "-m", "old")
	blob, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD:old.txt").Output()
	if err != nil {
		t.Fatal(err)
	}
	run(t, repo, "mv", "old.txt", "new.txt")
	put(t, filepath.Join(repo, "new.txt"), body+"one more\n")
	run(t, repo, "add", "new.txt")
	id := strings.TrimSpace(string(blob))
	if err := os.Remove(filepath.Join(repo, ".git", "objects", id[:2], id[2:])); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"core.repositoryformatversion": "1",
		"extensions.partialClone":      "origin",
		"remote.origin.url":            marks,
		"remote.origin.promisor":       "true",
		"remote.origin.uploadpack":     "touch " + filepath.Join(marks, "uploadpack") + "; false",
	} {
		run(t, repo, "config", key, value)
	}
	_, _, _ = Read(repo)
	_, _ = Diff(repo, Change{Path: "new.txt", Code: Renamed, Orig: "old.txt"}, true)
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("reading a partial clone fetched from its promisor remote: %v", got)
	}
}
