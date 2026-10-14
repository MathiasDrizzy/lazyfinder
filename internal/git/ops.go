package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Change is one path that differs from the last commit, on one side (the index or the work tree).
type Change struct {
	Path string // slash path relative to the repository root
	Orig string // for a rename, where it came from
	Code Code
}

// sideCode is the code of one side of an XY pair: X for the index, Y for the work tree.
func sideCode(b byte) Code {
	switch b {
	case 'A':
		return Added
	case 'D':
		return Deleted
	case 'R', 'C':
		return Renamed
	case '.':
		return None
	}
	return Modified
}

// split records an ordinary or renamed entry in the staged and unstaged lists.
func (s *Status) split(path, orig string, x, y byte) {
	if c := sideCode(x); c != None {
		s.Staged = append(s.Staged, Change{Path: path, Orig: orig, Code: c})
	}
	if c := sideCode(y); c != None {
		s.Unstaged = append(s.Unstaged, Change{Path: path, Code: c})
	}
}

const (
	writeTimeout   = 15 * time.Second
	networkTimeout = 90 * time.Second
	maxDiffBytes   = 256 << 10
)

// exec runs git for an operation that may change the repository. Prompts are off: a credential question would hang a
// terminal app that has no place to show it. The error carries what git said.
func do(timeout time.Duration, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		if ctx.Err() != nil {
			msg = "timed out"
		}
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(firstLines(msg, 3))
	}
	return out.String(), nil
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, " · ")
}

// cleanPath refuses anything that is not a path inside the repository.
func cleanPath(p string) (string, error) {
	c := filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	if p == "" || c == "." || c == ".." || strings.HasPrefix(c, "../") || filepath.IsAbs(p) {
		return "", fmt.Errorf("not a path inside the repository: %q", p)
	}
	return c, nil
}

// Stage adds the paths to the index.
func Stage(root string, paths ...string) error {
	args := []string{"add", "--"}
	for _, p := range paths {
		c, err := cleanPath(p)
		if err != nil {
			return err
		}
		args = append(args, c)
	}
	_, err := do(writeTimeout, root, args...)
	return err
}

// StageAll adds every change, new files included.
func StageAll(root string) error {
	_, err := do(writeTimeout, root, "add", "--all")
	return err
}

// hasHead tells whether the repository has a first commit; before it, "reset" has no HEAD to go back to.
func hasHead(root string) bool {
	_, err := do(writeTimeout, root, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

// Unstage takes the paths out of the index and leaves the files as they are.
func Unstage(root string, paths ...string) error {
	args := []string{"restore", "--staged", "--"}
	if !hasHead(root) {
		args = []string{"rm", "--cached", "-q", "--"}
	}
	for _, p := range paths {
		c, err := cleanPath(p)
		if err != nil {
			return err
		}
		args = append(args, c)
	}
	_, err := do(writeTimeout, root, args...)
	return err
}

// UnstageAll empties the index and leaves the files as they are.
func UnstageAll(root string) error {
	if !hasHead(root) {
		_, err := do(writeTimeout, root, "rm", "-r", "--cached", "-q", "--", ".")
		return err
	}
	_, err := do(writeTimeout, root, "restore", "--staged", "--", ".")
	return err
}

// Discard throws away the work tree side of one change: a tracked file goes back to what the index has and an
// untracked file is deleted. Ignored files are never touched and a conflict has to be resolved first.
func Discard(root string, c Change) error {
	p, err := cleanPath(c.Path)
	if err != nil {
		return err
	}
	switch c.Code {
	case Untracked:
		_, err = do(writeTimeout, root, "clean", "-f", "--", p)
	case Conflicted:
		return errors.New("resolve the conflict first")
	default:
		_, err = do(writeTimeout, root, "restore", "--", p)
	}
	return err
}

// DiscardAll throws away every work tree change, untracked files included; ignored files stay.
func DiscardAll(root string) error {
	if _, err := do(writeTimeout, root, "restore", "--", "."); err != nil {
		return err
	}
	_, err := do(writeTimeout, root, "clean", "-fd")
	return err
}

// Diff is the patch of one change: the index against HEAD when staged, the work tree against the index otherwise.
func Diff(root string, c Change, staged bool) (string, error) {
	p, err := cleanPath(c.Path)
	if err != nil {
		return "", err
	}
	args := []string{"diff", "--no-color", "--no-ext-diff"}
	switch {
	case staged:
		args = append(args, "--cached", "--", p)
	case c.Code == Untracked:
		args = append(args, "--no-index", "--", os.DevNull, p)
	default:
		args = append(args, "--", p)
	}
	out, err := do(writeTimeout, root, args...)
	if c.Code == Untracked && out != "" {
		err = nil // --no-index exits 1 when the files differ, which they always do here
	}
	if len(out) > maxDiffBytes {
		out = out[:maxDiffBytes] + "\n… (cut)\n"
	}
	return out, err
}

// Branch is a branch you can switch to.
type Branch struct {
	Name    string
	Current bool
	Remote  string // for a branch that exists only on a remote: where, as remote/name
}

// Branches lists the local branches, then the remote ones that have no local copy.
func Branches(root string) ([]Branch, error) {
	out, err := do(writeTimeout, root, "for-each-ref", "--format=%(HEAD)%09%(refname)%09%(refname:short)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	var local, remote []Branch
	seen := map[string]bool{}
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") { // the first column may be a space
		f := strings.Split(l, "\t")
		if len(f) != 3 {
			continue
		}
		switch {
		case strings.HasPrefix(f[1], "refs/heads/"):
			local = append(local, Branch{Name: f[2], Current: f[0] == "*"})
			seen[f[2]] = true
		case strings.HasPrefix(f[1], "refs/remotes/") && !strings.HasSuffix(f[1], "/HEAD"):
			short := f[2]
			if _, name, ok := strings.Cut(short, "/"); ok {
				remote = append(remote, Branch{Name: name, Remote: short})
			}
		}
	}
	for _, r := range remote {
		if !seen[r.Name] {
			seen[r.Name] = true
			local = append(local, r)
		}
	}
	return local, nil
}

// ValidBranch tells whether git accepts the name for a branch.
func ValidBranch(root, name string) error {
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid branch name: %q", name)
	}
	if _, err := do(writeTimeout, root, "check-ref-format", "--branch", name); err != nil {
		return fmt.Errorf("invalid branch name: %q", name)
	}
	return nil
}

// Checkout switches to a branch. It never forces: if the switch would overwrite local changes git refuses and the
// error says which files.
func Checkout(root, name string) error {
	if err := ValidBranch(root, name); err != nil {
		return err
	}
	_, err := do(writeTimeout, root, "switch", name)
	return err
}

// CreateBranch makes a branch at HEAD and switches to it, carrying the local changes along.
func CreateBranch(root, name string) error {
	if err := ValidBranch(root, name); err != nil {
		return err
	}
	_, err := do(writeTimeout, root, "switch", "-c", name)
	return err
}

// StashPush shelves every change, untracked files included.
func StashPush(root, message string) error {
	args := []string{"stash", "push", "--include-untracked"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := do(writeTimeout, root, args...)
	return err
}

// Stashes lists the stash entries, newest first, as "stash@{0}: message".
func Stashes(root string) ([]string, error) {
	out, err := do(writeTimeout, root, "stash", "list", "--format=%gd: %gs")
	if err != nil {
		return nil, err
	}
	var list []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l != "" {
			list = append(list, l)
		}
	}
	return list, nil
}

// StashPop brings back the newest stash and drops it; on a conflict git keeps the stash.
func StashPop(root string) error {
	_, err := do(writeTimeout, root, "stash", "pop")
	return err
}

// Fetch asks every remote for news and forgets branches that were deleted there.
func Fetch(root string) error {
	_, err := do(networkTimeout, root, "fetch", "--all", "--prune")
	return err
}

// Pull brings the upstream in when that is a fast-forward; anything else is left to the user (lazygit).
func Pull(root string) error {
	_, err := do(networkTimeout, root, "pull", "--ff-only")
	return err
}

// hasUpstream tells whether the current branch tracks another one.
func hasUpstream(root string) bool {
	_, err := do(writeTimeout, root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	return err == nil
}

// Push sends the current branch to its upstream. A branch without one is refused: PushSetUpstream does that on purpose.
func Push(root string) error {
	if !hasUpstream(root) {
		return errors.New("this branch has no upstream")
	}
	_, err := do(networkTimeout, root, "push")
	return err
}

// PushSetUpstream publishes the current branch on the first remote and makes it the upstream.
func PushSetUpstream(root string) error {
	out, err := do(writeTimeout, root, "remote")
	if err != nil {
		return err
	}
	remote, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	if remote == "" {
		return errors.New("this repository has no remote")
	}
	_, err = do(networkTimeout, root, "push", "--set-upstream", remote, "HEAD")
	return err
}
