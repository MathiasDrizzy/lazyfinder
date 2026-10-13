package ui

import (
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const (
	mochaSelectionBg = "48;2;69;71;90" // surface1 of catppuccin-mocha: the selection bar, blue-gray like lazymark's
	lattePeachBg     = "48;2;188;192;204"
	mochaPeachBg     = "48;2;250;179;135" // the old salmon bar, which must not come back
	mochaBase        = "48;2;30;30;46"
)

func raw(e *env) string { return e.m.View().Content }

func withConfigFile(t *testing.T, e *env) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	e.m.deps.ConfigPath = p
	return p
}

func TestListPanelShowsTheCounterAndTheFolderTitle(t *testing.T) {
	e := newEnv(t)
	e.mustHave("╭─ proj ─")
	e.mustHave("1 of 4 ─╯")
	e.key("down")
	e.mustHave("2 of 4 ─╯")
}

func TestKeyBarListsTheEffectiveKeys(t *testing.T) {
	e := newEnv(t)
	for _, s := range []string{"Open: ⏎", "Preview: Space", "Open with: O", "Settings: ,", "Quit: q"} {
		e.mustHave(s)
	}
}

func rawLine(e *env, word string) string {
	for _, l := range strings.Split(raw(e), "\n") {
		if strings.Contains(ansi.Strip(l), word) {
			return l
		}
	}
	return ""
}

// The selection bar is the theme's surface1, a blue-gray, as in lazymark (order 2026-10-13-05, block H).
func TestTheSelectionBarIsBlueGrayLikeLazymark(t *testing.T) {
	e := newEnv(t)
	line := rawLine(e, "▸ >")
	if !strings.Contains(line, mochaSelectionBg) || strings.Contains(raw(e), mochaPeachBg) {
		t.Fatalf("mocha selection bar is not surface1: %q", line)
	}
	e.m.deps.Cfg.Theme = "catppuccin-latte"
	e.m.restyle()
	if got := rawLine(e, "▸ >"); !strings.Contains(got, lattePeachBg) {
		t.Fatalf("latte selection bar not applied: %q", got)
	}
}

func TestScreenBackgroundTheme(t *testing.T) {
	e := newEnv(t)
	if !strings.Contains(raw(e), mochaBase) {
		t.Fatal("the theme background must be painted")
	}
	e.m.deps.Cfg.ScreenBackground = config.ScreenBackgroundTerminal
	e.m.restyle()
	if strings.Contains(raw(e), mochaBase) {
		t.Fatal("screen_background=terminal must leave the terminal's own background")
	}
}

func TestNoColorHasNoColorSequences(t *testing.T) {
	e := newEnv(t)
	e.m.deps.NoColor = true
	e.m.restyle()
	if strings.Contains(raw(e), "38;2;") || strings.Contains(raw(e), "48;2;") {
		t.Fatal("NO_COLOR must not emit colors")
	}
	e.mustHave("▸") // the selection survives without color
}

func TestTerminalThemeUsesNoTrueColor(t *testing.T) {
	e := newEnv(t)
	e.m.deps.Cfg.Theme = "terminal"
	e.m.restyle()
	if strings.Contains(raw(e), "48;2;") {
		t.Fatal("the terminal theme must use the terminal's own colors")
	}
}

func TestSettingsPopupOpensWithKeyAndGearAndCloses(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	e.mustHave("Settings")
	e.mustHave("Theme")
	e.key("esc")
	e.mustNotHave("Screen background")
	e.clickText("⚙")
	e.mustHave("Screen background")
	e.click(0, 20) // outside the popup: like lazymark, only Esc closes it (order 2026-10-13-05, block H)
	e.mustHave("Screen background")
	e.key("esc")
	e.mustNotHave("Screen background")
}

func TestSettingsThemeChangeAppliesAndIsSaved(t *testing.T) {
	e := newEnv(t)
	path := withConfigFile(t, e)
	e.key(",")
	e.key("right") // theme: next in the list
	cfg, err := config.Load(path, e.home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme == "catppuccin-mocha" || cfg.Theme != e.m.deps.Cfg.Theme {
		t.Fatalf("saved theme %q, running theme %q", cfg.Theme, e.m.deps.Cfg.Theme)
	}
}

func TestSettingsLanguageChangesTheTextAtOnce(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	e := newEnv(t)
	withConfigFile(t, e)
	e.key(",")
	e.key("down") // language
	e.key("right")
	e.key("right") // auto -> en -> es
	e.key("esc")
	e.mustHave("Abrir: ⏎")
	e.mustHave("Fecha de modif")
	e.mustHave("Favoritos")
}

func TestSettingsToggleHiddenReloads(t *testing.T) {
	e := newEnv(t)
	withConfigFile(t, e)
	e.key(",")
	for range 3 {
		e.key("down")
	}
	e.key("enter")
	e.key("esc")
	e.mustHave(".secret")
}

func TestSettingsDoesNotOverwriteABrokenConfig(t *testing.T) {
	e := newEnv(t)
	p := withConfigFile(t, e)
	if err := os.WriteFile(p, []byte("{ nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.key(",")
	e.key("right")
	data, _ := os.ReadFile(p)
	if string(data) != "{ nope" {
		t.Fatalf("the file the user is editing was replaced: %s", data)
	}
	e.mustHave("Error:")
}

func TestSettingsWithoutAConfigPathChangesOnlyTheSession(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	e.key("right")
	if e.m.deps.Cfg.Theme == "catppuccin-mocha" {
		t.Fatal("the theme should change for this session")
	}
}

func TestRebindingAKeyMovesTheAction(t *testing.T) {
	e := newEnv(t)
	cfg, err := config.Parse([]byte(`{"version":1,"keys":{"quit":["Z"],"down":["n"]}}`), e.home)
	if err != nil {
		t.Fatal(err)
	}
	e.m = New(Deps{Home: e.home, Start: e.root, Cfg: cfg, Runner: e.run, Now: e.m.deps.Now})
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.mustHave("Quit: Z")
	e.key("n")
	e.mustHave("2 of 4")
	e.key("j") // no longer bound
	e.mustHave("2 of 4")
	if cmd := e.m.onKey(keyMsg("q")); cmd != nil {
		t.Fatal("q must not quit once quit is rebound")
	}
	if cmd := e.m.onKey(keyMsg("Z")); cmd == nil {
		t.Fatal("Z must quit")
	}
}

func TestMouseOffIgnoresClicksAndWheel(t *testing.T) {
	e := newEnv(t)
	e.m.mouse = false
	x, y := e.find("beta")
	e.click(x, y)
	e.mustHave("1 of 4")
	if e.m.View().MouseMode != tea.MouseModeNone {
		t.Fatal("mouse reporting must be off")
	}
}

func TestStartupWarningFromTheConfigIsShown(t *testing.T) {
	e := newEnv(t)
	cfg := e.m.deps.Cfg
	cfg.Warnings = []string{"theme night.json: unreadable"}
	e.m = New(Deps{Home: e.home, Start: e.root, Cfg: cfg, Runner: e.run, Now: e.m.deps.Now})
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.mustHave("night.json")
}

func TestClicksLandOnPanelContentNotBorders(t *testing.T) {
	e := newEnv(t)
	x, y := e.find("beta")
	if z := e.m.hit(x, y); z.kind != zoneRow {
		t.Fatalf("beta row hit = %+v", z)
	}
	if z := e.m.hit(0, 3); z.kind != zoneNone { // sidebar left border
		t.Fatalf("border hit = %+v", z)
	}
	hx, hy := e.find("Date Modified")
	if z := e.m.hit(hx, hy); z.kind != zoneHeader {
		t.Fatalf("header hit = %+v", z)
	}
}
