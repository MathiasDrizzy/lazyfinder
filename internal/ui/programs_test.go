package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
)

// onPath puts a program called name first on PATH.
func onPath(t *testing.T, name string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func (e *env) openPrograms() string {
	e.t.Helper()
	path := e.withConfigFile()
	e.key(",")
	e.pickSetting("Programs")
	e.key("enter")
	e.mustHave("Add program")
	return path
}

// Mathias, 2026-10-22: "otra seccion para agregar programas los cuales despues yo pueda configurar poder abrir algun
// tipo de archivo con ese programa". Settings → Programs adds a program, says which file types it opens, and removes
// it; config.json keeps each change.
func TestProgramsAddsAProgramAndTheFileTypesItOpens(t *testing.T) {
	onPath(t, "micro")
	e := newEnv(t)
	path := e.openPrograms()
	e.key("a")
	e.typeText("micro")
	e.key("enter")
	want := config.Opener{Argv: []string{"micro", config.FilePlaceholder}, Mode: config.ModeTerminal}
	if got := e.m.deps.Cfg.Openers["micro"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("opener = %+v, want %+v", got, want)
	}
	e.key("enter") // the new program is selected: Enter asks for its file types
	e.typeText("md, .TXT")
	e.key("enter")
	for _, ext := range []string{"md", "txt"} {
		if e.m.deps.Cfg.OpenWith[ext] != "micro" {
			t.Fatalf("open_with = %v", e.m.deps.Cfg.OpenWith)
		}
	}
	e.mustHave("md, txt")
	saved := e.savedConfig(path)
	if !reflect.DeepEqual(saved.Openers["micro"], want) || saved.OpenWith["md"] != "micro" || saved.OpenWith["txt"] != "micro" {
		t.Fatalf("config.json: openers %v, open_with %v", saved.Openers, saved.OpenWith)
	}
	if c := e.m.resolve.Resolve("/x/readme.md"); c.Name != "micro" {
		t.Fatalf("a .md file opens with %q", c.Name)
	}
	e.key("d")
	saved = e.savedConfig(path)
	_, kept := e.m.deps.Cfg.Openers["micro"]
	_, savedKept := saved.Openers["micro"]
	if kept || savedKept || e.m.deps.Cfg.OpenWith["md"] != "" || saved.OpenWith["md"] != "" {
		t.Fatalf("removed: openers %v %v, open_with %v %v", e.m.deps.Cfg.Openers, saved.Openers, e.m.deps.Cfg.OpenWith, saved.OpenWith)
	}
}

// An application name opens with "open -a" in its own window; m switches a program between window and terminal.
func TestProgramsAddsAnApplicationAndSwitchesItsMode(t *testing.T) {
	old := opener.AppExists
	t.Cleanup(func() { opener.AppExists = old })
	opener.AppExists = func(name string) bool { return name == "Sublime Text" }
	e := newEnv(t)
	e.openPrograms()
	e.key("a")
	e.typeText("Sublime Text")
	e.key("enter")
	want := config.Opener{Argv: []string{"open", "-a", "Sublime Text", config.FilePlaceholder}, Mode: config.ModeGUI}
	if got := e.m.deps.Cfg.Openers["sublime-text"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("opener = %+v, want %+v", got, want)
	}
	e.key("m")
	if got := e.m.deps.Cfg.Openers["sublime-text"].Mode; got != config.ModeTerminal {
		t.Fatalf("mode = %q", got)
	}
}

// A program that is not installed, or one already added, is not added; the prompt stays to fix it.
func TestProgramsRefusesAMissingOrRepeatedProgram(t *testing.T) {
	onPath(t, "micro")
	e := newEnv(t)
	e.openPrograms()
	before := len(e.m.deps.Cfg.Openers)
	for _, cmd := range []string{"no-such-editor", "micro", "micro"} {
		if e.m.prompt.kind != promptNone {
			e.key("esc") // the prompt the last attempt left open
		}
		e.key("a")
		e.typeText(cmd)
		e.key("enter")
	}
	if len(e.m.deps.Cfg.Openers) != before+1 || !e.m.statusErr || e.m.prompt.kind == promptNone {
		t.Fatalf("openers %v, error %v %q", e.m.deps.Cfg.Openers, e.m.statusErr, e.m.status)
	}
}
