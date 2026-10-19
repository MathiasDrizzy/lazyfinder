package fileops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// specialBits are the setuid, setgid and sticky bits: Chmod keeps them as they were and never sets them.
const specialBits = fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky

// ParseMode reads permissions written in octal ("644", "0755"): three digits from 0 to 7, with an optional leading 0.
// The setuid, setgid and sticky digit is not taken.
func ParseMode(s string) (fs.FileMode, error) {
	s = strings.TrimSpace(s)
	if len(s) == 4 && s[0] == '0' {
		s = s[1:]
	}
	if len(s) != 3 {
		return 0, fmt.Errorf("%q is not a mode like 644 or 755", s)
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not a mode like 644 or 755", s)
	}
	return fs.FileMode(n), nil
}

// Chmod sets the permission bits of each path to perm, not those of what a folder holds, and records an undo that
// puts the old ones back. A link is refused: the change would reach the item it points to. On an error the items
// changed so far keep their new mode and the undo covers them.
func (o *Ops) Chmod(paths []string, perm fs.FileMode) error {
	perm &= fs.ModePerm
	type change struct {
		path string
		old  fs.FileMode
	}
	var done []change
	var firstErr error
	for _, p := range paths {
		fi, err := os.Lstat(p)
		if err != nil {
			firstErr = err
			break
		}
		old := fi.Mode() & (fs.ModePerm | specialBits)
		if err := chmodNoFollow(p, old&specialBits|perm); err != nil {
			firstErr = err
			break
		}
		done = append(done, change{p, old})
	}
	if len(done) > 0 {
		names := make([]string, len(done))
		for i, c := range done {
			names[i] = c.path
		}
		o.push(step{kind: KindChmod, subject: describe(names), undo: func() error {
			var errs []error
			for _, c := range done {
				errs = append(errs, chmodNoFollow(c.path, c.old))
			}
			return errors.Join(errs...)
		}, redo: func(context.Context, Progress) error { return o.Chmod(names, perm) }})
	}
	return firstErr
}

// fchmodat is unix.Fchmodat; a test swaps it to take the path of a system without the flag.
var fchmodat = unix.Fchmodat

// chmodNoFollow sets the mode of path itself and refuses a link, whose change would reach the item it points to.
// A link swapped in after the check is not followed either: fchmodat with AT_SYMLINK_NOFOLLOW changes only the link
// on macOS and fails on Linux. Where the system lacks it (Linux before fchmodat2), chmodFallback does the same.
func chmodNoFollow(path string, mode fs.FileMode) error {
	if isLink(path) {
		return linkError(path)
	}
	err := fchmodat(unix.AT_FDCWD, path, unixMode(mode), unix.AT_SYMLINK_NOFOLLOW)
	// A container's seccomp filter may answer EPERM or ENOSYS for fchmodat2; a real EPERM comes back from os.Chmod.
	if err == nil || !slices.ContainsFunc([]error{unix.EOPNOTSUPP, unix.ENOTSUP, unix.EPERM, unix.ENOSYS},
		func(e error) bool { return errors.Is(err, e) }) {
		return err
	}
	return chmodFallback(path, mode)
}

// beforeChange runs between the check of a path and the change of its mode; a test swaps the path for a link there.
var beforeChange = func(string) {}

func isLink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&fs.ModeSymlink != 0
}

func linkError(path string) error {
	return fmt.Errorf("%q is a link: change the item it points to instead", filepath.Base(path))
}

// unixMode is mode in the bits chmod(2) takes.
func unixMode(mode fs.FileMode) uint32 {
	m := uint32(mode.Perm())
	if mode&fs.ModeSetuid != 0 {
		m |= unix.S_ISUID
	}
	if mode&fs.ModeSetgid != 0 {
		m |= unix.S_ISGID
	}
	if mode&fs.ModeSticky != 0 {
		m |= unix.S_ISVTX
	}
	return m
}
