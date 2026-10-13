package search

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

var now = time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)

func file(name string, size int64, modified time.Time) fsmodel.Entry {
	return fsmodel.Entry{Name: name, Size: size, Modified: modified, Created: modified, Kind: fsmodel.KindOf(name, false, false)}
}

func dir(name string) fsmodel.Entry {
	return fsmodel.Entry{Name: name, IsDir: true, Modified: now, Created: now, Kind: "Folder"}
}

func mustParse(t *testing.T, s string) Query {
	t.Helper()
	q, err := Parse(s, now)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return q
}

func TestEmptyQueryMatchesEverything(t *testing.T) {
	q := mustParse(t, "   ")
	if !q.Empty() || !q.Match(file("a.txt", 1, now)) {
		t.Fatal("an empty query is empty and matches everything")
	}
}

func TestWordsAreCaseInsensitiveSubstringsAndAllMustMatch(t *testing.T) {
	q := mustParse(t, "READ me")
	for name, want := range map[string]bool{"readme.md": true, "ReadMe.TXT": true, "read.md": false, "me.md": false} {
		if got := q.Match(file(name, 1, now)); got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

func TestGlobWordsUseStarAndQuestionMark(t *testing.T) {
	q := mustParse(t, "*.go")
	if !q.Match(file("main.go", 1, now)) || q.Match(file("main.gox", 1, now)) || q.Match(file("go", 1, now)) {
		t.Fatal("*.go must match names ending in .go only")
	}
	q = mustParse(t, "a?c")
	if !q.Match(file("abc", 1, now)) || q.Match(file("abbc", 1, now)) {
		t.Fatal("? matches exactly one character")
	}
}

func TestQuotedWordsKeepTheirSpaces(t *testing.T) {
	q := mustParse(t, `"my notes"`)
	if !q.Match(file("my notes.txt", 1, now)) || q.Match(file("my.txt", 1, now)) {
		t.Fatal("a quoted phrase is one word")
	}
}

func TestKindFilters(t *testing.T) {
	cases := []struct {
		query string
		e     fsmodel.Entry
		want  bool
	}{
		{"kind:image", file("p.PNG", 1, now), true},
		{"kind:image", file("p.txt", 1, now), false},
		{"kind:folder", dir("src"), true},
		{"kind:folder", file("src", 1, now), false},
		{"kind:text", file("a.md", 1, now), true},
		{"kind:audio", file("a.mp3", 1, now), true},
		{"kind:video", file("a.mov", 1, now), true},
		{"kind:archive", file("a.zip", 1, now), true},
		{"kind:go", file("a.go", 1, now), true},
		{"ext:go", file("a.go", 1, now), true},
		{"ext:.go", file("a.go", 1, now), true},
		{"ext:go", file("a.rs", 1, now), false},
		{"kind:image kind:text", file("a.png", 1, now), false},
	}
	for _, c := range cases {
		if got := mustParse(t, c.query).Match(c.e); got != c.want {
			t.Errorf("%q on %s = %v, want %v", c.query, c.e.Name, got, c.want)
		}
	}
}

func TestSizeFilters(t *testing.T) {
	cases := []struct {
		query string
		size  int64
		want  bool
	}{
		{"size:>10MB", 11 << 20, true},
		{"size:>10MB", 10 << 20, false},
		{"size:>=10MB", 10 << 20, true},
		{"size:<1KB", 1023, true},
		{"size:<1KB", 1024, false},
		{"size:5", 5, true},
		{"size:5", 6, false},
		{"size:>1.5KB", 1537, true},
		{"size:>1gb", 2 << 30, true},
	}
	for _, c := range cases {
		if got := mustParse(t, c.query).Match(file("f", c.size, now)); got != c.want {
			t.Errorf("%q on %d = %v, want %v", c.query, c.size, got, c.want)
		}
	}
}

func TestSizeNeverMatchesAFolder(t *testing.T) {
	if mustParse(t, "size:>0").Match(dir("d")) {
		t.Fatal("folders have no size to compare")
	}
}

func TestDateFilters(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 9, 0, 0, 0, time.UTC) }
	cases := []struct {
		query string
		when  time.Time
		want  bool
	}{
		{"modified:today", day(13), true},
		{"modified:today", day(12), false},
		{"modified:yesterday", day(12), true},
		{"modified:week", day(8), true},
		{"modified:week", day(5), false},
		{"modified:month", time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), true},
		{"modified:year", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), true},
		{"modified:year", time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), false},
		{"modified:>2026-10-10", day(11), true},
		{"modified:>2026-10-10", day(10), false},
		{"modified:<2026-10-10", day(9), true},
		{"modified:2026-10-10", day(10), true},
		{"modified:2026-10-10", day(11), false},
	}
	for _, c := range cases {
		if got := mustParse(t, c.query).Match(file("f", 1, c.when)); got != c.want {
			t.Errorf("%q on %s = %v, want %v", c.query, c.when.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestCreatedFilterUsesTheCreationDate(t *testing.T) {
	e := file("f", 1, now)
	e.Created = now.AddDate(-1, 0, 0)
	if mustParse(t, "created:today").Match(e) || !mustParse(t, "modified:today").Match(e) {
		t.Fatal("created and modified are independent")
	}
}

func TestBadQueriesExplainThemselves(t *testing.T) {
	for _, q := range []string{"size:big", "modified:someday", "modified:>2026-13-45", "kind:", "color:red", "size:>"} {
		if _, err := Parse(q, now); err == nil {
			t.Errorf("Parse(%q) must fail", q)
		}
	}
}

func TestFiltersAndWordsCombine(t *testing.T) {
	q := mustParse(t, "report kind:text size:<1MB modified:year")
	if !q.Match(file("report.md", 100, now)) || q.Match(file("report.png", 100, now)) || q.Match(file("report.md", 2<<20, now)) {
		t.Fatal("every term must hold")
	}
}

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range []string{"a.txt", "sub/b.txt", "sub/deep/c.txt", "sub/deep/d.go", ".hidden/e.txt", "node/f.txt"} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func names(rs []fsmodel.Entry) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Name)
	}
	sort.Strings(out)
	return out
}

func collect(t *testing.T, root string, q Query, o Options) ([]fsmodel.Entry, error) {
	t.Helper()
	var got []fsmodel.Entry
	err := Walk(context.Background(), root, q, o, func(batch []fsmodel.Entry) { got = append(got, batch...) })
	return got, err
}

func TestNonRecursiveSearchLooksOnlyInTheFolder(t *testing.T) {
	root := tree(t)
	got, err := collect(t, root, mustParse(t, "txt"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if n := names(got); len(n) != 1 || n[0] != "a.txt" {
		t.Fatalf("got %v, want [a.txt]", n)
	}
}

func TestRecursiveSearchFindsDeepFilesAndSkipsHiddenByDefault(t *testing.T) {
	root := tree(t)
	got, err := collect(t, root, mustParse(t, "txt"), Options{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.txt", "b.txt", "c.txt", "f.txt"}
	if n := names(got); !equal(n, want) {
		t.Fatalf("got %v, want %v", n, want)
	}
	got, _ = collect(t, root, mustParse(t, "txt"), Options{Recursive: true, ShowHidden: true})
	if len(got) != 5 {
		t.Fatalf("with hidden shown got %d, want 5", len(got))
	}
}

func TestSkipFuncPrunesFolders(t *testing.T) {
	root := tree(t)
	skip := func(e fsmodel.Entry) bool { return e.Name == "node" }
	got, _ := collect(t, root, mustParse(t, "txt"), Options{Recursive: true, Skip: skip})
	for _, e := range got {
		if e.Name == "f.txt" {
			t.Fatal("a skipped folder must not be entered")
		}
	}
}

func TestMaxResultsStopsTheWalk(t *testing.T) {
	root := tree(t)
	got, err := collect(t, root, mustParse(t, "txt"), Options{Recursive: true, Max: 2})
	if err != ErrTruncated {
		t.Fatalf("err = %v, want ErrTruncated", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
}

func TestCancelStopsTheWalk(t *testing.T) {
	root := tree(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Walk(ctx, root, mustParse(t, "txt"), Options{Recursive: true}, func([]fsmodel.Entry) {})
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestSymlinkedFoldersAreNotFollowed(t *testing.T) {
	root := tree(t)
	if err := os.Symlink(root, filepath.Join(root, "sub", "loop")); err != nil {
		t.Fatal(err)
	}
	got, err := collect(t, root, mustParse(t, "a.txt"), Options{Recursive: true})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v err %v, want one a.txt and no loop", names(got), err)
	}
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
