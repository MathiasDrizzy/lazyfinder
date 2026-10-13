package search

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

func TestOnlyKeepsFilesOrFolders(t *testing.T) {
	q, _ := Parse("a", now)
	file, dir := fsmodel.Entry{Name: "a.txt"}, fsmodel.Entry{Name: "a", IsDir: true}
	if !q.Only("both").Match(file) || !q.Only("both").Match(dir) {
		t.Error("both keeps everything")
	}
	if !q.Only("files").Match(file) || q.Only("files").Match(dir) {
		t.Error("files drops folders")
	}
	if q.Only("folders").Match(file) || !q.Only("folders").Match(dir) {
		t.Error("folders drops files")
	}
}

func TestParseByContentTurnsWordsIntoContent(t *testing.T) {
	q, err := ParseBy("invoice 2026 kind:text", now, "content")
	if err != nil {
		t.Fatal(err)
	}
	if q.Content() != "invoice 2026" {
		t.Errorf("content = %q", q.Content())
	}
	if !q.Match(fsmodel.Entry{Name: "x.txt"}) { // the kind filter still applies to names
		t.Error("kind:text should match a .txt")
	}
	q2, _ := ParseBy("invoice", now, "name")
	if q2.Content() != "" || !reflect.DeepEqual(q2.Words(), []string{"invoice"}) {
		t.Errorf("by name: content %q words %v", q2.Content(), q2.Words())
	}
}

func TestIndexUsesTheSystemIndexAndFiltersItsPaths(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(root, "report.txt"), []byte("x"), 0o644))
	must(os.Mkdir(filepath.Join(root, "reports"), 0o755))
	must(os.WriteFile(filepath.Join(root, ".report-hidden"), []byte("x"), 0o644))
	var asked struct {
		text    string
		content bool
	}
	idx := func(_ context.Context, text string, content bool) ([]string, error) {
		asked.text, asked.content = text, content
		return []string{filepath.Join(root, "report.txt"), filepath.Join(root, "reports"), filepath.Join(root, ".report-hidden"), filepath.Join(root, "gone")}, nil
	}
	q, _ := Parse("report", now)
	var got []string
	err := Index(context.Background(), idx, q.Only("files"), Options{}, func(b []fsmodel.Entry) {
		for _, e := range b {
			got = append(got, e.Name)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if asked.text != "report" || asked.content {
		t.Errorf("asked %+v", asked)
	}
	if !reflect.DeepEqual(got, []string{"report.txt"}) {
		t.Fatalf("got %v: only the visible file that still exists", got)
	}
}

func TestIndexStopsAtTheLimitAndReportsAMissingIndex(t *testing.T) {
	root := t.TempDir()
	var paths []string
	for _, n := range []string{"a1", "a2", "a3"} {
		p := filepath.Join(root, n)
		_ = os.WriteFile(p, nil, 0o644)
		paths = append(paths, p)
	}
	q, _ := Parse("a", now)
	err := Index(context.Background(), func(context.Context, string, bool) ([]string, error) { return paths, nil }, q, Options{Max: 2}, func([]fsmodel.Entry) {})
	if !errors.Is(err, ErrTruncated) {
		t.Errorf("err = %v", err)
	}
	if err := Index(context.Background(), nil, q, Options{}, func([]fsmodel.Entry) {}); !errors.Is(err, ErrNoIndex) {
		t.Errorf("nil indexer: %v", err)
	}
}

func TestSpotlightQueryEscapesAndCombinesWords(t *testing.T) {
	if got := spotlightQuery(`my "doc" a\b`, false); got != `kMDItemDisplayName == "*my*"cd && kMDItemDisplayName == "*\"doc\"*"cd && kMDItemDisplayName == "*a\\b*"cd` {
		t.Errorf("name query = %s", got)
	}
	if got := spotlightQuery("invoice 2026", true); got != `kMDItemTextContent == "*invoice 2026*"cd` {
		t.Errorf("content query = %s", got)
	}
}
