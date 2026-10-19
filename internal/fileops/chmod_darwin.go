package fileops

import (
	"io/fs"
	"os"
)

// chmodFallback sets the mode of path where fchmodat cannot refuse a link, which macOS always can: it checks the name
// right before the change.
func chmodFallback(path string, mode fs.FileMode) error {
	if isLink(path) {
		return linkError(path)
	}
	beforeChange(path)
	return os.Chmod(path, mode)
}
