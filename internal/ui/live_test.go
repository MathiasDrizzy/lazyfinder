package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
)

// fakeWatcher records what the model asks for and never emits by itself.
type fakeWatcher struct {
	mu      sync.Mutex
	watched map[string]bool
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{watched: map[string]bool{}}
}

func (f *fakeWatcher) Watch(d string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.watched[d] = true
	return nil
}
func (f *fakeWatcher) Unwatch(d string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.watched, d)
}
func (f *fakeWatcher) Events() <-chan watch.Event { return nil }
func (f *fakeWatcher) Close() error               { return nil }
func (f *fakeWatcher) has(d string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watched[d]
}
func (f *fakeWatcher) only(d string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.watched) == 1 && f.watched[d]
}

// liveEnv is newEnv with a fake watcher. The model is built again so the watcher exists from the start.
func liveEnv(t *testing.T, cfgJSON string) (*env, *fakeWatcher) {
	t.Helper()
	e := newEnv(t)
	cfg := e.m.deps.Cfg
	if cfgJSON != "" {
		var err error
		if cfg, err = config.Parse([]byte(cfgJSON), e.home); err != nil {
			t.Fatal(err)
		}
	}
	fw := newFakeWatcher()
	d := e.m.deps
	d.Cfg, d.Watcher = cfg, fw
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return e, fw
}

func (e *env) touch(name, body string) {
	e.t.Helper()
	if err := os.WriteFile(filepath.Join(e.root, name), []byte(body), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func TestTheCurrentFolderIsWatchedAndFollowsNavigation(t *testing.T) {
	e, fw := liveEnv(t, "")
	if !fw.only(e.root) {
		t.Fatalf("watched = %v, want only %s", fw.watched, e.root)
	}
	e.clickText("alpha")
	e.key("enter")
	if !fw.only(filepath.Join(e.root, "alpha")) {
		t.Fatalf("after entering alpha watched = %v", fw.watched)
	}
	e.key("backspace")
	if !fw.only(e.root) {
		t.Fatalf("after going back watched = %v", fw.watched)
	}
}

func TestAChangeEventShowsTheNewFileWithoutRefreshing(t *testing.T) {
	e, _ := liveEnv(t, "")
	e.mustNotHave("created-by-agent.go")
	e.touch("created-by-agent.go", "package x\n")
	e.send(dirChangedMsg{dir: e.root})
	e.mustHave("created-by-agent.go")
}

func TestAChangeEventKeepsTheSelectionByName(t *testing.T) {
	e, _ := liveEnv(t, "")
	e.clickText("notes.txt")
	e.touch("aaa-first.txt", "x")
	e.send(dirChangedMsg{dir: e.root})
	if got, _ := e.m.selected(); got.Name != "notes.txt" {
		t.Fatalf("selected %q after a live update, want notes.txt", got.Name)
	}
}

func TestAChangeEventForAnotherFolderIsIgnored(t *testing.T) {
	e, _ := liveEnv(t, "")
	e.touch("late.txt", "x")
	e.send(dirChangedMsg{dir: filepath.Join(e.root, "alpha")})
	e.mustNotHave("late.txt")
}

func TestNewFilesAreMarkedUntilYouLeaveTheFolder(t *testing.T) {
	e, _ := liveEnv(t, "")
	e.touch("fresh.txt", "x")
	e.send(dirChangedMsg{dir: e.root})
	if !e.m.isFresh(filepath.Join(e.root, "fresh.txt")) || e.m.isFresh(filepath.Join(e.root, "notes.txt")) {
		t.Fatalf("fresh = %v, want only fresh.txt", e.m.fresh)
	}
	if !strings.Contains(e.row("fresh.txt"), "+") {
		t.Fatalf("a new file carries a + in its row (no color needed):\n%s", e.row("fresh.txt"))
	}
	e.clickText("alpha")
	e.key("enter")
	e.key("backspace")
	if e.m.isFresh(filepath.Join(e.root, "fresh.txt")) {
		t.Fatal("the new-file mark must clear after leaving the folder")
	}
}

func TestAVanishedFolderMovesTheViewToItsParent(t *testing.T) {
	e, _ := liveEnv(t, "")
	e.clickText("beta")
	e.key("enter")
	if err := os.Remove(filepath.Join(e.root, "beta")); err != nil {
		t.Fatal(err)
	}
	e.send(dirChangedMsg{dir: filepath.Join(e.root, "beta")})
	if e.m.dir() != e.root {
		t.Fatalf("dir = %s, want %s after the folder vanished", e.m.dir(), e.root)
	}
}

// repo turns the fixture folder into a git repository with an edited, a staged and an untracked file.
func repo(t *testing.T, e *env) {
	t.Helper()
	g := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = e.root
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	e.touch(".gitignore", "*.log\n")
	g("init", "-q", "-b", "trunk")
	g("add", ".")
	g("commit", "-q", "-m", "init")
	e.touch("notes.txt", "changed\n")
	e.touch("loose.txt", "x\n")
	e.touch("debug.log", "x\n")
}

func TestGitStateAndBranchAreShown(t *testing.T) {
	e, _ := liveEnv(t, "")
	repo(t, e)
	e.send(dirChangedMsg{dir: e.root})
	e.mustHave("trunk")
	if r := e.row("notes.txt"); !strings.Contains(r, "M") {
		t.Fatalf("notes.txt must show M:\n%s", r)
	}
	if r := e.row("loose.txt"); !strings.Contains(r, "?") {
		t.Fatalf("loose.txt must show ?:\n%s", r)
	}
	if r := e.row("data.xlsx"); strings.Contains(r, " M ") || strings.Contains(r, " ? ") {
		t.Fatalf("a clean file shows no state:\n%s", r)
	}
}

func TestOutsideARepositoryThereIsNoGitGutter(t *testing.T) {
	e, _ := liveEnv(t, "")
	if e.m.git.ok {
		t.Fatal("the fixture folder is not a repository")
	}
}

func TestHideIgnoredWithTheKey(t *testing.T) {
	e, _ := liveEnv(t, "")
	repo(t, e)
	e.send(dirChangedMsg{dir: e.root})
	e.mustHave("debug.log")
	e.key("I")
	e.mustNotHave("debug.log")
	e.key("I")
	e.mustHave("debug.log")
}

func TestGitCanBeTurnedOffInTheConfig(t *testing.T) {
	cfg := strings.Replace(cfgJSON, `"version": 1,`, `"version": 1, "git": {"enabled": false},`, 1)
	e, _ := liveEnv(t, cfg)
	repo(t, e)
	e.send(dirChangedMsg{dir: e.root})
	e.mustNotHave("trunk")
}
