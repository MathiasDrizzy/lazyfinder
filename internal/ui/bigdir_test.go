package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A folder of 20000 files must stay usable: moving down a row and drawing the frame take milliseconds (250 at most,
// with -race), not seconds.
func TestABigFolderStaysQuick(t *testing.T) {
	e := newEnv(t)
	big := filepath.Join(e.root, "big")
	if err := os.Mkdir(big, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 20000 {
		if err := os.WriteFile(filepath.Join(big, fmt.Sprintf("file-%05d.txt", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e.key("r")
	e.pick("big")
	e.key("enter")
	start := time.Now()
	for range 50 {
		e.key("j")
		_ = e.m.View()
	}
	if per := time.Since(start) / 50; per > 250*time.Millisecond {
		t.Fatalf("a row down and a frame take %v in a folder of 20000 files", per)
	} else {
		t.Logf("%v a step", per)
	}
}
