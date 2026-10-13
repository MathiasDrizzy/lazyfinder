package fsmodel

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOwnerIsTheUserWhoOwnsTheFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no owner on Windows")
	}
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := ReadDir(d, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	if got := l.Entries[0].Owner; got != u.Username {
		t.Fatalf("owner %q, want %q", got, u.Username)
	}
}

func TestSortByOwnerAndByPath(t *testing.T) {
	es := []Entry{
		{Name: "a", Path: "/z/a", Owner: "root"},
		{Name: "b", Path: "/a/b", Owner: "ana"},
	}
	by := func(c Column) string { return Listing{Entries: es}.Sorted(Sort{By: c}).Entries[0].Name }
	if by(ColOwner) != "b" || by(ColPath) != "b" {
		t.Fatalf("owner %s path %s", by(ColOwner), by(ColPath))
	}
}
