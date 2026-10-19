package fileops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// chmodFallback sets the mode of path where fchmodat cannot refuse a link (Linux before fchmodat2), with no gap
// between the check and the change: the path is opened once as an O_PATH handle that does not follow a link, and the
// mode is set through /proc/self/fd, which reaches the item opened whatever the name holds by then. Without /proc it
// checks the name right before the change, the best there is.
func chmodFallback(path string, mode fs.FileMode) error {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return &fs.PathError{Op: "chmod", Path: path, Err: err}
	}
	defer func() { _ = unix.Close(fd) }()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return &fs.PathError{Op: "chmod", Path: path, Err: err}
	}
	if st.Mode&unix.S_IFMT == unix.S_IFLNK {
		return linkError(path)
	}
	beforeChange(path)
	err = unix.Chmod(fmt.Sprintf("/proc/self/fd/%d", fd), unixMode(mode))
	if errors.Is(err, unix.ENOENT) && !procMounted() {
		if isLink(path) {
			return linkError(path)
		}
		return os.Chmod(path, mode)
	}
	if err != nil {
		return &fs.PathError{Op: "chmod", Path: path, Err: err}
	}
	return nil
}

func procMounted() bool {
	_, err := os.Stat("/proc/self/fd")
	return err == nil
}
