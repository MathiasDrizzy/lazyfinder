package ui

import (
	"strings"
	"testing"
)

const (
	nerdFolder = ""
	nerdGo     = ""
)

func TestRowsShowAnIconPerKind(t *testing.T) {
	e := newEnv(t)
	if r := e.row("alpha"); !strings.Contains(r, nerdFolder) {
		t.Fatalf("a closed folder shows the folder icon: %q", r)
	}
	e.key("+") // open alpha
	if r := e.row("alpha"); !strings.Contains(r, "") {
		t.Fatalf("an open folder shows the open icon: %q", r)
	}
	if r := e.row("notes.txt"); !strings.Contains(r, "") {
		t.Fatalf("a text file shows the text icon: %q", r)
	}
}

func TestIconsCanBeSwitchedToAsciiOrOff(t *testing.T) {
	for style, want := range map[string]string{"ascii": "/", "off": ""} {
		e := newEnv(t)
		d := e.m.deps
		d.Cfg.Icons = style
		e.m = New(d)
		e.send(sizeMsg())
		r := e.row("beta")
		if strings.ContainsAny(r, nerdFolder+"") {
			t.Fatalf("%s: no Nerd Font glyph expected: %q", style, r)
		}
		if want != "" && !strings.Contains(r, want+" beta") {
			t.Fatalf("%s: row = %q", style, r)
		}
	}
}

func TestIconMapFromTheConfigOverridesAnExtension(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.Cfg.IconMap = map[string]string{"txt": "T"}
	e.m = New(d)
	e.send(sizeMsg())
	if r := e.row("notes.txt"); !strings.Contains(r, "T notes.txt") {
		t.Fatalf("row = %q", r)
	}
}

func TestSettingsCanSwitchTheIconStyle(t *testing.T) {
	e := newEnv(t)
	e.key(",") // settings
	for e.m.set.sel != setIcons {
		e.key("down")
	}
	if got := e.m.settingValue(setIcons); got != "nerd" {
		t.Fatalf("default = %q", got)
	}
	e.key("right")
	if e.m.icons.Style != "ascii" {
		t.Fatalf("style = %q", e.m.icons.Style)
	}
	e.key("esc")
	if r := e.row("alpha"); strings.Contains(r, nerdFolder) {
		t.Fatalf("the Nerd glyph should be gone: %q", r)
	}
}
