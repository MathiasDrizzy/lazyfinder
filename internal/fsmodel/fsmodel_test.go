package fsmodel

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "Zero bytes"}, {1, "1 byte"}, {999, "999 bytes"}, {1000, "1 KB"}, {1500, "1.5 KB"},
		{1_000_000, "1 MB"}, {12_345_678, "12.3 MB"}, {2_000_000_000, "2 GB"},
	}
	for _, c := range cases {
		if got := HumanSize(c.in); got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKindOf(t *testing.T) {
	cases := []struct {
		name  string
		isDir bool
		link  bool
		want  string
	}{
		{"src", true, false, "Folder"},
		{"a.txt", false, false, "Plain Text"},
		{"A.TXT", false, false, "Plain Text"},
		{"r.pdf", false, false, "PDF Document"},
		{"p.jpg", false, false, "JPEG image"},
		{"p.JPEG", false, false, "JPEG image"},
		{"p.png", false, false, "PNG image"},
		{"main.go", false, false, "Go Source"},
		{"x.xlsx", false, false, "Excel Spreadsheet"},
		{"noext", false, false, "Document"},
		{"x.unknownext", false, false, "UNKNOWNEXT File"},
		{"link", false, true, "Alias"},
		{"dirlink", true, true, "Alias"},
	}
	for _, c := range cases {
		if got := KindOf(c.name, c.isDir, c.link); got != c.want {
			t.Errorf("KindOf(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 10, 13, 9, 5, 0, 0, time.UTC), "Today at 09:05"},
		{time.Date(2026, 10, 12, 23, 59, 0, 0, time.UTC), "Yesterday at 23:59"},
		{time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC), "2 Mar 2026"},
		{time.Time{}, "--"},
	}
	for _, c := range cases {
		if got := FormatDate(c.t, now); got != c.want {
			t.Errorf("FormatDate(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Mkdir(filepath.Join(d, "zdir"), 0o755))
	must(os.Mkdir(filepath.Join(d, "adir"), 0o755))
	must(os.WriteFile(filepath.Join(d, "b.txt"), []byte("hello"), 0o644))
	must(os.WriteFile(filepath.Join(d, "A.txt"), []byte("hi"), 0o644))
	must(os.WriteFile(filepath.Join(d, "file10.txt"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(d, "file2.txt"), []byte("xxxxxxxxxx"), 0o644))
	must(os.WriteFile(filepath.Join(d, ".hidden"), nil, 0o644))
	must(os.Symlink(filepath.Join(d, "nonexistent"), filepath.Join(d, "dangling")))
	return d
}

func names(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestReadDirHidesDotfilesAndSortsFoldersFirst(t *testing.T) {
	d := fixture(t)
	l, err := ReadDir(d, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"adir", "zdir", "A.txt", "b.txt", "dangling", "file2.txt", "file10.txt"}
	if got := names(l.Entries); !equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	l2, _ := ReadDir(d, ReadOptions{ShowHidden: true})
	if len(l2.Entries) != len(want)+1 {
		t.Fatalf("hidden entry not listed: %v", names(l2.Entries))
	}
}

func TestSortBySizeDescendingKeepsFoldersFirst(t *testing.T) {
	d := fixture(t)
	l, _ := ReadDir(d, ReadOptions{})
	s := l.Sorted(Sort{By: ColSize, Descending: true, FoldersFirst: true})
	got := names(s.Entries)
	if got[0] != "adir" && got[0] != "zdir" {
		t.Fatalf("folders must stay first: %v", got)
	}
	if got[2] != "file2.txt" {
		t.Fatalf("largest file first among files: %v", got)
	}
	// the receiver is immutable
	if names(l.Entries)[0] != "adir" {
		t.Fatal("Sorted mutated its receiver")
	}
}

func TestSortFoldersFirstOff(t *testing.T) {
	d := fixture(t)
	l, _ := ReadDir(d, ReadOptions{})
	s := l.Sorted(Sort{By: ColName, FoldersFirst: false})
	if got := names(s.Entries)[0]; got != "A.txt" {
		t.Fatalf("first = %q", got)
	}
}

func TestDanglingSymlinkIsAliasAndDoesNotFail(t *testing.T) {
	d := fixture(t)
	l, err := ReadDir(d, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range l.Entries {
		if e.Name == "dangling" {
			if !e.IsLink || e.Kind != "Alias" {
				t.Fatalf("dangling = %+v", e)
			}
			return
		}
	}
	t.Fatal("dangling link not listed")
}

func TestReadDirUnreadableReturnsActionableError(t *testing.T) {
	d := t.TempDir()
	sub := filepath.Join(d, "locked")
	if err := os.Mkdir(sub, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	_, err := ReadDir(sub, ReadOptions{})
	if err == nil {
		t.Fatal("want error")
	}
	if msg := err.Error(); msg == "" || !contains(msg, "locked") {
		t.Fatalf("error should name the folder: %q", msg)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestNaturalLess(t *testing.T) {
	if !naturalLess("file2", "file10") || naturalLess("file10", "file2") {
		t.Fatal("natural order broken")
	}
	if !naturalLess("a", "B") {
		t.Fatal("case-insensitive order broken")
	}
}

func TestMatchNameSubstringGlobAndCase(t *testing.T) {
	for _, tc := range []struct {
		pat, name string
		want      bool
	}{
		{"not", "Notes.TXT", true},
		{"NOTES", "notes.txt", true},
		{"xyz", "notes.txt", false},
		{"*.go", "Main.GO", true},
		{"*.go", "main.gox", false},
		{"n?tes*", "notes.txt", true},
		{"[", "a[b", true}, // a broken pattern is a plain substring
		{"", "anything", true},
	} {
		if got := MatchName(tc.pat, tc.name); got != tc.want {
			t.Errorf("MatchName(%q, %q) = %v, want %v", tc.pat, tc.name, got, tc.want)
		}
	}
}

func TestKindsAreTranslatedInEveryLanguage(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	for _, lang := range i18n.Languages {
		if lang == i18n.LangEN {
			continue
		}
		i18n.SetLanguage(string(lang))
		for ext, names := range kinds {
			if got := KindOf("f."+ext, false, false); got == names[1] && names[1] != "JavaScript" && names[1] != "Application" { // the same word in some languages
				t.Errorf("%s: the kind of .%s is still %q", lang, ext, got)
			}
		}
	}
}
