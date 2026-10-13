package search

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

// ErrNoIndex means there is no system index to ask (not macOS, or Spotlight is off).
var ErrNoIndex = errors.New("no system index available")

// Indexer asks the system index for the paths whose name (or, with content, text) matches. It never opens a
// file: the index was built by the system, not by this program.
type Indexer func(ctx context.Context, text string, content bool) ([]string, error)

// Spotlight is the Indexer of macOS, backed by mdfind.
func Spotlight(ctx context.Context, text string, content bool) ([]string, error) {
	bin, err := exec.LookPath("mdfind")
	if err != nil {
		return nil, ErrNoIndex
	}
	out, err := exec.CommandContext(ctx, bin, spotlightQuery(text, content)).Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrNoIndex
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}

func spotlightQuery(text string, content bool) string {
	quote := func(s string) string {
		return `"*` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `*"cd`
	}
	if content {
		return "kMDItemTextContent == " + quote(text)
	}
	var terms []string
	for _, w := range strings.Fields(text) {
		terms = append(terms, "kMDItemDisplayName == "+quote(w))
	}
	return strings.Join(terms, " && ")
}

// Index lists what the system index holds for the query's text, checked again against the query itself. Results
// that no longer exist or sit in a hidden folder (unless ShowHidden) are dropped; Options.Max gives ErrTruncated.
func Index(ctx context.Context, idx Indexer, q Query, o Options, emit func([]fsmodel.Entry)) error {
	if idx == nil {
		return ErrNoIndex
	}
	text, content := strings.Join(q.words, " "), false
	if q.content != "" {
		text, content = q.content, true
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	paths, err := idx(ctx, text, content)
	if err != nil {
		return err
	}
	var batch []fsmodel.Entry
	found := 0
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		if p == "" || (!o.ShowHidden && hiddenPath(p)) {
			continue
		}
		e, err := fsmodel.Stat(p)
		if err != nil {
			continue
		}
		if o.Tags != nil {
			e.Tags = o.Tags(e.Path)
		}
		if !q.Match(e) {
			continue
		}
		batch = append(batch, e)
		found++
		if len(batch) >= batchSize {
			emit(batch)
			batch = nil
		}
		if o.Max > 0 && found >= o.Max {
			emit(batch)
			return ErrTruncated
		}
	}
	if len(batch) > 0 {
		emit(batch)
	}
	return nil
}

func hiddenPath(p string) bool {
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}
