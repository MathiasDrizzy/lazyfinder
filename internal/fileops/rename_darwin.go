package fileops

import "golang.org/x/sys/unix"

// renameExclusive renames src to dst and fails when dst exists, in one step (renameatx_np with RENAME_EXCL).
func renameExclusive(src, dst string) error {
	return unix.RenameatxNp(unix.AT_FDCWD, src, unix.AT_FDCWD, dst, unix.RENAME_EXCL)
}
