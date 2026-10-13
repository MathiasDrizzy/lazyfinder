package ui

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/charmbracelet/x/ansi"
)

var update = flag.Bool("update", false, "rewrite the golden files")

const goldenCfg = `{
  "version": 1,
  "columns": ["name", "modified", "size", "kind"],
  "sidebar": {
    "favorites": [{"name": "Home", "path": "~"}, {"name": "Projects", "path": "~/proj"}],
    "locations": [{"name": "Disk", "path": "/"}]
  },
  "openers": {"edit": {"argv": ["myeditor", "{file}"], "mode": "gui"}},
  "open_with": {"txt": "edit"}
}`

// goldenEnv builds a tree with fixed names, sizes and modification times so screens are reproducible. The created
// column is left out of the config because the file system sets it at copy time.
func goldenEnv(t *testing.T, w, h int) *env {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, "proj")
	mtime := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	tree := map[string]string{
		"alpha/inner.txt":  "inner text\n",
		"alpha/deep/x.txt": "deep\n",
		"beta/.keep":       "",
		"notes.txt":        "first line\nsecond line\nfunc main() {}\n",
		"data.xlsx":        "PK\x03\x04",
		"main.go":          "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n",
	}
	for n, body := range tree {
		p := filepath.Join(root, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_ = filepath.WalkDir(root, func(p string, _ os.DirEntry, _ error) error { return os.Chtimes(p, mtime, mtime) })
	cfg, err := config.Parse([]byte(goldenCfg), home)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, root: root, home: home, run: &fakeRunner{}, now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	e.m = New(Deps{Home: home, Start: root, Cfg: cfg, Runner: e.run, Now: func() time.Time { return e.now }})
	e.send(tea.WindowSizeMsg{Width: w, Height: h})
	return e
}

// golden compares the plain-text screen with testdata/golden/<name>.txt. The temp directory in the sidebar-less
// breadcrumb row is not reproducible, so that row is masked; its behaviour has its own tests.
func (e *env) golden(name string) {
	e.t.Helper()
	lines := e.screen()
	lines[len(lines)-2] = "<breadcrumb>"
	got := strings.Join(trimRight(lines), "\n") + "\n"
	got = strings.ReplaceAll(got, e.home, "<HOME>")
	path := filepath.Join("testdata", "golden", name+".txt")
	if *update {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			e.t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		e.t.Fatalf("missing golden %s (run with -update): %v", path, err)
	}
	if string(want) != got {
		e.t.Fatalf("screen differs from %s (run with -update and review the diff)\n--- want\n%s\n--- got\n%s", path, want, got)
	}
}

func trimRight(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(ansi.Strip(l), " ")
	}
	return out
}

func TestGoldenList120x40(t *testing.T) { goldenEnv(t, 120, 40).golden("list-120x40") }
func TestGoldenList80x24(t *testing.T)  { goldenEnv(t, 80, 24).golden("list-80x24") }
func TestGoldenList60x16(t *testing.T)  { goldenEnv(t, 60, 16).golden("list-60x16") }

func TestGoldenTooSmall(t *testing.T) { goldenEnv(t, 50, 10).golden("too-small") }

func TestGoldenPreviewText(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.clickText("main.go")
	e.key("space")
	e.golden("preview-text")
}

func TestGoldenPreviewFolder(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.key("space")
	e.golden("preview-folder")
}

func TestGoldenOpenWithMenu(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.clickText("notes.txt")
	e.key("O")
	e.golden("open-with")
}

func TestGoldenSortedBySizeDescending(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.clickText("Size")
	e.clickText("Size")
	e.golden("sort-size-desc")
}

func TestGoldenSidebarFocus(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.key("tab")
	e.key("down")
	e.golden("sidebar-focus")
}

func TestGoldenEmptyFolder(t *testing.T) {
	e := goldenEnv(t, 120, 40)
	e.clickText("beta")
	e.key("enter")
	e.key(".")
	e.key(".")
	e.golden("empty-folder")
}
