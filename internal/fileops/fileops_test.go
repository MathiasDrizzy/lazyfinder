package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// world is a temp folder with a source tree, a destination and a trash can, so nothing real is ever touched.
type world struct {
	t                *testing.T
	root, dst, trash string
	ops              *Ops
}

func newWorld(t *testing.T) *world {
	t.Helper()
	root := t.TempDir()
	w := &world{t: t, root: root, dst: filepath.Join(root, "dst"), trash: filepath.Join(root, "trash")}
	for _, d := range []string{w.dst, filepath.Join(root, "src", "tree", "deep")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	w.put("src/a.txt", "alpha")
	w.put("src/photo.png", "png")
	w.put("src/tree/in.txt", "inner")
	w.put("src/tree/deep/x.txt", "deepest")
	w.ops = New(w.trash)
	return w
}

func (w *world) path(rel string) string { return filepath.Join(w.root, rel) }

func (w *world) put(rel, body string) {
	w.t.Helper()
	if err := os.WriteFile(w.path(rel), []byte(body), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) read(rel string) string {
	w.t.Helper()
	b, err := os.ReadFile(w.path(rel))
	if err != nil {
		w.t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func (w *world) exists(rel string) bool {
	_, err := os.Lstat(w.path(rel))
	return err == nil
}

func (w *world) must(err error) {
	w.t.Helper()
	if err != nil {
		w.t.Fatal(err)
	}
}

func TestUniqueNameFollowsFinder(t *testing.T) {
	w := newWorld(t)
	for _, tc := range []struct{ name, want string }{
		{"free.txt", "free.txt"},
		{"a.txt", "a copy.txt"},
		{"tree", "tree copy"},
	} {
		if got := UniqueName(filepath.Join(w.root, "src"), tc.name); got != tc.want {
			t.Errorf("UniqueName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
	w.put("src/a copy.txt", "x")
	if got := UniqueName(filepath.Join(w.root, "src"), "a.txt"); got != "a copy 2.txt" {
		t.Errorf("second copy = %q, want %q", got, "a copy 2.txt")
	}
	w.put("src/.env", "x")
	if got := UniqueName(filepath.Join(w.root, "src"), ".env"); got != ".env copy" {
		t.Errorf("a dotfile has no extension: %q", got)
	}
}

func TestCopyCopiesFilesAndWholeTrees(t *testing.T) {
	w := newWorld(t)
	made, err := w.ops.Copy([]string{w.path("src/a.txt"), w.path("src/tree")}, w.dst)
	w.must(err)
	if len(made) != 2 || made[0] != w.path("dst/a.txt") || made[1] != w.path("dst/tree") {
		t.Fatalf("made = %v", made)
	}
	if w.read("dst/a.txt") != "alpha" || w.read("dst/tree/deep/x.txt") != "deepest" {
		t.Fatal("content differs")
	}
	if !w.exists("src/a.txt") {
		t.Fatal("copy must leave the source")
	}
}

func TestCopyNeverOverwritesItPicksANewName(t *testing.T) {
	w := newWorld(t)
	w.put("dst/a.txt", "mine")
	made, err := w.ops.Copy([]string{w.path("src/a.txt")}, w.dst)
	w.must(err)
	if made[0] != w.path("dst/a copy.txt") || w.read("dst/a.txt") != "mine" || w.read("dst/a copy.txt") != "alpha" {
		t.Fatalf("made = %v, original = %q", made, w.read("dst/a.txt"))
	}
}

func TestCopyKeepsModeTimeAndSymlinks(t *testing.T) {
	w := newWorld(t)
	script := w.path("src/run.sh")
	w.put("src/run.sh", "#!/bin/sh")
	w.must(os.Chmod(script, 0o755))
	w.must(os.Symlink("a.txt", w.path("src/link")))
	_, err := w.ops.Copy([]string{script, w.path("src/link")}, w.dst)
	w.must(err)
	fi, err := os.Stat(w.path("dst/run.sh"))
	w.must(err)
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v", fi.Mode().Perm())
	}
	src, _ := os.Stat(script)
	if !fi.ModTime().Equal(src.ModTime()) {
		t.Errorf("mtime = %v, want %v", fi.ModTime(), src.ModTime())
	}
	target, err := os.Readlink(w.path("dst/link"))
	if err != nil || target != "a.txt" {
		t.Errorf("link = %q, %v: a symlink is copied as a symlink", target, err)
	}
}

func TestCopyAFolderIntoItselfIsRefused(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Copy([]string{w.path("src/tree")}, w.path("src/tree/deep"))
	if err == nil || !strings.Contains(err.Error(), "itself") {
		t.Fatalf("err = %v", err)
	}
	if w.exists("src/tree/deep/tree") {
		t.Fatal("nothing may be created")
	}
}

func TestMoveMovesAndKeepsTheContent(t *testing.T) {
	w := newWorld(t)
	made, err := w.ops.Move([]string{w.path("src/a.txt"), w.path("src/tree")}, w.dst)
	w.must(err)
	if w.exists("src/a.txt") || w.exists("src/tree") {
		t.Fatal("the source must be gone")
	}
	if len(made) != 2 || w.read("dst/a.txt") != "alpha" || w.read("dst/tree/deep/x.txt") != "deepest" {
		t.Fatalf("made = %v", made)
	}
}

func TestMoveIntoTheSameFolderChangesNothing(t *testing.T) {
	w := newWorld(t)
	made, err := w.ops.Move([]string{w.path("src/a.txt")}, w.path("src"))
	w.must(err)
	if len(made) != 0 || w.read("src/a.txt") != "alpha" || w.exists("src/a copy.txt") {
		t.Fatalf("made = %v", made)
	}
}

func TestMoveAFolderIntoItselfIsRefused(t *testing.T) {
	w := newWorld(t)
	if _, err := w.ops.Move([]string{w.path("src/tree")}, w.path("src/tree/deep")); err == nil {
		t.Fatal("expected an error")
	}
	if !w.exists("src/tree/in.txt") {
		t.Fatal("the folder must be untouched")
	}
}

func TestRenameChangesTheNameOnly(t *testing.T) {
	w := newWorld(t)
	got, err := w.ops.Rename(w.path("src/a.txt"), "b.txt")
	w.must(err)
	if got != w.path("src/b.txt") || w.exists("src/a.txt") || w.read("src/b.txt") != "alpha" {
		t.Fatalf("got = %s", got)
	}
}

func TestRenameRefusesBadNames(t *testing.T) {
	w := newWorld(t)
	for _, tc := range []struct{ name, why string }{
		{"", "empty"}, {"  ", "blank"}, {"x/y", "slash"}, {".", "dot"}, {"..", "dotdot"},
		{"photo.png", "taken"}, {"a\x00b", "nul"},
	} {
		if _, err := w.ops.Rename(w.path("src/a.txt"), tc.name); err == nil {
			t.Errorf("%s: %q must be refused", tc.why, tc.name)
		}
	}
	if w.read("src/a.txt") != "alpha" {
		t.Fatal("the file must be untouched")
	}
}

func TestRenameToTheSameNameIsANoOp(t *testing.T) {
	w := newWorld(t)
	got, err := w.ops.Rename(w.path("src/a.txt"), "a.txt")
	w.must(err)
	if got != w.path("src/a.txt") || len(w.ops.undo) != 0 {
		t.Fatalf("got = %s, undo steps = %d", got, len(w.ops.undo))
	}
}

func TestMkdirCreatesAFolderAndRefusesTaken(t *testing.T) {
	w := newWorld(t)
	got, err := w.ops.Mkdir(w.path("src"), "fresh")
	w.must(err)
	if fi, err := os.Stat(got); err != nil || !fi.IsDir() {
		t.Fatalf("not a folder: %v", err)
	}
	if _, err := w.ops.Mkdir(w.path("src"), "fresh"); err == nil {
		t.Fatal("an existing name must be refused")
	}
	if _, err := w.ops.Mkdir(w.path("src"), "a/b"); err == nil {
		t.Fatal("a slash must be refused")
	}
}

func TestTrashMovesToTheTrashAndNeverDeletes(t *testing.T) {
	w := newWorld(t)
	w.must(w.ops.Trash([]string{w.path("src/a.txt"), w.path("src/tree")}))
	if w.exists("src/a.txt") || w.exists("src/tree") {
		t.Fatal("the items must leave their folder")
	}
	if w.read("trash/a.txt") != "alpha" || w.read("trash/tree/deep/x.txt") != "deepest" {
		t.Fatal("the content must be in the trash")
	}
}

func TestTrashKeepsBothWhenNamesCollide(t *testing.T) {
	w := newWorld(t)
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	w.put("src/a.txt", "newer")
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	if w.read("trash/a.txt") != "alpha" || w.read("trash/a copy.txt") != "newer" {
		t.Fatal("the first item in the trash must not be overwritten")
	}
}

func TestTrashRefusesTheTrashItselfAndRoot(t *testing.T) {
	w := newWorld(t)
	w.must(os.MkdirAll(w.trash, 0o700))
	if err := w.ops.Trash([]string{w.trash}); err == nil {
		t.Fatal("the trash cannot be trashed")
	}
	if err := w.ops.Trash([]string{"/"}); err == nil {
		t.Fatal("the root cannot be trashed")
	}
}

func TestUndoReversesEachKindOfOperation(t *testing.T) {
	cases := []struct {
		name string
		do   func(w *world)
		gone string // exists after the operation, not after the undo
		back string // exists after the undo
	}{
		{"copy", func(w *world) { _, err := w.ops.Copy([]string{w.path("src/a.txt")}, w.dst); w.must(err) }, "dst/a.txt", "src/a.txt"},
		{"move", func(w *world) { _, err := w.ops.Move([]string{w.path("src/a.txt")}, w.dst); w.must(err) }, "dst/a.txt", "src/a.txt"},
		{"rename", func(w *world) { _, err := w.ops.Rename(w.path("src/a.txt"), "z.txt"); w.must(err) }, "src/z.txt", "src/a.txt"},
		{"mkdir", func(w *world) { _, err := w.ops.Mkdir(w.path("src"), "fresh"); w.must(err) }, "src/fresh", "src/a.txt"},
		{"trash", func(w *world) { w.must(w.ops.Trash([]string{w.path("src/a.txt")})) }, "trash/a.txt", "src/a.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			tc.do(w)
			if !w.exists(tc.gone) {
				t.Fatalf("%s: operation did not happen", tc.gone)
			}
			desc, err := w.ops.Undo()
			w.must(err)
			if desc == "" {
				t.Error("undo must say what it undid")
			}
			if w.exists(tc.gone) && tc.name != "trash" {
				t.Errorf("%s still exists after undo", tc.gone)
			}
			if !w.exists(tc.back) {
				t.Errorf("%s is not back", tc.back)
			}
		})
	}
}

func TestUndoOfACopyGoesToTheTrashNotToNothing(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Copy([]string{w.path("src/a.txt")}, w.dst)
	w.must(err)
	_, err = w.ops.Undo()
	w.must(err)
	if w.exists("dst/a.txt") || w.read("trash/a.txt") != "alpha" {
		t.Fatal("an undone copy is trashed: nothing is ever deleted for good")
	}
}

func TestUndoGoesBackStepByStep(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Rename(w.path("src/a.txt"), "b.txt")
	w.must(err)
	w.must(w.ops.Trash([]string{w.path("src/b.txt")}))
	_, err = w.ops.Undo()
	w.must(err)
	if !w.exists("src/b.txt") || w.exists("src/a.txt") {
		t.Fatal("first undo restores from the trash")
	}
	_, err = w.ops.Undo()
	w.must(err)
	if !w.exists("src/a.txt") {
		t.Fatal("second undo reverts the rename")
	}
	if _, err := w.ops.Undo(); err == nil {
		t.Fatal("nothing left to undo must be an error")
	}
}

func TestUndoTrashDoesNotOverwriteANewerFile(t *testing.T) {
	w := newWorld(t)
	w.must(w.ops.Trash([]string{w.path("src/a.txt")}))
	w.put("src/a.txt", "newer")
	if _, err := w.ops.Undo(); err != nil {
		t.Fatal(err)
	}
	if w.read("src/a.txt") != "newer" || w.read("src/a copy.txt") != "alpha" {
		t.Fatal("the restored file takes a free name beside the newer one")
	}
}

func TestOperationsOnMissingSourcesReportAndChangeNothing(t *testing.T) {
	w := newWorld(t)
	gone := w.path("src/gone.txt")
	if _, err := w.ops.Copy([]string{gone}, w.dst); err == nil {
		t.Error("copy of a missing file")
	}
	if _, err := w.ops.Move([]string{gone}, w.dst); err == nil {
		t.Error("move of a missing file")
	}
	if err := w.ops.Trash([]string{gone}); err == nil {
		t.Error("trash of a missing file")
	}
	if len(w.ops.undo) != 0 {
		t.Fatal("failed operations leave no undo step")
	}
}

func TestAPartialFailureKeepsWhatWasDoneUndoable(t *testing.T) {
	w := newWorld(t)
	_, err := w.ops.Copy([]string{w.path("src/a.txt"), w.path("src/gone.txt")}, w.dst)
	if err == nil {
		t.Fatal("expected the missing file to fail")
	}
	if !w.exists("dst/a.txt") {
		t.Fatal("the first copy was made")
	}
	if _, err := w.ops.Undo(); err != nil || w.exists("dst/a.txt") {
		t.Fatalf("the first copy must be undoable: %v", err)
	}
}

func TestInfoDescribesAFile(t *testing.T) {
	w := newWorld(t)
	in, err := Info(w.path("src/a.txt"))
	w.must(err)
	if in.Size != 5 || in.IsDir || in.Name != "a.txt" || in.Where != w.path("src") {
		t.Fatalf("info = %+v", in)
	}
	dir, err := Info(w.path("src/tree"))
	w.must(err)
	if !dir.IsDir || dir.Files != 2 || dir.Total != int64(len("inner")+len("deepest")) {
		t.Fatalf("folder info = %+v (files %d total %d)", dir, dir.Files, dir.Total)
	}
}
