package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The second key of an action is a key too: each alternative that no other test presses is pressed here, with its
// effect (order 07, point O).

func bigFolderEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t)
	for i := 0; i < 100; i++ {
		if err := os.WriteFile(filepath.Join(e.root, fmt.Sprintf("f%03d.txt", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e.key("r")
	return e
}

func TestPageKeysHomeAndEndMoveTheSelection(t *testing.T) {
	e := bigFolderEnv(t)
	e.key("pgdown")
	down := e.m.sel
	if down < 5 {
		t.Fatalf("pgdown moved to %d", down)
	}
	e.key("pgup")
	if e.m.sel >= down {
		t.Fatalf("pgup stayed at %d", e.m.sel)
	}
	e.key("end")
	if got := e.m.sel; got != len(e.m.entries())-1 {
		t.Fatalf("end: %d", got)
	}
	e.key("home")
	if e.m.sel != 0 {
		t.Fatalf("home: %d", e.m.sel)
	}
	e.key("ctrl+d")
	if e.m.sel == 0 {
		t.Fatal("ctrl+d is page down")
	}
}

func TestBracketsGoBackAndForward(t *testing.T) {
	e := newEnv(t)
	e.key("enter") // alpha
	e.key("[")
	if e.m.dir() != e.root {
		t.Fatalf("[ should go back, dir = %q", e.m.dir())
	}
	e.key("]")
	if e.m.dir() != filepath.Join(e.root, "alpha") {
		t.Fatalf("] should go forward, dir = %q", e.m.dir())
	}
}

func TestTheOtherKeysOfAnActionDoTheSame(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"ctrl+f searches", []string{"ctrl+f"}, "Look in"},
		{"f1 opens the help", []string{"f1"}, "Esc"},
		{"R renames", []string{"R"}, "Rename:"},
		{"= expands like +", []string{"="}, "inner.txt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			for _, k := range c.keys {
				e.key(k)
			}
			e.mustHave(c.want)
		})
	}
}

func TestCtrlVPastesLikeP(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("c")
	e.pick("alpha")
	e.key("enter")
	e.key("ctrl+v")
	if !e.onDisk("alpha/notes.txt") {
		t.Fatal("ctrl+v should paste the copied file")
	}
}
