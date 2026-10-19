package fileops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		if fi.Mode()&fs.ModeSymlink != 0 {
			firstErr = fmt.Errorf("%q is a link: change the item it points to instead", filepath.Base(p))
			break
		}
		old := fi.Mode() & (fs.ModePerm | specialBits)
		if err := os.Chmod(p, old&specialBits|perm); err != nil {
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
				errs = append(errs, os.Chmod(c.path, c.old))
			}
			return errors.Join(errs...)
		}, redo: func(context.Context, Progress) error { return o.Chmod(names, perm) }})
	}
	return firstErr
}
