package fileops

import "golang.org/x/sys/unix"

// renameExclusive renames src to dst and fails when dst exists, in one step (renameat2 with RENAME_NOREPLACE).
func renameExclusive(src, dst string) error {
	return unix.Renameat2(unix.AT_FDCWD, src, unix.AT_FDCWD, dst, unix.RENAME_NOREPLACE)
}
