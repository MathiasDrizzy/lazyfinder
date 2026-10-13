package search

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxContentFile = 2 << 20 // larger files are not read by the native scan
	sniffBytes     = 512
	rgTimeout      = 60 * time.Second
)

// Finder lists the files under root that contain text, using an external tool. hidden says whether hidden files
// are searched too.
type Finder func(ctx context.Context, root, text string, hidden bool) ([]string, error)

// ContentOptions controls ContentPaths.
type ContentOptions struct {
	Hidden bool
	Find   Finder // nil, or one that fails, means the native scan is used
}

// ContentPaths returns the files under root that contain text, ignoring case. It asks the external Finder first
// and falls back to a native scan of text files, which is slower but needs no tool.
func ContentPaths(ctx context.Context, root, text string, o ContentOptions) (map[string]bool, error) {
	if o.Find != nil {
		if paths, err := o.Find(ctx, root, text, o.Hidden); err == nil {
			set := make(map[string]bool, len(paths))
			for _, p := range paths {
				set[p] = true
			}
			return set, nil
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return nativeContent(ctx, root, text, o.Hidden)
}

func nativeContent(ctx context.Context, root, text string, hidden bool) (map[string]bool, error) {
	want := []byte(strings.ToLower(text))
	set := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if err != nil {
			return nil // an unreadable entry does not stop the scan
		}
		if path != root && !hidden && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if FileContains(path, want) {
			set[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return set, nil
}

// FileContains reports whether the text file at path holds want (already lower case). Binary files, files over
// 2 MiB and unreadable files do not.
func FileContains(path string, want []byte) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	if st, err := f.Stat(); err != nil || st.Size() > maxContentFile {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(f, maxContentFile))
	if err != nil {
		return false
	}
	if bytes.IndexByte(body[:min(len(body), sniffBytes)], 0) >= 0 {
		return false
	}
	return bytes.Contains(bytes.ToLower(body), want)
}

// ErrNoRg says ripgrep is not installed.
var ErrNoRg = errors.New("ripgrep (rg) is not installed")

// SystemRg is a Finder that runs ripgrep with fixed arguments and no shell. A path from rg is used only as data.
func SystemRg(ctx context.Context, root, text string, hidden bool) ([]string, error) {
	rg, err := exec.LookPath("rg")
	if err != nil {
		return nil, ErrNoRg
	}
	ctx, cancel := context.WithTimeout(ctx, rgTimeout)
	defer cancel()
	args := []string{"--files-with-matches", "--null", "--fixed-strings", "--ignore-case", "--max-filesize", "2M", "--no-messages"}
	if hidden {
		args = append(args, "--hidden")
	}
	args = append(args, "--", text, root)
	out, err := exec.CommandContext(ctx, rg, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 { // rg exits 1 when nothing matched
			return nil, nil
		}
		return nil, err
	}
	return ParseRg(string(out)), nil
}

// ParseRg splits the NUL separated output of rg --null.
func ParseRg(out string) []string {
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
