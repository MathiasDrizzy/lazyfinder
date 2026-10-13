package fsmodel

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMoreColumnsExtensionPermissionsAndLastOpened(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "report.final.PDF")
	if err := os.WriteFile(p, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	opened := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(p, opened, time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	l, err := ReadDir(d, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	e := l.Entries[0]
	if got := ExtOf(e); got != "pdf" {
		t.Errorf("extension %q", got)
	}
	if got := PermsOf(e); got != "-rw-r-----" {
		t.Errorf("permissions %q", got)
	}
	if !e.Opened.Equal(opened) {
		t.Errorf("last opened %v, want %v", e.Opened, opened)
	}
	if ExtOf(Entry{Name: "src", IsDir: true}) != "" || ExtOf(Entry{Name: ".bashrc"}) != "" {
		t.Error("a folder and a dotfile have no extension")
	}
}

func TestSortByTheNewColumns(t *testing.T) {
	es := []Entry{
		{Name: "b.zip", Opened: time.Unix(10, 0), Mode: 0o600},
		{Name: "a.txt", Opened: time.Unix(20, 0), Mode: 0o644},
	}
	by := func(c Column) string { return Listing{Entries: es}.Sorted(Sort{By: c}).Entries[0].Name }
	if by(ColExt) != "a.txt" || by(ColOpened) != "b.zip" || by(ColPerms) != "b.zip" {
		t.Fatalf("ext %s opened %s perms %s", by(ColExt), by(ColOpened), by(ColPerms))
	}
}

func TestFormatDateAs(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		format, want string
		t            time.Time
	}{
		{"numeric", "2026-10-13 09:05", time.Date(2026, 10, 13, 9, 5, 0, 0, time.UTC)},
		{"finder", "Today, 09:05", time.Date(2026, 10, 13, 9, 5, 0, 0, time.UTC)},
		{"finder", "Yesterday, 23:59", time.Date(2026, 10, 12, 23, 59, 0, 0, time.UTC)},
		{"finder", "2 Mar 2026, 08:00", time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)},
		{"relative", "Today at 09:05", time.Date(2026, 10, 13, 9, 5, 0, 0, time.UTC)},
		{"numeric", "--", time.Time{}},
	}
	for _, c := range cases {
		if got := FormatDateAs(c.format, c.t, now); got != c.want {
			t.Errorf("%s: %q, want %q", c.format, got, c.want)
		}
	}
}
