package opener

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

// Mathias, 2026-10-22: "porque no esta el abrir con, micro etc..". Open With offers the editors that ship built in and
// are installed, terminal programs on PATH and applications in /Applications, and only those.
func TestOpenWithOffersTheEditorsThatAreInstalled(t *testing.T) {
	old := AppExists
	t.Cleanup(func() { AppExists = old })
	AppExists = func(name string) bool { return name == "Zed" }
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "micro"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	var names []string
	for _, c := range New(config.Config{}).Choices("/d/a.txt") {
		names = append(names, c.Name)
	}
	if !reflect.DeepEqual(names, []string{"micro", "zed", "default"}) {
		t.Fatalf("names = %v, want the installed editors and the default", names)
	}
}
