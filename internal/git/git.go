// Package git reads the working tree status by asking git itself (docs/adr/0003-live-watcher.md): it never parses
// .git internals and never reimplements .gitignore.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	Head            string // the short hash of the commit checked out; empty before the first commit
	Upstream        string // the branch this one tracks, empty when it tracks none
	Ahead, Behind   int
	Staged          []Change        // changes in the index, in path order
	Unstaged        []Change        // changes in the work tree: modified, deleted, untracked and in conflict
	files           map[string]Code // slash path relative to Root
	viaDir, realDir string
}

// BranchLabel is the branch to show: its name, or where a detached HEAD is, as git says it.
func (s Status) BranchLabel() string {
	if s.Branch == "HEAD" && s.Head != "" {
		return "HEAD (detached at " + s.Head + ")"
	}
	return s.Branch
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
		// Not a repository, no git at all, or a folder git cannot enter (listed, but without x), so cannot read.
		if errors.As(err, &ee) || errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrPermission) {
			return Status{}, false, nil
		}
		return Status{}, false, err
	}
	root := strings.TrimSpace(string(top))
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	opts, env, err := readOnly(ctx, root)
	if err != nil {
		return Status{}, false, err
	}
	out, err := gitEnv(ctx, root, env, append(opts, "--no-optional-locks", "status", "--porcelain=v2", "--branch", "--ignored=matching", "--untracked-files=all", "--ignore-submodules=dirty", "-z")...)
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
	return gitEnv(ctx, dir, nil, args...)
}

// gitEnv is git with env added to the environment.
func gitEnv(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(append(cmd.Environ(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0"), env...)
	out, err := cmd.Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(bytes.TrimSpace(ee.Stderr)) > 0 { // say what git said, not only "exit status 128"
		return out, fmt.Errorf("%w: %s", err, firstLines(string(bytes.TrimSpace(ee.Stderr)), 1))
	}
	return out, err
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
		case strings.HasPrefix(r, "# branch.oid "):
			if oid := strings.TrimPrefix(r, "# branch.oid "); oid != "(initial)" {
				s.Head = oid[:min(7, len(oid))]
			}
		case strings.HasPrefix(r, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(r, "# branch.upstream ")
		case strings.HasPrefix(r, "# branch.ab "):
			f := strings.Fields(strings.TrimPrefix(r, "# branch.ab "))
			if len(f) == 2 {
				s.Ahead, _ = strconv.Atoi(strings.TrimPrefix(f[0], "+"))
				s.Behind, _ = strconv.Atoi(strings.TrimPrefix(f[1], "-"))
			}
		case strings.HasPrefix(r, "1 "):
			p := pathAfter(r, 8)
			s.set(p, classify(r[2:4]))
			s.split(p, "", r[2], r[3])
		case strings.HasPrefix(r, "2 "):
			p := pathAfter(r, 9)
			s.set(p, Renamed)
			orig := ""
			if i+1 < len(recs) {
				orig = string(recs[i+1])
			}
			i++ // the original path follows as its own record
			s.split(p, orig, r[2], r[3])
		case strings.HasPrefix(r, "u "):
			p := pathAfter(r, 10)
			s.set(p, Conflicted)
			s.Unstaged = append(s.Unstaged, Change{Path: p, Code: Conflicted})
		case strings.HasPrefix(r, "? "):
			s.set(r[2:], Untracked)
			s.Unstaged = append(s.Unstaged, Change{Path: strings.TrimSuffix(r[2:], "/"), Code: Untracked})
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

// rel is the slash path of abs relative to the repository root, through the folder the status was read from.
func (s Status) rel(abs string) (string, bool) {
	if s.viaDir != "" && (abs == s.viaDir || strings.HasPrefix(abs, s.viaDir+string(filepath.Separator))) {
		abs = s.realDir + strings.TrimPrefix(abs, s.viaDir)
	}
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// Of returns the state of the file or directory at the absolute path. A directory takes the strongest state of
// anything inside it, except Ignored, which only applies to the directory itself.
func (s Status) Of(abs string, isDir bool) Code {
	rel, ok := s.rel(abs)
	if !ok {
		return None
	}
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
