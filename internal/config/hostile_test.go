package config

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// LAZYFINDER_CONFIG may name anything. A FIFO blocked the start for good and a device such as /dev/zero was read
// until memory ran out: only a regular file of a sane size is read.
func TestTheConfigMustBeARegularFileOfASaneSize(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "fifo.json")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "big.json")
	if err := os.WriteFile(big, []byte(`{"version":1,"x":"`+strings.Repeat("a", 2<<20)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fifo, big, "/dev/zero"} {
		done := make(chan error, 1)
		go func() { _, err := Load(p, dir); done <- err }()
		select {
		case err := <-done:
			if err == nil {
				t.Errorf("Load(%s) worked", filepath.Base(p))
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Load(%s) is still reading", filepath.Base(p))
		}
	}
}
