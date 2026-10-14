package ui

import (
	"os"
	"path/filepath"
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
	if e.m.icons.Style != "emoji" {
		t.Fatalf("style = %q", e.m.icons.Style)
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

func TestEmojiIconsFillThreeCellsAndKeepNamesAligned(t *testing.T) {
	e := newEnv(t)
	e.m.icons.Style = "emoji"
	r := e.row("notes.txt")
	if !strings.Contains(r, "📄 notes.txt") {
		t.Fatalf("row = %q", r)
	}
	if got := e.m.iconW(); got != 3 {
		t.Fatalf("iconW = %d, want 3", got)
	}
}

// The click goes to the segment of the home folder even when a folder above it has the same digits in its name
// (cerebro, 2026-10-14: t.TempDir is ".../TestX2470001387/001", and strings.Index found the "001" of the parent).
func TestThePathBarDrawsAFolderIconBeforeEachSegmentAndKeepsItsClicks(t *testing.T) {
	for _, home := range []string{t.TempDir(), filepath.Join(t.TempDir(), "x001y", "001")} {
		thePathBarKeepsItsClicks(t, home)
	}
}

func thePathBarKeepsItsClicks(t *testing.T, home string) {
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	e := newEnvAt(t, home)
	bar := e.screen()[e.m.h-2]
	if got := strings.Count(bar, nerdFolder); got < 2 {
		t.Fatalf("every segment of the path has a folder icon, found %d in %q", got, bar)
	}
	// the click on the name of the parent folder still goes there
	base := filepath.Base(e.home)
	x := strings.LastIndex(bar[:max(0, strings.LastIndex(bar, filepath.Base(e.root)))], base) // the last segment before proj
	if x < 0 {
		t.Fatalf("%s is not in %q", base, bar)
	}
	e.click(len([]rune(bar[:x])), e.m.h-2)
	if e.m.dir() != e.home {
		t.Fatalf("dir = %s, want %s", e.m.dir(), e.home)
	}
}
