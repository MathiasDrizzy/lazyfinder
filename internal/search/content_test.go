package search

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

func contentTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"a.txt":         "Hello World\n",
		"b.txt":         "nothing here\n",
		"sub/c.go":      "// hello again\n",
		"blob.bin":      "hello\x00\x01binary",
		"sub/.hid.txt":  "hello hidden\n",
		"big/large.txt": "hello " + string(make([]byte, 0)),
	}
	for n, body := range files {
		p := filepath.Join(root, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func rel(root string, set map[string]bool) []string {
	var out []string
	for p := range set {
		r, _ := filepath.Rel(root, p)
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func TestParseReadsTheInFilter(t *testing.T) {
	q := mustParse(t, `in:"hello world" kind:text`)
	if q.Content() != "hello world" || q.Empty() {
		t.Fatalf("content = %q", q.Content())
	}
	if _, err := Parse("in:", now); err == nil {
		t.Fatal("in: needs a value")
	}
	if !mustParse(t, "in:x").Match(file("anything", 1, now)) {
		t.Fatal("the content is checked apart from the other terms, so Match ignores it")
	}
}

func TestNativeContentSearchIgnoresCaseBinariesAndHiddenByDefault(t *testing.T) {
	root := contentTree(t)
	set, err := ContentPaths(context.Background(), root, "HELLO", ContentOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := rel(root, set)
	want := []string{"a.txt", "big/large.txt", "sub/c.go"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	withHidden, _ := ContentPaths(context.Background(), root, "hello", ContentOptions{Hidden: true})
	if len(withHidden) != 4 {
		t.Fatalf("with hidden files: %v", rel(root, withHidden))
	}
}

func TestContentSearchUsesTheExternalFinderWhenItWorks(t *testing.T) {
	root := contentTree(t)
	called := false
	find := func(_ context.Context, r, text string, hidden bool) ([]string, error) {
		called = true
		if r != root || text != "needle" || hidden {
			t.Errorf("args = %q %q %v", r, text, hidden)
		}
		return []string{filepath.Join(root, "b.txt")}, nil
	}
	set, err := ContentPaths(context.Background(), root, "needle", ContentOptions{Find: find})
	if err != nil || !called || len(set) != 1 || !set[filepath.Join(root, "b.txt")] {
		t.Fatalf("set = %v err = %v called = %v", set, err, called)
	}
}

func TestContentSearchFallsBackToTheNativeScanWhenTheFinderFails(t *testing.T) {
	root := contentTree(t)
	find := func(context.Context, string, string, bool) ([]string, error) { return nil, errors.New("rg not found") }
	set, err := ContentPaths(context.Background(), root, "hello", ContentOptions{Find: find})
	if err != nil || len(set) != 3 {
		t.Fatalf("set = %v err = %v", rel(root, set), err)
	}
}

func TestParseRgSplitsOnNul(t *testing.T) {
	got := ParseRg("/a/b.txt\x00/a/c d.txt\x00")
	if len(got) != 2 || got[1] != "/a/c d.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestWalkKeepsOnlyAllowedFiles(t *testing.T) {
	root := contentTree(t)
	allowed := map[string]bool{filepath.Join(root, "sub", "c.go"): true}
	var names []string
	err := Walk(context.Background(), root, mustParse(t, ""), Options{Recursive: true, Allowed: func(p string) bool { return allowed[p] }},
		func(b []fsmodel.Entry) {
			for _, e := range b {
				names = append(names, e.Name)
			}
		})
	if err != nil || len(names) != 1 || names[0] != "c.go" {
		t.Fatalf("names = %v err = %v", names, err)
	}
}

func TestCancelledContentScanStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ContentPaths(ctx, contentTree(t), "hello", ContentOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
