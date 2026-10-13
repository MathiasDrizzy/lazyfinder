//go:build !darwin && !linux

package fsmodel

import (
	"os"
	"time"
)

// birthTime falls back to the modification time where creation time is not portable.
func birthTime(fi os.FileInfo) time.Time { return fi.ModTime() }

// accessTime falls back to the modification time where the access time is not portable.
func accessTime(fi os.FileInfo) time.Time { return fi.ModTime() }
