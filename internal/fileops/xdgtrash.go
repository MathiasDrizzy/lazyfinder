package fileops

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// xdgInfo is the info folder of an XDG trash (a folder Trash/files, as ~/.local/share/Trash/files), where each item
// needs a .trashinfo for the desktop to show it and put it back; "" for any other trash folder.
func xdgInfo(trash string) string {
	if filepath.Base(trash) != "files" || filepath.Base(filepath.Dir(trash)) != "Trash" {
		return ""
	}
	return filepath.Join(filepath.Dir(trash), "info")
}

// reserveTrashName picks the name p takes in the trash. In an XDG trash it also writes the .trashinfo of that name,
// created exclusively the way the specification asks, so the name is taken in files and in info together; info is
// that file's path ("" outside an XDG trash).
func (o *Ops) reserveTrashName(p string) (name, info string, err error) {
	infoDir := xdgInfo(o.trashDir)
	if infoDir == "" {
		return UniqueName(o.trashDir, filepath.Base(p)), "", nil
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return "", "", err
	}
	body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", (&url.URL{Path: p}).EscapedPath(), time.Now().Format("2006-01-02T15:04:05"))
	for n := 0; ; n++ {
		name = candidate(filepath.Base(p), n)
		if exists(filepath.Join(o.trashDir, name)) {
			continue
		}
		info = filepath.Join(infoDir, name+".trashinfo")
		f, err := os.OpenFile(info, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		_, werr := f.WriteString(body)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			_ = os.Remove(info)
			return "", "", werr
		}
		return name, info, nil
	}
}

// dropInfo removes the .trashinfo of an item that left the trash; one already gone is fine.
func dropInfo(info string) error {
	if info == "" {
		return nil
	}
	if err := os.Remove(info); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
