package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Exploratory QA of order 07 point O, kept as tests: the places a person finds by walking around (odd names,
// broken links, folders that cannot be read, big and empty folders, a repository in each state), in the normal
// window and in the sidebar mode, at three sizes.

// The window needs 60x16 at least; the sidebar mode is made for narrow terminals.
var (
	windowSizes  = [][2]int{{60, 16}, {80, 24}, {160, 50}}
	sidebarSizes = [][2]int{{24, 12}, {36, 16}, {80, 24}}
)

// modes runs f in the normal window and in the sidebar mode, on a copy of the fixture folder with extra set up.
func modes(t *testing.T, setup func(e *env), f func(t *testing.T, e *env)) {
	t.Helper()
	for _, project := range []bool{false, true} {
		sizes := windowSizes
		if project {
			sizes = sidebarSizes
		}
		for _, sz := range sizes {
			name := fmt.Sprintf("%dx%d/window", sz[0], sz[1])
			if project {
				name = fmt.Sprintf("%dx%d/sidebar", sz[0], sz[1])
			}
			t.Run(name, func(t *testing.T) {
				e := newEnv(t)
				if project {
					e = projectEnvWithoutGit(t, sz[0])
				}
				if setup != nil {
					setup(e)
				}
				e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
				e.key("r")
				f(t, e)
			})
		}
	}
}

// walk presses the navigation keys and opens and closes what can be opened, checking the screen after each.
func (e *env) walk() {
	e.t.Helper()
	for _, k := range []string{"down", "down", "down", "up", "end", "home", "pgdown", "pgup", "space", "esc", "i", "esc", "+", "-", "/", "esc", "tab", "tab", "?", "esc", ","} {
		e.key(k)
		e.mustFit("key " + k)
	}
	e.key("esc")
}

func TestOddNamesDoNotBreakTheScreen(t *testing.T) {
	names := []string{
		"a very long name that goes on and on and on and on and on and on and on and on and on and on.txt",
		"文件夹文件夹文件夹文件夹.txt", "emoji 🎉🎉🎉.md", "tab\there.txt", "-dash-first.txt", "  leading spaces.txt",
		"ansi \x1b[31mred\x1b[0m.txt", "bell\a.txt", "combining é.txt", "right-to-left ‮txt.exe", ".hidden-dot", "100%.txt", "{file}.txt", "$(touch x).txt",
	}
	modes(t, func(e *env) {
		for _, n := range names {
			if err := os.WriteFile(filepath.Join(e.root, n), []byte("x\n"), 0o644); err != nil {
				t.Skipf("this file system refuses %q: %v", n, err)
			}
		}
	}, func(t *testing.T, e *env) {
		e.mustFit("start")
		e.walk()
		for i := 0; i < 20; i++ { // select every row and ask for its details and menu
			e.key("down")
			e.key("i")
			e.mustFit("info")
			e.key("esc")
			e.key("O")
			e.mustFit("open with")
			e.key("esc")
		}
	})
}

func TestBrokenSymlinksAreListedAndSafe(t *testing.T) {
	modes(t, func(e *env) {
		if err := os.Symlink("/nonexistent/target", filepath.Join(e.root, "dangling")); err != nil {
			t.Skip(err)
		}
		_ = os.Symlink(".", filepath.Join(e.root, "loop")) // a link to its own folder
		_ = os.Symlink("loop2", filepath.Join(e.root, "loop2"))
	}, func(t *testing.T, e *env) {
		e.mustHave("dangling")
		e.walk()
		for _, n := range []string{"dangling", "loop", "loop2"} {
			x, y := e.find(n)
			e.click(x, y)
			start := e.m.dir()
			for _, k := range []string{"space", "esc", "i", "esc", "enter", "esc"} {
				e.key(k)
				e.mustFit(n + " key " + k)
			}
			if e.m.dir() != start && !e.m.projectMode() {
				e.key("alt+left")
			}
		}
	})
}

func TestAFolderWithoutPermissionSaysSoAndStays(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	modes(t, func(e *env) {
		p := filepath.Join(e.root, "locked")
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(p, 0o755) })
	}, func(t *testing.T, e *env) {
		e.mustHave("locked")
		x, y := e.find("locked")
		e.click(x, y)
		for _, k := range []string{"enter", "space", "esc", "+", "-", "i", "esc"} {
			e.key(k)
			e.mustFit("locked key " + k)
		}
		e.mustHave("locked") // still in the project, still showing the folder
	})
}

func TestABigFolderScrollsAndStaysResponsive(t *testing.T) {
	modes(t, func(e *env) {
		dir := filepath.Join(e.root, "big")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3000; i++ {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%04d.txt", i)), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}, func(t *testing.T, e *env) {
		x, y := e.find("big")
		e.click(x, y)
		e.key("enter")
		for _, k := range []string{"end", "pgup", "home", "pgdown", "pgdown", "G", "g"} {
			e.key(k)
			e.mustFit("big key " + k)
		}
		e.key("/")
		e.typeText("file-29")
		e.mustFit("search")
		e.key("esc")
	})
}

func TestAnEmptyFolderShowsAndDoesNotFallOver(t *testing.T) {
	modes(t, func(e *env) {
		for _, n := range []string{"notes.txt", "data.xlsx", ".secret"} {
			_ = os.Remove(filepath.Join(e.root, n))
		}
		_ = os.RemoveAll(filepath.Join(e.root, "alpha"))
		_ = os.RemoveAll(filepath.Join(e.root, "beta"))
	}, func(t *testing.T, e *env) {
		e.walk()
		for _, k := range []string{"enter", "space", "i", "d", "x", "c", "p", "R", "O", "v", "ctrl+a", "t", "u"} {
			e.key(k)
			e.mustFit("empty key " + k)
			e.key("esc")
		}
	})
}

// gitStates builds a repository in each state a person meets.
func cleanRepo(e *env) {
	e.git("init", "-q", "-b", "main")
	e.git("add", "-A")
	e.git("commit", "-q", "-m", "init")
}

var gitStates = map[string]func(e *env){
	"clean":      cleanRepo,
	"no commits": func(e *env) { e.git("init", "-q", "-b", "main") },
	"modified": func(e *env) {
		cleanRepo(e)
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("changed\n"), 0o644)
	},
	"staged and untracked": func(e *env) {
		cleanRepo(e)
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("changed\n"), 0o644)
		e.git("add", "notes.txt")
		_ = os.WriteFile(filepath.Join(e.root, "loose.txt"), []byte("x\n"), 0o644)
	},
	"detached": func(e *env) {
		cleanRepo(e)
		e.git("checkout", "-q", "--detach")
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("changed\n"), 0o644)
	},
	"conflict": func(e *env) {
		cleanRepo(e)
		e.git("checkout", "-q", "-b", "other")
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("other\n"), 0o644)
		e.git("commit", "-q", "-am", "other")
		e.git("checkout", "-q", "main")
		_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("main\n"), 0o644)
		e.git("commit", "-q", "-am", "main")
		_ = e.gitMayFail("merge", "other")
	},
	"ahead and behind": func(e *env) {
		cleanRepo(e)
		remote := filepath.Join(filepath.Dir(e.root), "remote.git")
		e.git("init", "-q", "--bare", "-b", "main", remote)
		e.git("remote", "add", "origin", remote)
		e.git("push", "-q", "-u", "origin", "main")
		_ = os.WriteFile(filepath.Join(e.root, "ahead.txt"), []byte("x\n"), 0o644)
		e.git("add", "ahead.txt")
		e.git("commit", "-q", "-m", "ahead")
		other := filepath.Join(filepath.Dir(e.root), "clone")
		e.git("clone", "-q", remote, other)
		_ = os.WriteFile(filepath.Join(other, "behind.txt"), []byte("x\n"), 0o644)
		e.gitIn(other, "add", "behind.txt")
		e.gitIn(other, "commit", "-q", "-m", "behind")
		e.gitIn(other, "push", "-q", "origin", "main")
		e.git("fetch", "-q")
	},
}

func TestTheGitTabInEachRepositoryState(t *testing.T) {
	for name, build := range gitStates {
		t.Run(name, func(t *testing.T) {
			for _, sz := range sidebarSizes {
				t.Run(fmt.Sprintf("%dx%d", sz[0], sz[1]), func(t *testing.T) {
					e := projectEnvWithoutGit(t, sz[0])
					build(e)
					e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
					e.key("r")
					_, spans := e.m.projectTabsLine()
					e.click(spans[tabGit].x+1, 0) // at 24 columns only the glyph of the tab is left
					e.mustFit("git tab")
					if strings.Contains(e.text(), "Not a git repository") {
						t.Fatalf("a repository in state %q is not recognised:\n%s", name, e.text())
					}
					for _, k := range []string{"down", "down", "s", "u", "enter", "esc", "B", "esc", "f", "p", "P", "z", "d", "n", "o", "up", "enter", "esc"} {
						e.key(k)
						e.mustFit(name + " key " + k)
					}
				})
			}
		})
	}
}

func (e *env) gitIn(dir string, args ...string) {
	e.t.Helper()
	if out, err := e.gitRaw(dir, args...); err != nil {
		e.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// gitMayFail runs git where a failure is the point, as a merge that conflicts.
func (e *env) gitMayFail(args ...string) error {
	_, err := e.gitRaw(e.root, args...)
	return err
}

func (e *env) gitRaw(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	return cmd.CombinedOutput()
}
