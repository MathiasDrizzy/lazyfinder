package fileops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

// volume attaches a small disk image under a temp folder: a second file system, so a move out of the temp folder
// crosses devices for real. Skipped where hdiutil does not exist (CI on Linux).
func volume(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("simulated volumes need hdiutil")
	}
	if _, err := exec.LookPath("hdiutil"); err != nil {
		t.Skip("hdiutil not found")
	}
	dir := t.TempDir()
	image, mount := filepath.Join(dir, "vol.dmg"), filepath.Join(dir, "mnt")
	if err := os.Mkdir(mount, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("hdiutil", "create", "-size", "8m", "-fs", "APFS", "-volname", "lfvol", image).CombinedOutput(); err != nil {
		t.Skipf("hdiutil create: %v: %s", err, out)
	}
	if out, err := exec.Command("hdiutil", "attach", image, "-mountpoint", mount, "-nobrowse", "-noverify").CombinedOutput(); err != nil {
		t.Skipf("hdiutil attach: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("hdiutil", "detach", mount, "-force").Run() })
	return mount
}

func TestMoveAcrossTwoVolumesCopiesThenRemovesAndUndoes(t *testing.T) {
	w := newWorld(t)
	vol := volume(t)
	if err := os.Rename(filepath.Join(w.root, "src", "a.txt"), filepath.Join(vol, "probe")); !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("the volume is not a second device: rename err = %v", err)
	}
	got, err := w.ops.Move([]string{filepath.Join(w.root, "src", "tree")}, vol)
	if err != nil || len(got) != 1 {
		t.Fatalf("move = %v, %v", got, err)
	}
	if b, err := os.ReadFile(filepath.Join(vol, "tree", "deep", "x.txt")); err != nil || string(b) != "deepest" {
		t.Fatalf("copy = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(w.root, "src", "tree")); !os.IsNotExist(err) {
		t.Fatalf("the source survived the move: %v", err)
	}
	if _, err := w.ops.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(w.root, "src", "tree", "in.txt")); err != nil || string(b) != "inner" {
		t.Fatalf("after undo = %q, %v", b, err)
	}
}

func TestFailedMoveAcrossVolumesLeavesTheSourceWhole(t *testing.T) {
	w := newWorld(t)
	vol := volume(t)
	blocked := filepath.Join(w.root, "src", "tree", "deep", "secret.txt")
	w.put("src/tree/deep/secret.txt", "s")
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := w.ops.Move([]string{filepath.Join(w.root, "src", "tree")}, vol); err == nil {
		t.Skip("running as a user that can read everything")
	}
	if b, err := os.ReadFile(filepath.Join(w.root, "src", "tree", "in.txt")); err != nil || string(b) != "inner" {
		t.Fatalf("source damaged: %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(vol, "tree")); !os.IsNotExist(err) {
		t.Fatalf("half copy left behind: %v", err)
	}
}
