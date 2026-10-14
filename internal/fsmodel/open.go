package fsmodel

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// OpenRegular opens path for reading only when it is a regular file, following links. It opens without blocking, so
// a FIFO does not wait for a writer, and checks the type of what it opened, not of what a Stat saw a moment before:
// a FIFO, /dev/tty or /dev/zero in a folder (or a link to one) must never stall a preview, which reads on the UI's
// path. The caller closes the file.
func OpenRegular(path string) (*os.File, os.FileInfo, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, nil, fmt.Errorf("%q is not a regular file", filepath.Base(path))
	}
	return f, info, nil
}
