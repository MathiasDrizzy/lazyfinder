package fsmodel

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// within fails the test when f does not return in time: a blocked open or read never returns at all.
func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s is still blocked after 3 s", what)
	}
}

func TestOpenRegularRefusesFifosAndDevicesWithoutBlocking(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.txt")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skip("no FIFOs here:", err)
	}
	for _, link := range []struct{ name, target string }{{"tty.txt", "/dev/tty"}, {"zero.txt", "/dev/zero"}} {
		if err := os.Symlink(link.target, filepath.Join(dir, link.name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"pipe.txt", "tty.txt", "zero.txt"} {
		within(t, "OpenRegular("+name+")", func() {
			if f, _, err := OpenRegular(filepath.Join(dir, name)); err == nil {
				_ = f.Close()
				t.Errorf("OpenRegular(%s) opened it; want an error", name)
			}
		})
	}
}

func TestOpenRegularOpensAFileThroughALink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	f, info, err := OpenRegular(filepath.Join(dir, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil || string(buf[:n]) != "hello" || info.Size() != 5 {
		t.Fatalf("read %q, %v, size %d", buf[:n], err, info.Size())
	}
}
