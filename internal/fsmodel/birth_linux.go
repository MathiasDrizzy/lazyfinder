package fsmodel

import (
	"os"
	"syscall"
	"time"
)

// birthTime falls back to the modification time: Linux file systems do not report the creation time through stat.
func birthTime(fi os.FileInfo) time.Time { return fi.ModTime() }

// accessTime returns the last access time.
func accessTime(fi os.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
	}
	return fi.ModTime()
}
