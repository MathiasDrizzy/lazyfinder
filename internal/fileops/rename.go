package fileops

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// renameNoReplace renames src to dst but never replaces what is at dst: a file created there after the free name
// was chosen makes it fail with fs.ErrExist instead of being lost. On a file system without an exclusive rename it
// checks and renames, which leaves only a short window.
func renameNoReplace(src, dst string) error {
	err := renameExclusive(src, dst)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EINVAL) || errors.Is(err, unix.ENOSYS) {
		if exists(dst) {
			return &os.LinkError{Op: "rename", Old: src, New: dst, Err: fs.ErrExist}
		}
		return os.Rename(src, dst)
	}
	if err != nil {
		return &os.LinkError{Op: "rename", Old: src, New: dst, Err: err}
	}
	return nil
}
