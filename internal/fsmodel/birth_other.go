//go:build !darwin

package fsmodel

import (
	"os"
	"time"
)

// birthTime falls back to the modification time where creation time is not portable.
func birthTime(fi os.FileInfo) time.Time { return fi.ModTime() }
