package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls [][]string
	dirs  []string // the folder of each call made with StartIn
}

func (f *fakeRunner) StartIn(dir string, argv []string) error {
	f.mu.Lock()
	f.dirs = append(f.dirs, dir)
	f.mu.Unlock()
	return f.Start(argv)
}

func (f *fakeRunner) Start(argv []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, argv)
	return nil
}

type env struct {
	t    *testing.T
	root string
	home string
	run  *fakeRunner
	m    *Model
	now  time.Time
}

const cfgJSON = `{
  "version": 1,
  "density": "compact",
  "openers": {
    "edit": {"argv": ["myeditor", "{file}"], "mode": "gui"},
    "vim":  {"argv": ["vim", "{file}"], "mode": "terminal"}
  },
  "open_with": {"txt": "edit"},
  "sidebar": {
    "favorites": [{"name": "Home", "path": "~"}, {"name": "Gone", "path": "~/missing"}],
    "locations": [{"name": "Disk", "path": "/"}]
  }
}`

func newEnv(t *testing.T) *env {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, "proj")
	for _, d := range []string{"alpha", "beta", "alpha/deep"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"notes.txt":        "first line\nsecond line\n",
		"data.xlsx":        "PK",
		"alpha/inner.txt":  "inner text\n",
		"alpha/deep/x.txt": "deep\n",
		".secret":          "hidden",
	}
	for n, body := range files {
		if err := os.WriteFile(filepath.Join(root, n), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Parse([]byte(cfgJSON), home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { i18n.SetLanguage("en") }) // a random sweep can reach the Language setting: no test inherits it
	e := &env{t: t, root: root, home: home, run: &fakeRunner{}, now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	e.m = New(Deps{Home: home, Start: root, Cfg: cfg, Runner: e.run, Now: func() time.Time { return e.now }})
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	return e
}

// send feeds msg to the model and then the messages produced by the resulting commands.
func (e *env) send(msg tea.Msg) {
	e.t.Helper()
	var cmds []tea.Cmd
	next, cmd := e.m.Update(msg)
	e.m = next.(*Model)
	cmds = append(cmds, cmd)
	for len(cmds) > 0 {
		c := cmds[0]
		cmds = cmds[1:]
		if c == nil {
			continue
		}
		out := c()
		if b, ok := out.(tea.BatchMsg); ok {
			cmds = append(cmds, b...)
			continue
		}
		if out == nil {
			continue
		}
		next, cmd := e.m.Update(out)
		e.m = next.(*Model)
		cmds = append(cmds, cmd)
	}
}

func (e *env) key(k string) {
	e.t.Helper()
	e.send(keyMsg(k))
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "alt+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}
	case "alt+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}
	case "ctrl+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}
	case "ctrl+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl}
	case "ctrl+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
	case "ctrl+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "f1":
		return tea.KeyPressMsg{Code: tea.KeyF1}
	case "f2":
		return tea.KeyPressMsg{Code: tea.KeyF2}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	}
	if strings.HasPrefix(k, "ctrl+") {
		return tea.KeyPressMsg{Code: []rune(k)[5], Mod: tea.ModCtrl}
	}
	r := []rune(k)
	return tea.KeyPressMsg{Code: r[0], Text: k}
}

func (e *env) screen() []string {
	return strings.Split(ansi.Strip(e.m.View().Content), "\n")
}

func (e *env) text() string { return strings.Join(e.screen(), "\n") }

// find returns the position of the first occurrence of s on screen.
func (e *env) find(s string) (x, y int) {
	e.t.Helper()
	for y, line := range e.screen() {
		if i := strings.Index(line, s); i >= 0 {
			return ansi.StringWidth(line[:i]), y
		}
	}
	e.t.Fatalf("%q not on screen:\n%s", s, e.text())
	return 0, 0
}

func (e *env) click(x, y int) {
	e.t.Helper()
	e.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
}

func (e *env) clickText(s string) {
	e.t.Helper()
	x, y := e.find(s)
	e.click(x, y)
}

func (e *env) has(s string) bool { return strings.Contains(e.text(), s) }

func (e *env) mustHave(s string) {
	e.t.Helper()
	if !e.has(s) {
		e.t.Fatalf("expected %q on screen:\n%s", s, e.text())
	}
}

func (e *env) mustNotHave(s string) {
	e.t.Helper()
	if e.has(s) {
		e.t.Fatalf("did not expect %q on screen:\n%s", s, e.text())
	}
}

// row returns the screen line that contains s.
func (e *env) row(s string) string {
	_, y := e.find(s)
	return e.screen()[y]
}

func TestInitialScreenShowsListColumnsSidebarBreadcrumb(t *testing.T) {
	e := newEnv(t)
	for _, s := range []string{"Name", "Date Modified", "Date Created", "Size", "Kind",
		"Favorites", "Locations", "Home", "Disk", "alpha", "beta", "notes.txt", "Folder", "Plain Text", "proj"} {
		e.mustHave(s)
	}
	e.mustNotHave(".secret")
	if got := e.screen(); len(got) != 40 {
		t.Fatalf("screen has %d lines, want 40", len(got))
	}
	for i, l := range e.screen() {
		if w := ansi.StringWidth(l); w > 120 {
			t.Fatalf("line %d is %d wide", i, w)
		}
	}
}

func TestFolderMarkerAndFoldersFirst(t *testing.T) {
	e := newEnv(t)
	if !strings.Contains(e.row("alpha"), ">") {
		t.Fatalf("folder row needs the > marker: %q", e.row("alpha"))
	}
	_, ya := e.find("alpha")
	_, yn := e.find("notes.txt")
	if ya >= yn {
		t.Fatal("folders must come first")
	}
}

func TestKeyboardNavigation(t *testing.T) {
	e := newEnv(t)
	e.key("enter") // first entry is alpha
	e.mustHave("inner.txt")
	e.key("enter") // deep
	e.mustHave("x.txt")
	e.key("backspace")
	e.mustHave("inner.txt")
	e.key("left") // parent
	e.mustHave("notes.txt")
	// parent re-selects the folder we left
	if !strings.Contains(e.row("alpha"), "alpha") {
		t.Fatal("alpha must be visible")
	}
	e.key("down")
	e.key("up")
	e.key("l")
	e.mustHave("inner.txt")
}

func TestHiddenToggle(t *testing.T) {
	e := newEnv(t)
	e.key(".")
	e.mustHave(".secret")
	e.key(".")
	e.mustNotHave(".secret")
}

func TestMouseClickSelectsAndDoubleClickEnters(t *testing.T) {
	e := newEnv(t)
	x, y := e.find("beta")
	e.click(x, y)
	e.mustNotHave("inner.txt")
	e.now = e.now.Add(100 * time.Millisecond)
	e.click(x, y)
	if e.m.dir() != filepath.Join(e.root, "beta") {
		t.Fatalf("dir = %q after double click", e.m.dir())
	}
	// slow second click only selects
	e = newEnv(t)
	x, y = e.find("alpha")
	e.click(x, y)
	e.now = e.now.Add(2 * time.Second)
	e.click(x, y)
	if e.m.dir() != e.root {
		t.Fatalf("slow clicks must not enter: %q", e.m.dir())
	}
}

func TestReachNestedFolderWithClicksOnly(t *testing.T) {
	e := newEnv(t)
	dbl := func(s string) {
		x, y := e.find(s)
		e.click(x, y)
		e.now = e.now.Add(50 * time.Millisecond)
		e.click(x, y)
		e.now = e.now.Add(2 * time.Second)
	}
	dbl("alpha")
	dbl("deep")
	if e.m.dir() != filepath.Join(e.root, "alpha", "deep") {
		t.Fatalf("dir = %q", e.m.dir())
	}
	e.mustHave("x.txt")
}

func TestBackForwardButtonsAndKeys(t *testing.T) {
	e := newEnv(t)
	e.key("enter") // alpha
	e.clickText("‹")
	if e.m.dir() != e.root {
		t.Fatalf("back button: dir = %q", e.m.dir())
	}
	e.clickText("›")
	if e.m.dir() != filepath.Join(e.root, "alpha") {
		t.Fatalf("forward button: dir = %q", e.m.dir())
	}
	e.key("alt+left")
	if e.m.dir() != e.root {
		t.Fatal("alt+left should go back")
	}
	e.key("alt+right")
	if e.m.dir() != filepath.Join(e.root, "alpha") {
		t.Fatal("alt+right should go forward")
	}
}

func TestBreadcrumbSegmentsAreClickable(t *testing.T) {
	e := newEnv(t)
	e.key("enter")
	e.key("enter") // alpha/deep
	// the breadcrumb is on the second-to-last row; click the "proj" segment there
	y := len(e.screen()) - 2
	line := e.screen()[y]
	i := strings.Index(line, "proj")
	if i < 0 {
		t.Fatalf("breadcrumb row = %q", line)
	}
	e.click(ansi.StringWidth(line[:i]), y)
	if e.m.dir() != e.root {
		t.Fatalf("dir = %q", e.m.dir())
	}
}

func TestSidebarClickAndMissingPlace(t *testing.T) {
	e := newEnv(t)
	e.key("enter")
	e.clickText("Home")
	if e.m.dir() != e.home {
		t.Fatalf("dir = %q", e.m.dir())
	}
	e.clickText("Gone") // missing: dimmed, not an error and not navigated
	if e.m.dir() != e.home {
		t.Fatal("missing place must not navigate")
	}
}

func TestTabMovesFocusToSidebar(t *testing.T) {
	e := newEnv(t)
	e.key("tab")
	e.key("down") // Recents, Home, Gone, Disk
	e.key("down")
	e.key("down")
	e.key("enter")
	if e.m.dir() != "/" {
		t.Fatalf("dir = %q", e.m.dir())
	}
}

func TestHeaderClickSortsAndReverses(t *testing.T) {
	e := newEnv(t)
	e.clickText("Size")
	_, y1 := e.find("notes.txt")
	_, y2 := e.find("data.xlsx")
	if y1 >= y2 { // notes.txt (22 bytes) < data.xlsx? sizes: notes 22, data 2 -> ascending puts data first
		// ascending by size: data.xlsx (2) before notes.txt (22)
	} else {
		t.Fatalf("size ascending expected data.xlsx before notes.txt")
	}
	e.clickText("Size")
	_, y1 = e.find("notes.txt")
	_, y2 = e.find("data.xlsx")
	if y1 > y2 {
		t.Fatalf("size descending expected notes.txt before data.xlsx")
	}
	e.mustHave("▼")
}

func TestSortKeys(t *testing.T) {
	e := newEnv(t)
	e.key("s") // name -> modified
	e.mustHave("▲")
	e.key("S")
	e.mustHave("▼")
}

func TestOpenFileUsesConfiguredOpener(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	e.key("enter")
	if len(e.run.calls) != 1 {
		t.Fatalf("calls = %v", e.run.calls)
	}
	want := []string{"myeditor", filepath.Join(e.root, "notes.txt")}
	if strings.Join(e.run.calls[0], "|") != strings.Join(want, "|") {
		t.Fatalf("argv = %v, want %v", e.run.calls[0], want)
	}
}

func TestOpenFileWithoutRuleUsesSystemOpen(t *testing.T) {
	e := newEnv(t)
	e.clickText("data.xlsx")
	e.key("enter")
	if len(e.run.calls) != 1 || e.run.calls[0][0] != "open" || e.run.calls[0][1] != "--" {
		t.Fatalf("calls = %v", e.run.calls)
	}
}

func TestOpenWithMenuListsChoicesAndRunsSelection(t *testing.T) {
	e := newEnv(t)
	e.clickText("data.xlsx")
	e.key("O")
	for _, s := range []string{"Open With", "edit", "vim", "default"} {
		e.mustHave(s)
	}
	e.clickText("edit")
	if len(e.run.calls) != 1 || e.run.calls[0][0] != "myeditor" {
		t.Fatalf("calls = %v", e.run.calls)
	}
	e.mustNotHave("Open With")
}

func TestOpenWithMenuEscCancels(t *testing.T) {
	e := newEnv(t)
	e.clickText("data.xlsx")
	e.key("O")
	e.key("esc")
	e.mustNotHave("Open With")
	if len(e.run.calls) != 0 {
		t.Fatal("nothing must run")
	}
}

func TestSpacePreviewShowsTextAndClosesWithEsc(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	e.key("space")
	e.mustHave("first line")
	e.mustHave("second line")
	e.key("esc")
	e.mustNotHave("first line")
	e.key("space")
	e.key("space")
	e.mustNotHave("first line")
}

func TestPreviewFollowsSelectionWhileOpen(t *testing.T) {
	e := newEnv(t)
	e.clickText("notes.txt")
	e.key("space")
	e.mustHave("first line")
	e.key("down") // data.xlsx? order: ..., data.xlsx before notes.txt, so notes is last; go up instead
	e.key("up")
	e.mustHave("PK")
}

func TestPreviewOfFolderListsItsEntries(t *testing.T) {
	e := newEnv(t)
	e.key("space") // alpha is selected first
	e.mustHave("inner.txt")
	e.mustHave("deep/")
}

func TestTooSmallTerminal(t *testing.T) {
	e := newEnv(t)
	e.send(tea.WindowSizeMsg{Width: 30, Height: 4}) // narrower than 60 the window becomes the sidebar, which has its own minimum
	e.mustHave("too small")
	e.send(tea.WindowSizeMsg{Width: 100, Height: 30}) // and 80 columns bring the normal window back
	e.mustNotHave("too small")
	e.mustHave("Name")
	e.send(tea.WindowSizeMsg{Width: 60, Height: 16})
	e.mustHave("Name")
	for _, sz := range [][2]int{{80, 24}, {60, 16}, {200, 60}} {
		e.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		if got := len(e.screen()); got != sz[1] {
			t.Fatalf("%v: %d lines", sz, got)
		}
		for _, l := range e.screen() {
			if ansi.StringWidth(l) > sz[0] {
				t.Fatalf("%v: line too wide: %q", sz, l)
			}
		}
	}
}

func TestUnreadableFolderShowsErrorAndStays(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip()
	}
	e := newEnv(t)
	locked := filepath.Join(e.root, "beta")
	_ = os.Chmod(locked, 0)
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	e.clickText("beta")
	e.key("enter")
	e.mustHave("cannot read folder")
	if e.m.dir() != e.root {
		t.Fatalf("dir = %q", e.m.dir())
	}
}

func TestQuitReturnsQuitCmd(t *testing.T) {
	e := newEnv(t)
	_, cmd := e.m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("got %T", cmd())
	}
}

func TestViewRequestsMouseAndAltScreen(t *testing.T) {
	e := newEnv(t)
	v := e.m.View()
	if !v.AltScreen || v.MouseMode == tea.MouseModeNone {
		t.Fatalf("view = alt:%v mouse:%v", v.AltScreen, v.MouseMode)
	}
}

func TestWheelScrollsList(t *testing.T) {
	e := newEnv(t)
	// make the list longer than the screen
	for i := 0; i < 60; i++ {
		_ = os.WriteFile(filepath.Join(e.root, "f"+strings.Repeat("0", 2)+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt"), nil, 0o644)
	}
	e.key("r") // reload
	e.mustNotHave("fzzz")
	for i := 0; i < 12; i++ {
		e.send(tea.MouseWheelMsg{X: 60, Y: 10, Button: tea.MouseWheelDown})
	}
	e.mustNotHave("alpha")
}

func TestMarkersWithoutColor(t *testing.T) {
	// selection must be visible in the plain text (reverse video is not enough): the selected row carries a marker
	e := newEnv(t)
	if !strings.Contains(e.row("alpha"), "▸") && !strings.Contains(e.row("alpha"), "›") {
		// selection is signalled with reverse video AND a leading pointer
		t.Fatalf("selected row has no pointer: %q", e.row("alpha"))
	}
}

func TestImagePreviewUsesPlaceholdersWhenTerminalSupportsKitty(t *testing.T) {
	e := newEnv(t)
	writePNG(t, filepath.Join(e.root, "pic.png"), 64, 32)
	e.key("r")
	e.send(uv.KittyGraphicsEvent{Options: kitty.Options{ID: 31}, Payload: []byte("OK")})
	e.clickText("pic.png")
	e.key("space")
	if !strings.ContainsRune(ansi.Strip(e.m.View().Content), 0x10EEEE) {
		t.Fatalf("expected Kitty placeholders in the popup:\n%s", e.text())
	}
	e.key("esc")
	if strings.ContainsRune(e.m.View().Content, 0x10EEEE) {
		t.Fatal("placeholders must disappear when the popup closes")
	}
}

func TestImagePreviewWithoutSupportOffersQuickLook(t *testing.T) {
	e := newEnv(t)
	writePNG(t, filepath.Join(e.root, "pic.png"), 64, 32)
	e.key("r")
	e.clickText("pic.png")
	e.key("space")
	e.mustHave("Quick Look")
	e.key("Q")
	if len(e.run.calls) != 1 || e.run.calls[0][0] != "qlmanage" || e.run.calls[0][1] != "-p" {
		t.Fatalf("calls = %v", e.run.calls)
	}
}

func TestSpaceKeyStringIsHandled(t *testing.T) {
	if s := keyMsg("space").String(); s != "space" && s != " " {
		t.Fatalf("space key stringifies as %q", s)
	}
}

func TestTerminalOpenerHandsOverTheTerminal(t *testing.T) {
	e := newEnv(t)
	e.clickText("data.xlsx")
	e.key("O")
	e.clickText("vim")
	if len(e.run.calls) != 0 {
		t.Fatalf("terminal openers must not go through the GUI runner: %v", e.run.calls)
	}
}

func TestSessionNeverWritesToTheTree(t *testing.T) {
	e := newEnv(t)
	before := snapshot(t, e.root)
	for _, k := range []string{"down", "enter", "space", "esc", "backspace", "s", "S", ".", "O", "esc", "r", "tab", "esc"} {
		e.key(k)
	}
	e.clickText("Size")
	if after := snapshot(t, e.root); after != before {
		t.Fatalf("tree changed:\nbefore %s\nafter  %s", before, after)
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		fi, _ := d.Info()
		b.WriteString(p + "|" + fi.ModTime().String() + "|" + fi.Mode().String() + "\n")
		return nil
	})
	return b.String()
}

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
}

func TestOverlayKeepsTheLineWidthWhenAWideGlyphStraddlesTheBox(t *testing.T) {
	base := []string{"文件夹文件夹文件夹"} // 18 cells
	for x := 0; x < 10; x++ {
		out := overlay(base, []string{"[]"}, x, 0)
		if got := ansi.StringWidth(out[0]); got != 18 {
			t.Errorf("box at %d: the line is %d wide, want 18: %q", x, got, ansi.Strip(out[0]))
		}
	}
}

// The helper that presses keys must press the key a binding names, or every test that uses it proves nothing.
func TestKeyHelperPressesTheKeyItIsAskedFor(t *testing.T) {
	for a, keys := range keymap.Defaults {
		for _, k := range keys {
			if got := keyMsg(k).String(); got != k {
				t.Errorf("%s: keyMsg(%q) presses %q", a, k, got)
			}
		}
	}
}

// clickTab clicks the middle of a tab of the sidebar mode: the tabs show only their icon, so there is no text to find.
func (e *env) clickTab(t projectTab) {
	e.t.Helper()
	_, spans := e.m.projectTabsLine()
	e.click(spans[t].x+spans[t].w/2, 0)
}
