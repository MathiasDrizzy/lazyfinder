package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
)

// Order 2026-10-15-16: as the pane of the herdr plugin the sidebar «siga la carpeta donde esta el space»: it goes to
// the folder of the pane next to it when that folder changes, and a folder picked by hand stays until it does.
func TestTheSidebarFollowsTheFolderOfTheHerdrPane(t *testing.T) {
	e := newEnv(t)
	other := filepath.Join(e.root, "sub")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	next := e.root
	e.m.deps.HerdrDir = func() string { return next }
	if e.m.Init() == nil {
		t.Fatal("no command to poll herdr")
	}
	// Update, not e.send: e.send runs every command, and the poll asks again forever
	follow := func(dir string) { e.m.Update(herdrDirMsg{dir: dir}) }
	follow(e.root)
	if e.m.dir() != e.root {
		t.Fatalf("dir %q", e.m.dir())
	}
	follow(other) // cd in the pane next to it
	if e.m.dir() != other {
		t.Fatalf("did not follow: %q", e.m.dir())
	}
	e.m.enter(e.root, "") // the user goes back by hand
	follow(other)
	if e.m.dir() != e.root {
		t.Fatalf("the same folder again must not undo a choice by hand: %q", e.m.dir())
	}
	follow("") // the sidebar has the focus
	follow(filepath.Join(e.root, "gone"))
	if e.m.dir() != e.root {
		t.Fatalf("a folder that does not exist must not be followed: %q", e.m.dir())
	}
}

func TestWithoutHerdrNothingIsPolled(t *testing.T) {
	e := newEnv(t)
	if _, cmd := e.m.Update(herdrDirMsg{dir: e.root}); cmd != nil {
		t.Fatal("no poll outside herdr")
	}
}

// Mathias, 2026-10-22: "si salgo con q me cambio de space y despues vuelvo al mismo space se abre automaticamente,
// esto no deberia pasar porque me sali". As the pane of the herdr plugin, q tells herdr the sidebar was closed in its
// tab before it quits.
func TestQuittingThePluginPaneHidesTheSidebarInItsTab(t *testing.T) {
	e := newEnv(t)
	hidden := 0
	e.m.deps.HerdrHide = func() { hidden++ }
	e.key("q")
	if hidden != 1 || !e.m.quitting {
		t.Fatalf("hidden %d, quitting %v", hidden, e.m.quitting)
	}
}

// Mathias, 2026-10-22: the sidebar keeps its width when the window changes ("debe ser automatico"), and leaving the
// sidebar mode widens it to what the normal mode needs ("que el recuadro se acomode con el minimo para funcionar").
func TestThePluginPaneKeepsItsWidth(t *testing.T) {
	e := newEnv(t)
	type fit struct{ normal, force bool }
	var fits []fit
	e.m.deps.HerdrFit = func(_ config.Herdr, normal, force bool) error {
		fits = append(fits, fit{normal, force})
		return nil
	}
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	if len(fits) != 1 || fits[0] != (fit{false, false}) {
		t.Fatalf("on a new size: %+v", fits)
	}
	e.key("ctrl+b")
	if len(fits) != 2 || fits[1] != (fit{true, true}) {
		t.Fatalf("leaving the sidebar mode: %+v", fits)
	}
	e.key("ctrl+b")
	if len(fits) != 3 || fits[2] != (fit{false, true}) {
		t.Fatalf("back in the sidebar mode: %+v", fits)
	}
}

// In the pane of the herdr plugin the width follows the mode, not the other way round: herdr widens the pane when the
// screen grows (another monitor), and that must not switch it to the normal mode; ctrl+b does.
func TestThePluginPaneKeepsItsModeWhenHerdrWidensIt(t *testing.T) {
	e := newEnv(t)
	e.m.deps.HerdrFit = func(config.Herdr, bool, bool) error { return nil }
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	e.send(tea.WindowSizeMsg{Width: 67, Height: 30})
	if !e.m.projectMode() {
		t.Fatal("a wider screen switched the plugin pane to the normal mode")
	}
}

// pluginPane starts e as the pane of the herdr plugin does: in the sidebar mode.
func pluginPane(e *env) {
	if !e.m.projectMode() {
		e.m.toggleProject()
	}
}

// Mathias, 2026-10-22: "como plugin cuando apreto espacio ... bug visual y tirita. no deberia hacerse vista previa".
// In the herdr sidebar Space says where to turn the preview on, unless herdr.preview is on.
func TestThePluginPaneLeavesThePreviewOff(t *testing.T) {
	e := newEnv(t)
	e.m.deps.HerdrOpen = func(string, []string, string) error { return nil }
	e.pick("notes.txt")
	e.key("space")
	if e.m.pv.open {
		t.Fatal("the preview opened in the herdr sidebar")
	}
	e.mustHave("Preview in herdr")
	e.m.deps.Cfg.Herdr.Preview = true
	e.key("space")
	if !e.m.pv.open {
		t.Fatal("herdr.preview on: no preview")
	}
}

// Mathias, 2026-10-22: "click y saltar a otra pestaña de herdr con el programa configurado". A terminal program opens
// in a new tab of herdr; with herdr.open_in "pane" it takes the sidebar pane, as outside herdr.
func TestThePluginPaneOpensATerminalProgramInANewTab(t *testing.T) {
	e := newEnv(t)
	var got []string
	e.m.deps.HerdrOpen = func(dir string, argv []string, label string) error {
		got = append([]string{dir, label}, argv...)
		return nil
	}
	e.m.deps.Cfg.Openers = map[string]config.Opener{"micro": {Argv: []string{"micro", config.FilePlaceholder}, Mode: config.ModeTerminal}}
	e.m.deps.Cfg.OpenWith = map[string]string{"txt": "micro"}
	e.m.resolve = opener.New(e.m.deps.Cfg)
	e.pick("notes.txt")
	file := filepath.Join(e.root, "notes.txt")
	_, cmd := e.m.Update(keyMsg("enter"))
	want := []string{e.root, "micro", "micro", file}
	if !slices.Equal(got, want) {
		t.Fatalf("opened %q, want %q", got, want)
	}
	if cmd != nil {
		t.Fatal("the sidebar pane handed itself over too")
	}
	got = nil
	e.m.deps.Cfg.Herdr.OpenIn = "pane"
	if _, cmd := e.m.Update(keyMsg("enter")); got != nil || cmd == nil {
		t.Fatalf("open_in pane: herdr %q, hand over %v", got, cmd != nil)
	}
}

// Mathias, 2026-10-22: "abajo ... un >> que sirva para esconder la barra completa". The herdr sidebar has a button at
// the right end of its last line, beside the key hints, that closes it as q does; herdr.close_button takes it away.
func TestTheCloseButtonOfTheHerdrSidebar(t *testing.T) {
	e := newEnv(t)
	hidden := 0
	e.m.deps.HerdrHide = func() { hidden++ }
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	line := func() string { return strings.TrimRight(e.screen()[e.m.h-1], " ") }
	if !strings.HasSuffix(line(), "«") {
		t.Fatalf("no close button on the last line: %q", line())
	}
	if strings.ContainsAny(e.screen()[e.m.h-2], "«»") {
		t.Fatalf("the branch line keeps no button: %q", e.screen()[e.m.h-2])
	}
	e.m.deps.Cfg.Herdr.Dock = "right"
	if !strings.HasSuffix(line(), "»") {
		t.Fatalf("docked on the right, the button points right: %q", line())
	}
	e.m.deps.Cfg.Herdr.CloseButton = false
	if strings.ContainsAny(line(), "«»") {
		t.Fatalf("herdr.close_button off: %q", line())
	}
	e.m.deps.Cfg.Herdr.CloseButton = true
	e.click(e.m.w-2, e.m.h-1)
	if hidden != 1 || !e.m.quitting {
		t.Fatalf("hidden %d, quitting %v", hidden, e.m.quitting)
	}
}

func TestOutsideHerdrTheLastLineHasNoCloseButton(t *testing.T) {
	e := newEnv(t)
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	if l := e.screen()[e.m.h-1]; strings.ContainsAny(l, "«»") {
		t.Fatalf("%q", l)
	}
}

// Mathias, 2026-10-24: three dots showed between "Salir: q" and the close button although the hints fit.
func TestTheCloseButtonLeavesNoEllipsisWhenTheHintsFit(t *testing.T) {
	e := newEnv(t)
	e.m.deps.HerdrHide = func() {}
	pluginPane(e)
	for _, w := range []int{28, 40, 80} {
		e.send(tea.WindowSizeMsg{Width: w, Height: 30})
		if l := e.screen()[e.m.h-1]; strings.Contains(l, "…") {
			t.Fatalf("width %d: %q", w, l)
		}
	}
	e.send(tea.WindowSizeMsg{Width: 28, Height: 30})
	e.m.status = "a status that is far longer than the sidebar can hold at all"
	if l := e.screen()[e.m.h-1]; !strings.HasSuffix(strings.TrimRight(l, " "), "«") {
		t.Fatalf("a long status hides the button: %q", l)
	}
}

// Mathias, 2026-10-24: "dos botones redundantes ... Salir: q y « ... que no lo tapen". The button is the way to close
// the sidebar, so the hints no longer list Quit while it shows, and the tooltip of the button leaves it in view.
func TestTheCloseButtonReplacesTheQuitHintAndStaysUnderItsTooltip(t *testing.T) {
	e := newEnv(t)
	e.m.deps.HerdrHide = func() {}
	pluginPane(e)
	e.send(tea.WindowSizeMsg{Width: 40, Height: 30})
	last := func() string { return strings.TrimRight(e.screen()[e.m.h-1], " ") }
	if strings.Contains(last(), "Quit") {
		t.Fatalf("the hints still list Quit beside the button: %q", last())
	}
	e.move(e.m.w-2, e.m.h-1)
	if l := last(); !strings.Contains(l, "Close the sidebar: q") || !strings.HasSuffix(l, "«") {
		t.Fatalf("the tooltip hides the button: %q", l)
	}
	e.move(0, 0)
	e.m.deps.Cfg.Herdr.CloseButton = false
	e.send(tea.WindowSizeMsg{Width: 40, Height: 30})
	if !strings.Contains(last(), "Quit: q") {
		t.Fatalf("without the button the hints keep Quit: %q", last())
	}
}
