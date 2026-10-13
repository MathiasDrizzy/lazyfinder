package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

func (e *env) onDisk(rel string) bool {
	_, err := os.Lstat(filepath.Join(e.root, rel))
	return err == nil
}

func (e *env) inTrash(name string) bool {
	_, err := os.Lstat(filepath.Join(e.home, ".Trash", name))
	return err == nil
}

func (e *env) selectedName() string {
	sel, _ := e.m.selected()
	return sel.Name
}

// pick selects the row that shows name.
func (e *env) pick(name string) {
	e.t.Helper()
	e.clickText(name)
}

func (e *env) ctrlClickText(s string) {
	e.t.Helper()
	x, y := e.find(s)
	e.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft, Mod: tea.ModCtrl})
}

func TestVMarksTheRowAndMovesDown(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("up")
	e.key("v")
	e.mustHave("1 selected")
	if !strings.Contains(e.row("notes.txt"), "▸") {
		t.Fatal("v moves the cursor down")
	}
	e.key("up")
	if !strings.Contains(e.row("data.xlsx"), "●") {
		t.Fatalf("a marked row shows a dot:\n%s", e.row("data.xlsx"))
	}
	e.key("v")
	e.mustNotHave("selected")
}

func TestCtrlClickMarksWithoutOpeningAnything(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.ctrlClickText("data.xlsx")
	e.ctrlClickText("alpha")
	e.mustHave("2 selected")
	if e.m.dir() != e.root || len(e.run.calls) != 0 {
		t.Fatal("ctrl-click only marks")
	}
}

func TestCtrlAMarksEverythingAndEscClears(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+a")
	e.mustHave("4 selected")
	e.key("esc")
	e.mustNotHave("selected")
}

func TestMarksClearWhenTheFolderChanges(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("v")
	e.pick("beta")
	e.key("enter")
	e.key("backspace")
	e.mustNotHave("selected")
}

func TestCopyThenPasteIntoAnotherFolderKeepsTheOriginal(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("c")
	e.mustHave("Copied")
	e.pick("beta")
	e.key("enter")
	e.key("p")
	e.mustHave("notes.txt")
	if !e.onDisk("beta/notes.txt") || !e.onDisk("notes.txt") {
		t.Fatal("copy must leave the original and create the copy")
	}
}

func TestCutThenPasteMovesAndEmptiesTheClipboard(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("x")
	e.pick("beta")
	e.key("enter")
	e.key("p")
	if e.onDisk("notes.txt") || !e.onDisk("beta/notes.txt") {
		t.Fatal("cut and paste moves")
	}
	e.key("p")
	e.mustHave("Nothing to paste")
}

func TestPasteInTheSameFolderAfterCopyMakesACopy(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("c")
	e.key("p")
	e.mustHave("notes copy.txt")
	if e.selectedName() != "notes copy.txt" {
		t.Fatalf("the new item is selected, got %s", e.selectedName())
	}
}

func TestCopyOfMarkedItemsPastesThemAll(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.ctrlClickText("data.xlsx")
	e.ctrlClickText("notes.txt")
	e.key("c")
	e.mustHave("Copied 2 items")
	e.pick("beta")
	e.key("enter")
	e.key("p")
	if !e.onDisk("beta/notes.txt") || !e.onDisk("beta/data.xlsx") {
		t.Fatal("both items are pasted")
	}
}

func TestDKeyDuplicates(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("D")
	if !e.onDisk("notes copy.txt") || e.selectedName() != "notes copy.txt" {
		t.Fatalf("duplicate: on disk %v, selected %s", e.onDisk("notes copy.txt"), e.selectedName())
	}
}

func TestRenameWithF2PrefillsTheNameAndRenames(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("f2")
	e.mustHave("Rename:")
	e.mustHave("notes.txt▏")
	e.key("ctrl+u")
	e.typeText("todo.txt")
	e.key("enter")
	if e.onDisk("notes.txt") || !e.onDisk("todo.txt") {
		t.Fatal("the file must be renamed on disk")
	}
	if e.selectedName() != "todo.txt" {
		t.Fatalf("selected %s", e.selectedName())
	}
}

func TestRenameToATakenNameShowsTheErrorAndChangesNothing(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("f2")
	e.key("ctrl+u")
	e.typeText("data.xlsx")
	e.key("enter")
	e.mustHave("already exists")
	if !e.onDisk("notes.txt") {
		t.Fatal("nothing may change")
	}
	e.key("esc")
	e.mustNotHave("Rename:")
}

func TestEscCancelsTheRename(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("f2")
	e.typeText("zzz")
	e.key("esc")
	if !e.onDisk("notes.txt") {
		t.Fatal("cancel keeps the name")
	}
}

func TestNewFolderAsksForANameAndSelectsIt(t *testing.T) {
	e := newEnv(t)
	e.key("N")
	e.mustHave("New folder")
	e.mustHave("untitled folder")
	e.key("ctrl+u")
	e.typeText("fresh")
	e.key("enter")
	fi, err := os.Stat(filepath.Join(e.root, "fresh"))
	if err != nil || !fi.IsDir() {
		t.Fatalf("the folder was not made: %v", err)
	}
	if e.selectedName() != "fresh" {
		t.Fatalf("selected %s", e.selectedName())
	}
}

func TestTrashOfOneItemAsksNothingAndUndoBringsItBack(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("d")
	if e.onDisk("notes.txt") || !e.inTrash("notes.txt") {
		t.Fatal("the file must be in the Trash")
	}
	e.mustNotHave("Plain Text") // notes.txt is the only text file in the folder
	e.mustHave("Trash")
	e.key("u")
	if !e.onDisk("notes.txt") || e.inTrash("notes.txt") {
		t.Fatal("undo brings it back")
	}
	e.mustHave("notes.txt")
}

func TestDeleteKeyTrashesToo(t *testing.T) {
	e := newEnv(t)
	e.pick("data.xlsx")
	e.key("delete")
	if !e.inTrash("data.xlsx") {
		t.Fatal("delete trashes")
	}
}

func TestTrashOfSeveralItemsAsksFirst(t *testing.T) {
	e := newEnv(t)
	e.key("ctrl+a")
	e.key("d")
	e.mustHave("Move 4 items to the Trash?")
	e.key("n")
	if !e.onDisk("notes.txt") {
		t.Fatal("no keeps everything")
	}
	e.key("d")
	e.key("y")
	for _, n := range []string{"notes.txt", "data.xlsx", "alpha", "beta"} {
		if e.onDisk(n) || !e.inTrash(n) {
			t.Fatalf("%s must be in the Trash", n)
		}
	}
	e.key("u")
	if !e.onDisk("alpha/inner.txt") || !e.onDisk("notes.txt") {
		t.Fatal("one undo brings back the whole batch")
	}
}

func TestTheConfirmationThresholdIsConfigurable(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.Cfg.Ops.ConfirmTrashAbove = 0
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.pick("notes.txt")
	e.key("d")
	e.mustHave("Move notes.txt to the Trash?")
	e.key("y")
	if !e.inTrash("notes.txt") {
		t.Fatal("y confirms")
	}
}

func TestUndoWithNothingToUndoSaysSo(t *testing.T) {
	e := newEnv(t)
	e.key("u")
	e.mustHave("Nothing to undo")
}

func TestUndoReversesARename(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("f2")
	e.key("ctrl+u")
	e.typeText("todo.txt")
	e.key("enter")
	e.key("ctrl+z")
	if !e.onDisk("notes.txt") || e.onDisk("todo.txt") {
		t.Fatal("undo renames back")
	}
}

func TestOperationsWorkOnRowsOpenedInPlace(t *testing.T) {
	e := newEnv(t)
	e.pick("alpha")
	e.key("+")
	e.pick("inner.txt")
	e.key("d")
	if e.onDisk("alpha/inner.txt") || !e.inTrash("inner.txt") {
		t.Fatal("the child row is trashed from its own folder")
	}
	e.mustHave("deep")
}

func TestInfoPopupShowsDetailsAndEscClosesIt(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key("i")
	e.mustHave("Get Info")
	e.mustHave("notes.txt")
	e.mustHave("Where")
	e.mustHave("Plain Text")
	e.key("esc")
	e.mustNotHave("Get Info")
}

func TestInfoOfAFolderCountsItsFiles(t *testing.T) {
	e := newEnv(t)
	e.pick("alpha")
	e.key("i")
	e.mustHave("2 files")
}

func TestOperationsAreIgnoredWhileTheSettingsPopupIsOpen(t *testing.T) {
	e := newEnv(t)
	e.pick("notes.txt")
	e.key(",")
	e.key("d")
	if !e.onDisk("notes.txt") {
		t.Fatal("keys belong to the popup")
	}
}

func TestOpsConfigIsRead(t *testing.T) {
	e := newEnv(t)
	cfg, err := config.Parse([]byte(`{"version":1,"ops":{"trash_dir":"~/can"}}`), e.home)
	if err != nil {
		t.Fatal(err)
	}
	d := e.m.deps
	d.Cfg = cfg
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.pick("notes.txt")
	e.key("d")
	if _, err := os.Lstat(filepath.Join(e.home, "can", "notes.txt")); err != nil {
		t.Fatalf("the configured trash is used: %v", err)
	}
}
