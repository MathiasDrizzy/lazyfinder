// Package fileops changes the file system on the user's behalf: copy, move, rename, new folder and Trash, each with
// an undo. Nothing here deletes for good: what leaves a folder goes to the Trash, and an undone copy is trashed too
// (docs/adr/0010-file-operations.md).
package fileops

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// maxUndo is how many steps are remembered.
const maxUndo = 100

// step is one reversible operation.
type step struct {
	desc string
	undo func() error
}

// Ops runs the operations and remembers how to reverse them. trashDir is the folder that receives trashed items.
type Ops struct {
	trashDir string
	undo     []step
}

// New returns Ops that trash into the folder trash.
func New(trash string) *Ops { return &Ops{trashDir: trash} }

// CanUndo says whether there is something to undo.
func (o *Ops) CanUndo() bool { return len(o.undo) > 0 }

func (o *Ops) push(desc string, undo func() error) {
	o.undo = append(o.undo, step{desc: desc, undo: undo})
	if len(o.undo) > maxUndo {
		o.undo = o.undo[len(o.undo)-maxUndo:]
	}
}

// Undo reverses the last operation and returns what it undid.
func (o *Ops) Undo() (string, error) {
	if len(o.undo) == 0 {
		return "", errors.New("nothing to undo")
	}
	s := o.undo[len(o.undo)-1]
	if err := s.undo(); err != nil {
		return "", fmt.Errorf("undo %s: %w", s.desc, err)
	}
	o.undo = o.undo[:len(o.undo)-1]
	return s.desc, nil
}

// UniqueName returns name, or "name copy.ext", "name copy 2.ext"… when it is taken in dir, the way Finder names
// duplicates. A leading dot is part of the name, not an extension.
func UniqueName(dir, name string) string {
	if !exists(filepath.Join(dir, name)) {
		return name
	}
	ext := filepath.Ext(name)
	if ext == name { // a dotfile
		ext = ""
	}
	stem := strings.TrimSuffix(name, ext)
	for n := 1; ; n++ {
		suffix := " copy"
		if n > 1 {
			suffix = fmt.Sprintf(" copy %d", n)
		}
		if cand := stem + suffix + ext; !exists(filepath.Join(dir, cand)) {
			return cand
		}
	}
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// inside says whether p is dir itself or lies below it.
func inside(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func checkTarget(src, dstDir string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() && (inside(dstDir, src) || inside(resolved(dstDir), resolved(src))) {
		return fmt.Errorf("cannot put %q inside itself", filepath.Base(src))
	}
	return nil
}

// resolved is p with its links followed, so a folder reached through a link is compared as itself; p when it cannot
// be resolved.
func resolved(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// Copy copies each source into dstDir under a free name and returns the paths it made. Folders are copied whole,
// symbolic links as links. It stops at the first failure; what was copied by then can still be undone.
func (o *Ops) Copy(srcs []string, dstDir string) ([]string, error) {
	var made []string
	var firstErr error
	for _, src := range srcs {
		if err := checkTarget(src, dstDir); err != nil {
			firstErr = err
			break
		}
		dst := filepath.Join(dstDir, UniqueName(dstDir, filepath.Base(src)))
		if err := copyTree(src, dst); err != nil {
			firstErr = err
			break
		}
		made = append(made, dst)
	}
	if len(made) > 0 {
		created := slices.Clone(made)
		o.push(fmt.Sprintf("copy of %s", describe(created)), func() error {
			n, err := o.discard(created)
			created = created[n:] // a failed undo is tried again from the first copy still there
			return err
		})
	}
	return made, firstErr
}

// discard trashes paths without recording an undo step: it is how an undo gets rid of what it had made. It returns
// how many of paths, from the first, went to the trash.
func (o *Ops) discard(paths []string) (int, error) {
	done, err := o.toTrash(paths)
	return len(done), err
}

func describe(paths []string) string {
	if len(paths) == 1 {
		return fmt.Sprintf("%q", filepath.Base(paths[0]))
	}
	return fmt.Sprintf("%d items", len(paths))
}

// copyTree copies src to dst, which must not exist.
func copyTree(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case fi.IsDir():
		if err := os.Mkdir(dst, fi.Mode().Perm()|0o700); err != nil {
			return err
		}
		des, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, de := range des {
			if err := copyTree(filepath.Join(src, de.Name()), filepath.Join(dst, de.Name())); err != nil {
				return err
			}
		}
		if err := os.Chmod(dst, fi.Mode().Perm()); err != nil {
			return err
		}
		return os.Chtimes(dst, fi.ModTime(), fi.ModTime())
	case fi.Mode().IsRegular():
		return copyFile(src, dst, fi)
	}
	return fmt.Errorf("%q is not a regular file, folder or link", filepath.Base(src))
}

func copyFile(src, dst string, fi fs.FileInfo) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }() // read only: nothing to lose
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fi.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Chmod(dst, fi.Mode().Perm())
		}
		if err == nil {
			err = os.Chtimes(dst, fi.ModTime(), fi.ModTime())
		}
	}()
	_, err = io.Copy(out, in)
	return err
}

// Move moves each source into dstDir under a free name and returns the new paths. A source that already lives in
// dstDir stays where it is. Across file systems it copies first and removes the source only after the copy is whole.
func (o *Ops) Move(srcs []string, dstDir string) ([]string, error) {
	type pair struct{ from, to string }
	var done []pair
	var firstErr error
	for _, src := range srcs {
		if filepath.Dir(src) == filepath.Clean(dstDir) {
			continue
		}
		if err := checkTarget(src, dstDir); err != nil {
			firstErr = err
			break
		}
		dst := filepath.Join(dstDir, UniqueName(dstDir, filepath.Base(src)))
		if err := move(src, dst); err != nil {
			firstErr = err
			break
		}
		done = append(done, pair{src, dst})
	}
	made := make([]string, len(done))
	for i, p := range done {
		made[i] = p.to
	}
	if len(done) > 0 {
		left := len(done) // a failed undo is tried again from the item it stopped at
		o.push(fmt.Sprintf("move of %s", describe(made)), func() error {
			for ; left > 0; left-- {
				p := done[left-1]
				back := filepath.Join(filepath.Dir(p.from), UniqueName(filepath.Dir(p.from), filepath.Base(p.from)))
				if err := move(p.to, back); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return made, firstErr
}

// move renames src to dst, or copies and then removes it when they are on different file systems.
func move(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil || !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst) // the half copy; the source is still whole
		return err
	}
	return os.RemoveAll(src)
}

// badName says why name cannot be the name of one file or folder, or returns nil.
func badName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("the name is empty")
	case name == "." || name == "..":
		return fmt.Errorf("%q is not a valid name", name)
	case strings.ContainsAny(name, "/\x00"):
		return errors.New("a name cannot contain / or NUL")
	}
	return nil
}

// Rename gives path a new name in the same folder and returns the new path. An existing name is never replaced.
func (o *Ops) Rename(path, name string) (string, error) {
	if err := badName(name); err != nil {
		return "", err
	}
	if name == filepath.Base(path) {
		return path, nil
	}
	dst := filepath.Join(filepath.Dir(path), name)
	if exists(dst) {
		return "", fmt.Errorf("%q already exists", name)
	}
	if err := os.Rename(path, dst); err != nil {
		return "", err
	}
	o.push(fmt.Sprintf("rename of %q", filepath.Base(path)), func() error {
		if exists(path) {
			return fmt.Errorf("%q exists again", filepath.Base(path))
		}
		return os.Rename(dst, path)
	})
	return dst, nil
}

// Mkdir creates the folder name in dir and returns its path.
func (o *Ops) Mkdir(dir, name string) (string, error) {
	if err := badName(name); err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	if exists(p) {
		return "", fmt.Errorf("%q already exists", name)
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		return "", err
	}
	o.push(fmt.Sprintf("new folder %q", name), func() error {
		if err := os.Remove(p); err == nil || errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		_, err := o.discard([]string{p}) // it gained content since: keep it, in the trash
		return err
	})
	return p, nil
}

// Touch creates the empty file name in dir and returns its path. Undoing it removes the file, or keeps it in the
// trash when it gained content since.
func (o *Ops) Touch(dir, name string) (string, error) {
	if err := badName(name); err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	if exists(p) {
		return "", fmt.Errorf("%q already exists", name)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	o.push(fmt.Sprintf("new file %q", name), func() error {
		if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
			_, err := o.discard([]string{p})
			return err
		}
		if err := os.Remove(p); err == nil || errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		_, err := o.discard([]string{p})
		return err
	})
	return p, nil
}

// Trash moves paths to the Trash folder, where nothing is lost: a name taken there gets a copy suffix.
func (o *Ops) Trash(paths []string) error {
	moved, err := o.toTrash(paths)
	if len(moved) > 0 {
		left := len(moved) // a failed undo is tried again from the item it stopped at
		o.push(fmt.Sprintf("trashing of %s", describe(originals(moved))), func() error {
			for ; left > 0; left-- {
				p := moved[left-1]
				dir := filepath.Dir(p.from)
				back := filepath.Join(dir, UniqueName(dir, filepath.Base(p.from)))
				if err := move(p.to, back); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return err
}

type moved struct{ from, to string }

func originals(ms []moved) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.from
	}
	return out
}

func (o *Ops) toTrash(paths []string) ([]moved, error) {
	var done []moved
	if err := os.MkdirAll(o.trashDir, 0o700); err != nil {
		return nil, err
	}
	for _, p := range paths {
		if filepath.Dir(p) == p || inside(o.trashDir, p) {
			return done, fmt.Errorf("cannot trash %q", p)
		}
		if _, err := os.Lstat(p); err != nil {
			return done, err
		}
		dst := filepath.Join(o.trashDir, UniqueName(o.trashDir, filepath.Base(p)))
		if err := move(p, dst); err != nil {
			return done, err
		}
		done = append(done, moved{p, dst})
	}
	return done, nil
}
