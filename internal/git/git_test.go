package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// run executes git in dir with a config that cannot read the user's own.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func put(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture builds a repo with one commit and then a modified, a staged, an untracked and an ignored file.
func fixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run(t, dir, "init", "-q", "-b", "main")
	put(t, filepath.Join(dir, ".gitignore"), "*.log\nbuild/\n")
	put(t, filepath.Join(dir, "kept.txt"), "one\n")
	put(t, filepath.Join(dir, "edited.txt"), "one\n")
	put(t, filepath.Join(dir, "sub", "deep.txt"), "one\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "init")
	put(t, filepath.Join(dir, "edited.txt"), "two\n")
	put(t, filepath.Join(dir, "sub", "deep.txt"), "two\n")
	put(t, filepath.Join(dir, "staged.txt"), "new\n")
	run(t, dir, "add", "staged.txt")
	put(t, filepath.Join(dir, "loose.txt"), "x\n")
	put(t, filepath.Join(dir, "fresh", "a.txt"), "x\n")
	put(t, filepath.Join(dir, "run.log"), "x\n")
	put(t, filepath.Join(dir, "build", "out.bin"), "x\n")
	return dir
}

func TestReadReportsBranchAndFileStates(t *testing.T) {
	dir := fixture(t)
	s, ok, err := Read(dir)
	if err != nil || !ok {
		t.Fatalf("Read = ok %v, err %v", ok, err)
	}
	if s.Branch != "main" || s.Root != dir {
		t.Fatalf("branch %q root %q, want main and %q", s.Branch, s.Root, dir)
	}
	want := map[string]Code{
		"kept.txt":   None,
		"edited.txt": Modified,
		"staged.txt": Added,
		"loose.txt":  Untracked,
		"run.log":    Ignored,
		"build":      Ignored,
	}
	for name, code := range want {
		if got := s.Of(filepath.Join(dir, name), name == "build"); got != code {
			t.Errorf("%s = %q, want %q", name, got, code)
		}
	}
}

func TestADirectoryTakesTheStrongestStateInside(t *testing.T) {
	dir := fixture(t)
	s, _, _ := Read(dir)
	if got := s.Of(filepath.Join(dir, "sub"), true); got != Modified {
		t.Errorf("sub = %q, want Modified", got)
	}
	if got := s.Of(filepath.Join(dir, "fresh"), true); got != Untracked {
		t.Errorf("fresh = %q, want Untracked", got)
	}
}

func TestAConflictWinsOverEverythingElse(t *testing.T) {
	dir := fixture(t)
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", "second")
	run(t, dir, "checkout", "-q", "-b", "other")
	put(t, filepath.Join(dir, "edited.txt"), "other\n")
	run(t, dir, "commit", "-q", "-am", "other")
	run(t, dir, "checkout", "-q", "main")
	put(t, filepath.Join(dir, "edited.txt"), "main\n")
	run(t, dir, "commit", "-q", "-am", "main")
	// The identity comes from flags: a CI runner has none, and without one the merge stops before it conflicts.
	cmd := exec.Command("git", "-c", "user.name=t", "-c", "user.email=t@example.com", "merge", "other")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	_ = cmd.Run() // fails on purpose: it leaves the conflict
	s, _, _ := Read(dir)
	if got := s.Of(filepath.Join(dir, "edited.txt"), false); got != Conflicted {
		t.Fatalf("edited.txt = %q, want Conflicted", got)
	}
}

func TestReadFromASubdirectoryUsesTheRepositoryRoot(t *testing.T) {
	dir := fixture(t)
	s, ok, err := Read(filepath.Join(dir, "sub"))
	if err != nil || !ok || s.Root != dir {
		t.Fatalf("ok %v err %v root %q, want root %q", ok, err, s.Root, dir)
	}
	if got := s.Of(filepath.Join(dir, "sub", "deep.txt"), false); got != Modified {
		t.Fatalf("deep.txt = %q, want Modified", got)
	}
}

func TestOutsideARepositoryIsNotAnError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	_, ok, err := Read(t.TempDir())
	if err != nil || ok {
		t.Fatalf("ok %v err %v, want a quiet false", ok, err)
	}
}

func TestADetachedHeadIsNamed(t *testing.T) {
	dir := fixture(t)
	run(t, dir, "checkout", "-q", "--detach")
	s, _, _ := Read(dir)
	if s.Branch != "HEAD" {
		t.Fatalf("branch %q, want HEAD for a detached head", s.Branch)
	}
}

func TestPathsThroughASymlinkedFolderStillMatch(t *testing.T) {
	dir := fixture(t)
	link := filepath.Join(t.TempDir(), "via")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	s, ok, err := Read(link)
	if err != nil || !ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	if got := s.Of(filepath.Join(link, "edited.txt"), false); got != Modified {
		t.Fatalf("edited.txt through the link = %q, want Modified", got)
	}
}
