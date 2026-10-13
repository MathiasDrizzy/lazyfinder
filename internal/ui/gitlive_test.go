package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
)

// tapWatcher is a real watcher whose events the test pumps by hand, so no command of the model blocks on the channel.
type tapWatcher struct{ *watch.FS }

func (tapWatcher) Events() <-chan watch.Event { return nil }

// realEnv is liveEnv with the real file watcher, as the app runs.
func realEnv(t *testing.T) (*env, tapWatcher) {
	t.Helper()
	e := newEnv(t)
	fs, err := watch.NewFS(watch.DefaultOptions())
	if err != nil {
		t.Skipf("no file watcher here: %v", err)
	}
	t.Cleanup(func() { _ = fs.Close() })
	tw := tapWatcher{fs}
	d := e.m.deps
	d.Watcher = tw
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return e, tw
}

// until delivers the watcher's events to the model until cond holds or the time is up, and returns how long it took.
func (e *env) until(w tapWatcher, within time.Duration, cond func() bool) (time.Duration, bool) {
	e.t.Helper()
	start := time.Now()
	for time.Since(start) < within {
		if cond() {
			return time.Since(start), true
		}
		select {
		case ev := <-w.FS.Events():
			e.send(dirChangedMsg{dir: ev.Dir})
		case <-time.After(5 * time.Millisecond):
		}
	}
	return time.Since(start), cond()
}

func (e *env) git(args ...string) {
	e.t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = e.root
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (e *env) rowHas(name, letter string) func() bool {
	return func() bool {
		for _, l := range e.screen() {
			if strings.Contains(l, name) {
				return strings.Contains(l, " "+letter+" ")
			}
		}
		return false
	}
}

// Order 2026-10-13-06 point 6: the letters of VS Code's explorer, in color.
func TestGitLettersAreVSCodesAndColored(t *testing.T) {
	e, _ := liveEnv(t, "")
	repo(t, e)
	e.touch("fresh.txt", "x\n")
	e.git("add", "fresh.txt")
	e.send(dirChangedMsg{dir: e.root})
	for name, letter := range map[string]string{"notes.txt": "M", "loose.txt": "U", "fresh.txt": "A", "debug.log": "I"} {
		if !e.rowHas(name, letter)() {
			t.Errorf("%s must show %s:\n%s", name, letter, e.row(name))
		}
	}
	for name, color := range map[string]string{"notes.txt": "38;2;249;226;175", "loose.txt": "38;2;166;227;161", "fresh.txt": "38;2;166;227;161"} {
		if !strings.Contains(rawLine(e, name), color) {
			t.Errorf("the letter of %s must be drawn in %s: %q", name, color, rawLine(e, name))
		}
	}
}

func TestStagingCommittingAndSwitchingBranchRefreshByThemselves(t *testing.T) {
	e, w := realEnv(t)
	repo(t, e)
	e.send(dirChangedMsg{dir: e.root})
	if !e.rowHas("loose.txt", "U")() {
		t.Fatalf("setup: loose.txt is untracked:\n%s", e.text())
	}
	e.git("add", "loose.txt")
	if d, ok := e.until(w, 2*time.Second, e.rowHas("loose.txt", "A")); !ok {
		t.Fatalf("git add did not refresh the letter:\n%s", e.text())
	} else {
		t.Logf("git add -> A in %v", d)
	}
	e.git("commit", "-q", "-am", "second")
	if d, ok := e.until(w, 2*time.Second, func() bool { return !e.has(" M ") && !e.has(" A ") }); !ok {
		t.Fatalf("git commit did not clear the letters:\n%s", e.text())
	} else {
		t.Logf("git commit -> clean in %v", d)
	}
	e.git("checkout", "-q", "-b", "feature")
	if d, ok := e.until(w, 2*time.Second, func() bool { return e.has("feature") }); !ok {
		t.Fatalf("git checkout did not refresh the branch:\n%s", e.text())
	} else {
		t.Logf("git checkout -> branch in %v", d)
	}
}

// The latency of "another program made a file" -> "it is on the screen", with the real watcher and a big folder.
const liveLatencyLimit = 300 * time.Millisecond

func TestANewFileAppearsInUnder300msEvenInAFolderOfThousands(t *testing.T) {
	e, w := realEnv(t)
	for i := 0; i < 5000; i++ {
		if err := os.WriteFile(filepath.Join(e.root, fmt.Sprintf("bulk-%04d.dat", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e.send(dirChangedMsg{dir: e.root})
	e.until(w, 2*time.Second, func() bool { return len(e.m.entries()) > 5000 })
	for _, name := range []string{"made-by-agent.go", "other-agent.md"} {
		start := time.Now()
		e.touch(name, "x\n")
		d, ok := e.until(w, 2*time.Second, func() bool {
			for _, en := range e.m.entries() {
				if en.Name == name {
					return true
				}
			}
			return false
		})
		_ = start
		if !ok || d > liveLatencyLimit {
			t.Fatalf("%s took %v (limit %v, found: %v)", name, d, liveLatencyLimit, ok)
		}
		t.Logf("%s on the list after %v with %d entries", name, d, len(e.m.entries()))
	}
}

func TestWithoutGitInstalledTheListStillWorks(t *testing.T) {
	e, _ := liveEnv(t, "")
	t.Setenv("PATH", t.TempDir())
	e.send(dirChangedMsg{dir: e.root})
	if e.m.git.ok {
		t.Fatal("no git, no marks")
	}
	e.mustHave("notes.txt")
}

// The project sidebar mode (the VS Code explorer) is entered and left from inside the app with ctrl+b.
func TestCtrlBTurnsTheProjectSidebarOnAndOff(t *testing.T) {
	e, _ := liveEnv(t, "")
	if e.m.deps.Follow || e.m.sidebarW() == 0 {
		t.Fatal("the normal window starts with the places sidebar")
	}
	e.key("ctrl+b")
	if !e.m.deps.Follow || e.m.sidebarW() != 0 {
		t.Fatalf("ctrl+b must enter the project mode (no places, a tree):\n%s", e.text())
	}
	if !e.m.expanded[filepath.Join(e.root, "alpha")] {
		t.Fatal("the first level of folders opens when the mode starts")
	}
	if got := e.m.columns(); len(got) != 1 {
		t.Fatalf("the tree shows only names, got %v", got)
	}
	e.key("ctrl+b")
	if e.m.deps.Follow || e.m.sidebarW() == 0 || len(e.m.columns()) < 2 || len(e.m.expanded) != 0 {
		t.Fatalf("ctrl+b again must bring the window back:\n%s", e.text())
	}
}

func TestProjectModeDoesNotOpenIgnoredFoldersByItself(t *testing.T) {
	e, _ := liveEnv(t, "")
	repo(t, e)
	if err := os.MkdirAll(filepath.Join(e.root, "node_modules", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	e.touch(".gitignore", "*.log\nnode_modules\n")
	e.send(dirChangedMsg{dir: e.root})
	e.key("ctrl+b")
	if e.m.expanded[filepath.Join(e.root, "node_modules")] {
		t.Fatal("node_modules is ignored: opening it would read thousands of files")
	}
	if !e.m.expanded[filepath.Join(e.root, "alpha")] {
		t.Fatal("a normal folder still opens")
	}
}
