// Package search parses the one-line query of the search bar (docs/adr/0009-finder-and-explorer-parity.md) and
// walks folders looking for matches. It knows nothing about the UI.
package search

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

// ErrTruncated is returned by Walk when it stopped at Options.Max results.
var ErrTruncated = errors.New("search stopped at the result limit")

const batchSize = 64

// Query is a parsed search. The zero value matches everything.
type Query struct {
	preds []func(fsmodel.Entry) bool
}

// Empty reports whether the query has no terms.
func (q Query) Empty() bool { return len(q.preds) == 0 }

// Match reports whether e satisfies every term.
func (q Query) Match(e fsmodel.Entry) bool {
	for _, p := range q.preds {
		if !p(e) {
			return false
		}
	}
	return true
}

// Parse reads a query: words match the name, key:value tokens filter. now fixes what "today" means.
func Parse(s string, now time.Time) (Query, error) {
	var q Query
	for _, tok := range tokens(s) {
		key, val, isFilter := strings.Cut(tok, ":")
		key = strings.ToLower(key)
		if !isFilter || !filterKeys[key] {
			if isFilter && looksLikeFilter(key) {
				return Query{}, fmt.Errorf("unknown filter %q (use kind, ext, size, modified, created or name)", key)
			}
			q.preds = append(q.preds, wordMatcher(tok))
			continue
		}
		if val == "" {
			return Query{}, fmt.Errorf("the %s filter needs a value, for example %s", key, example[key])
		}
		p, err := filter(key, val, now)
		if err != nil {
			return Query{}, err
		}
		q.preds = append(q.preds, p)
	}
	return q, nil
}

var filterKeys = map[string]bool{"kind": true, "ext": true, "size": true, "modified": true, "created": true, "name": true}

var example = map[string]string{
	"kind": "kind:image", "ext": "ext:go", "size": "size:>10MB",
	"modified": "modified:week", "created": "created:2026-10-01", "name": "name:report",
}

// looksLikeFilter tells "color:red" (a mistyped filter) from "a:b" in a file name that has no meaning as a filter.
func looksLikeFilter(key string) bool { return len(key) >= 3 && !strings.ContainsAny(key, `/\.*?`) }

// tokens splits on spaces; double quotes keep spaces inside one token.
func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	quoted, any := false, false
	flush := func() {
		if any {
			out = append(out, cur.String())
		}
		cur.Reset()
		any = false
	}
	for _, r := range s {
		switch {
		case r == '"':
			quoted, any = !quoted, true
		case (r == ' ' || r == '\t') && !quoted:
			flush()
		default:
			cur.WriteRune(r)
			any = true
		}
	}
	flush()
	return out
}

func wordMatcher(w string) func(fsmodel.Entry) bool {
	w = strings.ToLower(w)
	if strings.ContainsAny(w, "*?") {
		return func(e fsmodel.Entry) bool {
			ok, _ := path.Match(w, strings.ToLower(e.Name))
			return ok
		}
	}
	return func(e fsmodel.Entry) bool { return strings.Contains(strings.ToLower(e.Name), w) }
}

func filter(key, val string, now time.Time) (func(fsmodel.Entry) bool, error) {
	switch key {
	case "name":
		return wordMatcher(val), nil
	case "ext":
		want := strings.ToLower(strings.TrimPrefix(val, "."))
		return func(e fsmodel.Entry) bool { return !e.IsDir && extOf(e.Name) == want }, nil
	case "kind":
		return kindMatcher(strings.ToLower(val)), nil
	case "size":
		return sizeMatcher(val)
	case "modified":
		return dateMatcher(val, now, func(e fsmodel.Entry) time.Time { return e.Modified })
	default:
		return dateMatcher(val, now, func(e fsmodel.Entry) time.Time { return e.Created })
	}
}

func extOf(name string) string { return strings.ToLower(strings.TrimPrefix(path.Ext(name), ".")) }

var kindExts = map[string][]string{
	"image":    {"png", "jpg", "jpeg", "gif", "webp", "bmp", "tiff", "tif", "heic", "svg", "ico", "avif"},
	"text":     {"txt", "md", "markdown", "rst", "log", "csv", "tsv", "json", "yaml", "yml", "toml", "xml", "ini", "cfg", "conf"},
	"audio":    {"mp3", "wav", "flac", "aac", "m4a", "ogg", "opus", "aiff"},
	"video":    {"mp4", "mov", "mkv", "avi", "webm", "m4v", "wmv"},
	"archive":  {"zip", "tar", "gz", "tgz", "bz2", "xz", "7z", "rar", "zst", "dmg"},
	"document": {"pdf", "doc", "docx", "odt", "rtf", "pages", "xls", "xlsx", "ods", "numbers", "ppt", "pptx", "odp", "key"},
}

func kindMatcher(k string) func(fsmodel.Entry) bool {
	switch k {
	case "folder", "dir", "directory":
		return func(e fsmodel.Entry) bool { return e.IsDir }
	case "file":
		return func(e fsmodel.Entry) bool { return !e.IsDir }
	}
	exts, ok := kindExts[k]
	if !ok {
		exts = []string{strings.TrimPrefix(k, ".")} // any other value is an extension
	}
	return func(e fsmodel.Entry) bool {
		if e.IsDir {
			return false
		}
		x := extOf(e.Name)
		for _, w := range exts {
			if x == w {
				return true
			}
		}
		return false
	}
}

var sizeUnits = map[string]float64{"": 1, "b": 1, "kb": 1 << 10, "mb": 1 << 20, "gb": 1 << 30, "tb": 1 << 40}

func sizeMatcher(val string) (func(fsmodel.Entry) bool, error) {
	op, rest := splitOp(val)
	i := 0
	for i < len(rest) && (rest[i] == '.' || (rest[i] >= '0' && rest[i] <= '9')) {
		i++
	}
	n, err := strconv.ParseFloat(rest[:i], 64)
	unit, ok := sizeUnits[strings.ToLower(strings.TrimSpace(rest[i:]))]
	if err != nil || !ok {
		return nil, fmt.Errorf("size %q is not a size (examples: size:>10MB, size:<1KB)", val)
	}
	limit := int64(n * unit)
	return func(e fsmodel.Entry) bool {
		if e.IsDir {
			return false
		}
		return compare(e.Size, limit, op)
	}, nil
}

func compare(v, limit int64, op string) bool {
	switch op {
	case ">":
		return v > limit
	case ">=":
		return v >= limit
	case "<":
		return v < limit
	case "<=":
		return v <= limit
	}
	return v == limit
}

func splitOp(s string) (op, rest string) {
	for _, o := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(s, o) {
			return o, s[len(o):]
		}
	}
	return "", s
}

func dateMatcher(val string, now time.Time, get func(fsmodel.Entry) time.Time) (func(fsmodel.Entry) bool, error) {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	between := func(from, to time.Time) func(fsmodel.Entry) bool {
		return func(e fsmodel.Entry) bool { t := get(e); return !t.Before(from) && t.Before(to) }
	}
	tomorrow := today.AddDate(0, 0, 1)
	switch strings.ToLower(val) {
	case "today":
		return between(today, tomorrow), nil
	case "yesterday":
		return between(today.AddDate(0, 0, -1), today), nil
	case "week":
		return between(today.AddDate(0, 0, -6), tomorrow), nil
	case "month":
		return between(today.AddDate(0, -1, 0), tomorrow), nil
	case "year":
		return between(today.AddDate(-1, 0, 0), tomorrow), nil
	}
	op, rest := splitOp(val)
	d, err := time.ParseInLocation("2006-01-02", rest, loc)
	if err != nil {
		return nil, fmt.Errorf("date %q is not valid (use today, yesterday, week, month, year or YYYY-MM-DD, optionally with > or <)", val)
	}
	next := d.AddDate(0, 0, 1)
	switch op {
	case ">":
		return func(e fsmodel.Entry) bool { return !get(e).Before(next) }, nil
	case "<":
		return func(e fsmodel.Entry) bool { return get(e).Before(d) }, nil
	}
	return between(d, next), nil
}

// Options controls Walk.
type Options struct {
	Recursive  bool
	ShowHidden bool
	Max        int                      // stop after this many results; 0 means no limit
	Skip       func(fsmodel.Entry) bool // folders for which it returns true are not entered
}

// Walk looks for matches under root and hands them to emit in batches. It stops on ctx, on Options.Max
// (ErrTruncated) or when the walk ends. Symbolic links to folders are listed but never entered.
func Walk(ctx context.Context, root string, q Query, o Options, emit func([]fsmodel.Entry)) error {
	var batch []fsmodel.Entry
	found := 0
	flush := func() {
		if len(batch) > 0 {
			emit(batch)
			batch = nil
		}
	}
	var visit func(dir string, top bool) error
	visit = func(dir string, top bool) error {
		l, err := fsmodel.ReadDir(dir, fsmodel.ReadOptions{ShowHidden: o.ShowHidden})
		if err != nil {
			if top {
				return err
			}
			return nil // an unreadable subfolder does not stop the search
		}
		for _, e := range l.Entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if q.Match(e) {
				batch = append(batch, e)
				found++
				if len(batch) >= batchSize {
					flush()
				}
				if o.Max > 0 && found >= o.Max {
					return ErrTruncated
				}
			}
			if o.Recursive && e.IsDir && !e.IsLink && (o.Skip == nil || !o.Skip(e)) {
				if err := visit(e.Path, false); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := visit(root, true)
	flush()
	return err
}
