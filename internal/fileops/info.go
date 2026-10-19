package fileops

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Details is what the Get Info popup shows about one item.
type Details struct {
	Name, Where string
	IsDir       bool
	Link        string // where a symbolic link points, if it is one
	Size        int64  // the item's own size
	Files       int64  // files below a folder, from Count
	Total       int64  // bytes of those files, from Count
	Mode        fs.FileMode
	Modified    time.Time
}

// Info reads the details of path; Count adds up what a folder holds.
func Info(path string) (Details, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return Details{}, err
	}
	d := Details{Name: filepath.Base(path), Where: filepath.Dir(path), Size: fi.Size(), Mode: fi.Mode(), Modified: fi.ModTime()}
	if fi.Mode()&fs.ModeSymlink != 0 {
		d.Link, _ = os.Readlink(path)
		if target, err := os.Stat(path); err == nil {
			d.IsDir = target.IsDir()
		}
		return d, nil
	}
	d.IsDir = fi.IsDir()
	return d, nil
}

// Count walks the folder path and adds up the files below it and their bytes, links not followed. A walk of / or a
// home folder takes a while, so it runs apart from the UI and stops with ctx.Err() once ctx is cancelled.
func Count(ctx context.Context, path string) (files, total int64, err error) {
	err = filepath.WalkDir(path, func(p string, de fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || de.IsDir() || de.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if info, err := de.Info(); err == nil {
			files++
			total += info.Size()
		}
		return nil
	})
	return files, total, err
}
