package fsmodel

import (
	"os"
	"syscall"
	"time"
)

// birthTime returns the creation time reported by the file system.
func birthTime(fi os.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec)
	}
	return fi.ModTime()
}
