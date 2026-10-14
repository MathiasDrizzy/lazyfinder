package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// The sweep is the exploratory QA of order 07 point O, made repeatable: every default key, clicks, right clicks,
// hover and wheel at random places, and resizes, in both modes, at several sizes and in several git states. After
// every event the screen must fit the terminal and nothing may panic.
var sweepSizes = [][2]int{{20, 8}, {36, 16}, {50, 24}, {80, 24}, {120, 40}, {200, 50}}

func sweepKeys() []string {
	var keys []string
	for _, a := range keymap.Actions {
		if a == keymap.Quit {
			continue
		}
		keys = append(keys, keymap.Defaults[a]...)
	}
	// the keys of the popups and prompts, which no action names
	keys = append(keys, "esc", "enter", "y", "n", "up", "down", "left", "right", "backspace", "tab", "a", "z", "o", "u", "d", "L", "P", "1", "2")
	slices.Sort(keys)
	return slices.Compact(keys)
}

func (e *env) mustFit(step string) {
	e.t.Helper()
	w, h := e.m.w, e.m.h
	lines := e.screen()
	if len(lines) > h {
		e.t.Fatalf("%s: %d lines on a %dx%d terminal:\n%s", step, len(lines), w, h, e.text())
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got > w {
			raw := strings.Split(e.m.View().Content, "\n")
			e.t.Fatalf("%s: line %d is %d wide on a %dx%d terminal: %q\nraw: %q\nstatus: %q\n%s", step, i, got, w, h, l, raw[i], e.m.status, e.text())
		}
	}
}

func (e *env) sweep(seed int64, events int) {
	e.t.Helper()
	rng := rand.New(rand.NewSource(seed))
	keys := sweepKeys()
	var trail []string
	for i := 0; i < events; i++ {
		var step string
		switch n := rng.Intn(10); {
		case n < 5:
			k := keys[rng.Intn(len(keys))]
			step = "key " + k
			e.key(k)
		case n < 7:
			x, y := rng.Intn(e.m.w), rng.Intn(e.m.h)
			step = fmt.Sprintf("click %d,%d", x, y)
			e.click(x, y)
		case n < 8:
			x, y := rng.Intn(e.m.w), rng.Intn(e.m.h)
			step = fmt.Sprintf("right click %d,%d", x, y)
			e.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
		case n < 9:
			x, y := rng.Intn(e.m.w), rng.Intn(e.m.h)
			step = fmt.Sprintf("hover %d,%d", x, y)
			e.send(tea.MouseMotionMsg{X: x, Y: y})
		default:
			b := tea.MouseWheelDown
			if rng.Intn(2) == 0 {
				b = tea.MouseWheelUp
			}
			step = fmt.Sprintf("wheel %v", b)
			e.send(tea.MouseWheelMsg{X: rng.Intn(e.m.w), Y: rng.Intn(e.m.h), Button: b})
		}
		trail = append(trail, step)
		e.sweepCheck(trail)
		e.stayHome()
	}
}

func (e *env) sweepCheck(trail []string) {
	e.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			e.t.Fatalf("panic %v after %v", r, trail[max(0, len(trail)-12):])
		}
	}()
	e.mustFit(fmt.Sprintf("after %v", trail[max(0, len(trail)-6):]))
}

func TestSweepNormalWindowKeepsTheScreenInsideTheTerminal(t *testing.T) {
	for _, sz := range sweepSizes {
		t.Run(fmt.Sprintf("%dx%d", sz[0], sz[1]), func(t *testing.T) {
			e := newEnv(t)
			e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			e.sweep(int64(sz[0]*100+sz[1])+sweepSeed(), 400)
		})
	}
}

func TestSweepSidebarModeKeepsTheScreenInsideTheTerminal(t *testing.T) {
	for _, sz := range sweepSizes {
		t.Run(fmt.Sprintf("%dx%d", sz[0], sz[1]), func(t *testing.T) {
			e := projectEnv(t, sz[0])
			e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			e.sweep(int64(sz[0]*7+sz[1])+sweepSeed(), 400)
		})
	}
}

func TestSweepSurvivesResizesInBothModes(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprintf("project=%v", project), func(t *testing.T) {
			e := newEnv(t)
			if project {
				e = projectEnv(t, 50)
			}
			rng := rand.New(rand.NewSource(7))
			for i := 0; i < 150; i++ {
				e.send(tea.WindowSizeMsg{Width: 1 + rng.Intn(220), Height: 1 + rng.Intn(60)})
				e.key(sweepKeys()[rng.Intn(len(sweepKeys()))])
				e.mustFit(fmt.Sprintf("resize %dx%d", e.m.w, e.m.h))
			}
		})
	}
}

// sweepSeed lets a person widen the search: LAZYFINDER_SWEEP=17 go test ./internal/ui -run TestSweep.
func sweepSeed() int64 {
	var n int64
	fmt.Sscan(os.Getenv("LAZYFINDER_SWEEP"), &n)
	return n * 1000003
}

// Found by the sweep: the search chips of a narrow terminal are cut, but their hover area was not.
func TestHoverOnACutSearchChipStaysInsideTheTerminal(t *testing.T) {
	e := projectEnvWithoutGit(t, 36)
	e.send(tea.WindowSizeMsg{Width: 36, Height: 16})
	e.key("/")
	e.send(tea.MouseMotionMsg{X: 29, Y: 14})
	e.mustFit("hover")
}

// Found by the sweep: Get Info was 58 cells wide whatever the terminal.
func TestGetInfoFitsTheNarrowestSidebar(t *testing.T) {
	e := projectEnvWithoutGit(t, 20)
	e.send(tea.WindowSizeMsg{Width: 20, Height: 12})
	e.key("i")
	e.mustHave("Get Info")
	e.mustFit("info")
}

// The same sweep with the Git tab open on a repository with staged, unstaged and stashed work, so the keys and
// clicks land on the rows, buttons, branch menu and diff.
func TestSweepGitTabKeepsTheScreenInsideTheTerminal(t *testing.T) {
	for _, sz := range [][2]int{{24, 12}, {36, 16}, {50, 24}, {120, 40}} {
		t.Run(fmt.Sprintf("%dx%d", sz[0], sz[1]), func(t *testing.T) {
			e := projectEnvWithoutGit(t, sz[0])
			cleanRepo(e)
			_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("stashed\n"), 0o644)
			e.git("stash", "-q")
			_ = os.WriteFile(filepath.Join(e.root, "notes.txt"), []byte("changed\n"), 0o644)
			_ = os.WriteFile(filepath.Join(e.root, "loose.txt"), []byte("x\n"), 0o644)
			_ = os.WriteFile(filepath.Join(e.root, "data.xlsx"), []byte("PK2"), 0o644)
			e.git("add", "data.xlsx")
			e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			e.key("r")
			_, spans := e.m.projectTabsLine()
			e.click(spans[tabGit].x+1, 0)
			e.sweep(int64(sz[0])+sweepSeed(), 500)
		})
	}
}

// Found by the Git tab sweep: git's "would be overwritten by merge" error has a tab, which the terminal expands.
func TestAnErrorWithATabOrNewlineStaysOnOneRow(t *testing.T) {
	e := newEnv(t)
	e.m.setError(errors.New("overwritten by merge:\n\tnotes.txt\nPlease commit\x1b[31m or stash"))
	e.mustFit("error")
	e.mustHave("overwritten by merge: notes.txt Please commit [31m or stash")
}

// stayHome brings the sweep back when a random key took it outside the temporary home: a recursive search from the
// real root of the disk would walk the user's files, and a test never touches them.
func (e *env) stayHome() {
	e.t.Helper()
	if e.m.dir() == e.home || strings.HasPrefix(e.m.dir(), e.home+string(filepath.Separator)) {
		return
	}
	e.key("esc")
	if cmd := e.m.enter(e.root, ""); cmd != nil {
		if msg := cmd(); msg != nil {
			e.send(msg)
		}
	}
}
