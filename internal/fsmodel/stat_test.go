package fsmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatDescribesOnePath(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "a.go")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Stat(f)
	if err != nil || e.Name != "a.go" || e.Path != f || e.Size != 5 || e.Kind != "Go Source" {
		t.Fatalf("%+v %v", e, err)
	}
	if e, _ := Stat(d); !e.IsDir {
		t.Error("a folder is a folder")
	}
	if _, err := Stat(filepath.Join(d, "nope")); err == nil {
		t.Error("a missing path is an error")
	}
}
