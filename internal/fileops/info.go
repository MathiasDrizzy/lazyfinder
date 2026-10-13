package fileops

import (
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
	Files       int64  // files below a folder
	Total       int64  // bytes of those files
	Mode        fs.FileMode
	Modified    time.Time
}

// Info reads the details of path. For a folder it walks it, counting what is inside.
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
	if !d.IsDir {
		return d, nil
	}
	_ = filepath.WalkDir(path, func(p string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || de.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if info, err := de.Info(); err == nil {
			d.Files++
			d.Total += info.Size()
		}
		return nil
	})
	return d, nil
}
