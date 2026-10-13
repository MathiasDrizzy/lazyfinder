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
	at := func(y int, mo time.Month, d, h, mi int) time.Time { return time.Date(y, mo, d, h, mi, 0, 0, time.UTC) }
	cases := []struct {
		format, want string
		t            time.Time
	}{
		{"numeric", "2026-10-13 09:05", at(2026, 10, 13, 9, 5)},
		{"absolute", "13 Oct 2026, 09:05", at(2026, 10, 13, 9, 5)},
		{"absolute", "12 Oct 2026, 23:59", at(2026, 10, 12, 23, 59)}, // yesterday keeps its date: one format per column
		{"absolute", "2 Mar 2026, 08:00", at(2026, 3, 2, 8, 0)},
		{"relative", "5 h ago", at(2026, 10, 13, 9, 5)},
		{"relative", "just now", at(2026, 10, 13, 15, 0)},
		{"relative", "30 min ago", at(2026, 10, 13, 14, 30)},
		{"relative", "1 d ago", at(2026, 10, 12, 14, 0)},
		{"relative", "3 mo ago", at(2026, 7, 10, 9, 0)},
		{"relative", "2 y ago", at(2024, 9, 1, 9, 0)},
		{"relative", "14 Oct 2026, 09:00", at(2026, 10, 14, 9, 0)}, // a date in the future has no "ago": it is written out
		{"numeric", "--", time.Time{}},
		{"absolute", "--", time.Time{}},
		{"relative", "--", time.Time{}},
	}
	for _, c := range cases {
		if got := FormatDateAs(c.format, c.t, now); got != c.want {
			t.Errorf("%s %v: %q, want %q", c.format, c.t, got, c.want)
		}
	}
}

// Mathias, 2026-10-13: "hay uno que dice 1979". A timestamp that is empty, from before 1980 or the epoch (a system
// folder whose birth time is 0) is a missing date, not a real one, in every format.
func TestAMissingTimestampIsShownAsDashesInEveryFormat(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	for _, bad := range []time.Time{
		{}, time.Unix(0, 0), time.Unix(0, 0).In(time.FixedZone("UTC-3", -3*3600)),
		time.Date(1979, 12, 31, 21, 0, 0, 0, time.UTC), time.Date(1969, 12, 31, 21, 0, 0, 0, time.UTC),
	} {
		for _, f := range []string{"absolute", "relative", "numeric"} {
			if got := FormatDateAs(f, bad, now); got != "--" {
				t.Errorf("%s %v: %q, want --", f, bad, got)
			}
		}
		if got := FormatDate(bad, now); got != "--" {
			t.Errorf("FormatDate %v: %q, want --", bad, got)
		}
	}
	if got := FormatDateAs("absolute", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), now); got == "--" {
		t.Error("1980 is a real date")
	}
}

// A file whose time stamps are 0 reads as the epoch from the file system; its columns must say "--".
func TestAnEpochFileShowsDashesInTheDateColumns(t *testing.T) {
	now := time.Now()
	path := filepath.Join(t.TempDir(), "old")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, time.Unix(0, 0), time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	e, err := Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatDateAs("absolute", e.Modified, now); got != "--" {
		t.Fatalf("modified: %q", got)
	}
}

// Real case seen on macOS 2026-10-13: a provisioning profile inside Discord.app has birth and modification time 0.
func TestARealSystemFileWithBirthTimeZero(t *testing.T) {
	const path = "/Applications/Discord.app/Contents/embedded.provisionprofile"
	e, err := Stat(path)
	if err != nil {
		t.Skip("not on this machine: " + path)
	}
	for _, d := range []time.Time{e.Modified, e.Created} {
		if got := FormatDateAs("absolute", d, time.Now()); d.Before(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)) && got != "--" {
			t.Fatalf("%v shown as %q", d, got)
		}
	}
}
