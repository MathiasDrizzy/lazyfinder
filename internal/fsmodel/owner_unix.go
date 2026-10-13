//go:build unix

package fsmodel

import (
	"os"
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

var owners sync.Map // uid string -> user name

// ownerOf is the name of the user who owns the file, or the numeric id when the system has no such user.
func ownerOf(fi os.FileInfo) string {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	uid := strconv.FormatUint(uint64(st.Uid), 10)
	if name, ok := owners.Load(uid); ok {
		return name.(string)
	}
	name := uid
	if u, err := user.LookupId(uid); err == nil {
		name = u.Username
	}
	owners.Store(uid, name)
	return name
}
