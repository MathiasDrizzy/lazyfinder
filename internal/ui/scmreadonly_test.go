package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/git"
)

// Mathias, 2026-10-14: "no quiero hacer commit en un sidebar … lo de agregar para commit o descartar no, solo ver que
// es lo que cambio". The sidebar is a quick look at what changed; staging, discarding, stashing, switching branches,
// pulling and pushing are left to lazygit.

// unchanged fails when anything the sidebar could have done to the repository happened.
func (e *env) unchanged() {
	e.t.Helper()
	st := e.status()
	if len(st.Staged) != 0 || len(st.Unstaged) != 2 || st.Branch != "main" {
		e.t.Fatalf("the repository changed: staged=%v unstaged=%v branch=%s", st.Staged, st.Unstaged, st.Branch)
	}
	if b, _ := os.ReadFile(filepath.Join(e.root, "notes.txt")); string(b) != "changed\n" {
		e.t.Fatalf("notes.txt lost its change: %q", b)
	}
	if l, _ := git.Stashes(e.root); len(l) != 0 {
		e.t.Fatalf("something was stashed: %v", l)
	}
}

func TestTheGitTabOfTheSidebarOnlyShowsWhatChanged(t *testing.T) {
	e := gitEnv(t, 50)
	e.clickText("notes.txt")
	for _, k := range []string{"s", "u", "d", "delete", "z", "f", "p", "P", "B"} {
		e.key(k)
		if e.m.prompt.kind == promptConfirm {
			e.key("y")
		}
		if e.m.gm.open {
			t.Fatalf("%q opened the branch menu", k)
		}
		e.unchanged()
	}
	if n := e.m.actionsShown(); n != 0 {
		t.Fatalf("the header of the Git tab has no git buttons, got %d", n)
	}
	e.click(e.m.listX()+2, e.m.headerRow()) // the branch name
	if e.m.gm.open {
		t.Fatal("a click on the branch opens no menu")
	}
	for i, r := range e.m.scmRows() {
		e.m.scm.sel = i
		for _, b := range e.m.scmButtons(i) {
			if b.act != scmActOpen {
				t.Fatalf("row %d (%v) offers %q; only the diff is left", i, r.kind, scmGlyph(b.act))
			}
		}
	}
	e.m.scm.sel = 0
	e.clickText("notes.txt")
	e.key("enter")
	e.mustHave("+changed") // looking at the diff stays
}

func TestTheRightClickOfTheSidebarOffersNoGitChanges(t *testing.T) {
	e := projectEnv(t, 50)
	x, y := e.find("notes.txt")
	e.rightClick(x, y)
	e.mustHave("Show changes")
	for _, s := range []string{"Stage", "Unstage", "Discard changes"} {
		if strings.Contains(e.text(), s) {
			t.Fatalf("the sidebar offers %q:\n%s", s, e.text())
		}
	}
}
