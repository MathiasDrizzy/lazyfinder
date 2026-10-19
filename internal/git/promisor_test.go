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
	// The read fails closed, and says what git said, not only its exit status.
	if _, _, err := Read(repo); err == nil || !strings.Contains(err.Error(), "exit status 128: ") {
		t.Errorf("Read = %v, want the failure with git's message", err)
	}
	_, _ = Diff(repo, Change{Path: "new.txt", Code: Renamed, Orig: "old.txt"}, true)
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("reading a partial clone fetched from its promisor remote: %v", got)
	}
}

// Without git on the PATH a folder is simply not a repository: Read reported the missing program as an error, and
// the status line showed it on every folder.
func TestWithoutGitAFolderIsNoRepository(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", filepath.Join(dir, "nothing"))
	if _, ok, err := Read(dir); ok || err != nil {
		t.Fatalf("Read = %v, %v", ok, err)
	}
}

// git could not start in a folder you may list but not enter, and Read reported it as an error, which the status line
// showed on every refresh. git cannot read such a folder, so it is no repository either.
func TestAFolderGitCannotEnterIsNoRepository(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters any folder")
	}
	dir := filepath.Join(t.TempDir(), "d")
	if err := os.Mkdir(dir, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, ok, err := Read(dir); ok || err != nil {
		t.Fatalf("Read = %v, %v", ok, err)
	}
}
