// Package git reads the working tree status by asking git itself (docs/adr/0003-live-watcher.md): it never parses
// .git internals and never reimplements .gitignore.
package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Code is the one-letter state shown next to a file, the letters of the VS Code explorer: M modified, U untracked,
// A added, D deleted, R renamed, ! in conflict and I ignored.
type Code string

// States, from the weakest to the strongest.
const (
	None       Code = ""
	Ignored    Code = "I"
	Untracked  Code = "U"
	Added      Code = "A"
	Renamed    Code = "R"
	Deleted    Code = "D"
	Modified   Code = "M"
	Conflicted Code = "!"
)

var rank = map[Code]int{None: 0, Ignored: 1, Untracked: 2, Added: 3, Renamed: 4, Deleted: 5, Modified: 6, Conflicted: 7}

const timeout = 5 * time.Second

// Status is the state of one repository at one moment.
type Status struct {
	Root            string // absolute, symlinks resolved
	Branch          string // "HEAD" when detached
	Ahead, Behind   int
	files           map[string]Code // slash path relative to Root
	viaDir, realDir string
}

// Changed is how many paths differ from the last commit; ignored ones do not count.
func (s Status) Changed() int {
	n := 0
	for _, c := range s.files {
		if c != Ignored && c != None {
			n++
		}
	}
	return n
}

// Read returns the status of the repository that contains dir. ok is false, with no error, outside a repository.
func Read(dir string) (Status, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	top, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return Status{}, false, nil
		}
		return Status{}, false, err
	}
	root := strings.TrimSpace(string(top))
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	out, err := git(ctx, root, "--no-optional-locks", "status", "--porcelain=v2", "--branch", "--ignored=matching", "-z")
	if err != nil {
		return Status{}, false, err
	}
	s := parse(out)
	s.Root = root
	if real, err := filepath.EvalSymlinks(dir); err == nil && real != dir {
		s.viaDir, s.realDir = dir, real
	}
	return s, true, nil
}

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0")
	return cmd.Output()
}

func parse(out []byte) Status {
	s := Status{files: map[string]Code{}}
	recs := bytes.Split(out, []byte{0})
	for i := 0; i < len(recs); i++ {
		r := string(recs[i])
		switch {
		case strings.HasPrefix(r, "# branch.head "):
			s.Branch = strings.TrimPrefix(r, "# branch.head ")
			if s.Branch == "(detached)" {
				s.Branch = "HEAD"
			}
		case strings.HasPrefix(r, "# branch.ab "):
			f := strings.Fields(strings.TrimPrefix(r, "# branch.ab "))
			if len(f) == 2 {
				s.Ahead, _ = strconv.Atoi(strings.TrimPrefix(f[0], "+"))
				s.Behind, _ = strconv.Atoi(strings.TrimPrefix(f[1], "-"))
			}
		case strings.HasPrefix(r, "1 "):
			s.set(pathAfter(r, 8), classify(r[2:4]))
		case strings.HasPrefix(r, "2 "):
			s.set(pathAfter(r, 9), Renamed)
			i++ // the original path follows as its own record
		case strings.HasPrefix(r, "u "):
			s.set(pathAfter(r, 10), Conflicted)
		case strings.HasPrefix(r, "? "):
			s.set(r[2:], Untracked)
		case strings.HasPrefix(r, "! "):
			s.set(r[2:], Ignored)
		}
	}
	return s
}

// pathAfter returns the path that follows n space-separated fields.
func pathAfter(rec string, n int) string {
	parts := strings.SplitN(rec, " ", n+1)
	if len(parts) <= n {
		return ""
	}
	return parts[n]
}

// classify turns the XY pair of an ordinary entry into one code: the work tree wins over the index.
func classify(xy string) Code {
	x, y := xy[0], xy[1]
	switch {
	case y == 'M' || x == 'M' || y == 'T' || x == 'T':
		return Modified
	case y == 'D' || x == 'D':
		return Deleted
	case x == 'A':
		return Added
	case x == 'R' || x == 'C':
		return Renamed
	}
	return Modified
}

func (s *Status) set(path string, c Code) {
	path = strings.TrimSuffix(path, "/")
	if path != "" {
		s.files[path] = c
	}
}

// Of returns the state of the file or directory at the absolute path. A directory takes the strongest state of
// anything inside it, except Ignored, which only applies to the directory itself.
func (s Status) Of(abs string, isDir bool) Code {
	if s.viaDir != "" && (abs == s.viaDir || strings.HasPrefix(abs, s.viaDir+string(filepath.Separator))) {
		abs = s.realDir + strings.TrimPrefix(abs, s.viaDir)
	}
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return None
	}
	rel = filepath.ToSlash(rel)
	if c, ok := s.files[rel]; ok {
		return c
	}
	for p := filepath.ToSlash(filepath.Dir(rel)); p != "." && p != "/"; p = filepath.ToSlash(filepath.Dir(p)) {
		if s.files[p] == Ignored { // inside an ignored directory
			return Ignored
		}
	}
	if !isDir {
		return None
	}
	best := None
	prefix := rel + "/"
	for p, c := range s.files {
		if strings.HasPrefix(p, prefix) && c != Ignored && rank[c] > rank[best] {
			best = c
		}
	}
	return best
}
