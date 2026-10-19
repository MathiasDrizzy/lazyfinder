package fileops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// copyTree copies src to dst, which must not exist. Below the two folders it starts from, every item is reached from
// the folder above it, already open, and nothing follows a link: an item swapped for a link while the copy runs (in a
// shared folder, by someone else) can neither make the copy read from where the link points nor take the mode and
// times meant for what the copy made.
func copyTree(ctx context.Context, src, dst string, onBytes Progress) error {
	from, err := openDir(unix.AT_FDCWD, filepath.Dir(src), filepath.Dir(src), false)
	if err != nil {
		return untouched{err}
	}
	defer func() { _ = from.Close() }() // read only: nothing to lose
	to, err := openDir(unix.AT_FDCWD, filepath.Dir(dst), filepath.Dir(dst), false)
	if err != nil {
		return untouched{err}
	}
	defer func() { _ = to.Close() }()
	return copyAt(ctx, item{from, src}, item{to, dst}, onBytes)
}

// untouched is an error from before the copy made dst: whatever is there is not the copy's.
type untouched struct{ error }

func (u untouched) Unwrap() error { return u.error }

// item is a name in an open folder; path, the same item from the working folder, is only for messages.
type item struct {
	dir  *os.File
	path string
}

func (i item) fd() int               { return int(i.dir.Fd()) }
func (i item) name() string          { return filepath.Base(i.path) }
func (i item) child(n string) string { return filepath.Join(i.path, n) }

// openDir opens the folder name in dir. noFollow refuses a link in its place.
func openDir(dir int, name, path string, noFollow bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC
	if noFollow {
		flags |= unix.O_NOFOLLOW
	}
	fd, err := unix.Openat(dir, name, flags, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(fd), path), nil
}

func copyAt(ctx context.Context, src, dst item, onBytes Progress) error {
	if err := ctx.Err(); err != nil {
		return untouched{err}
	}
	var st unix.Stat_t
	if err := unix.Fstatat(src.fd(), src.name(), &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return untouched{&fs.PathError{Op: "lstat", Path: src.path, Err: err}}
	}
	switch uint32(st.Mode) & unix.S_IFMT {
	case unix.S_IFLNK:
		target, err := readlinkAt(src)
		if err != nil {
			return untouched{err}
		}
		if err := unix.Symlinkat(target, dst.fd(), dst.name()); err != nil {
			return untouched{&os.LinkError{Op: "symlink", Old: target, New: dst.path, Err: err}}
		}
		return nil
	case unix.S_IFDIR:
		return copyDirAt(ctx, src, dst, onBytes)
	case unix.S_IFREG:
		return copyFileAt(ctx, src, dst, onBytes)
	}
	return untouched{fmt.Errorf("%q is not a regular file, folder or link", src.name())}
}

func readlinkAt(src item) (string, error) {
	for size := 256; ; size *= 2 {
		b := make([]byte, size)
		n, err := unix.Readlinkat(src.fd(), src.name(), b)
		if err != nil {
			return "", &fs.PathError{Op: "readlink", Path: src.path, Err: err}
		}
		if n < size {
			return string(b[:n]), nil
		}
	}
}

func copyDirAt(ctx context.Context, src, dst item, onBytes Progress) error {
	in, err := openDir(src.fd(), src.name(), src.path, true)
	if err != nil {
		return untouched{err}
	}
	defer func() { _ = in.Close() }() // read only: nothing to lose
	fi, err := in.Stat()
	if err != nil {
		return untouched{err}
	}
	if err := unix.Mkdirat(dst.fd(), dst.name(), uint32(fi.Mode().Perm()|0o700)); err != nil {
		return untouched{&fs.PathError{Op: "mkdir", Path: dst.path, Err: err}}
	}
	names, err := in.Readdirnames(-1)
	if err != nil {
		return err
	}
	out, err := openDir(dst.fd(), dst.name(), dst.path, true)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	for _, n := range names {
		if err := copyAt(ctx, item{in, src.child(n)}, item{out, dst.child(n)}, onBytes); err != nil {
			var u untouched
			if errors.As(err, &u) { // the item inside made nothing, but this folder was made
				return u.error
			}
			return err
		}
	}
	if err := copyXattrs(ctx, in, out); err != nil {
		return err
	}
	if err := out.Chmod(fi.Mode().Perm()); err != nil {
		return err
	}
	return setTimes(dst, fi)
}

func copyFileAt(ctx context.Context, src, dst item, onBytes Progress) (err error) {
	// O_NONBLOCK: a fifo swapped in after the stat would block the open; a regular file reads the same with it.
	fd, err := unix.Openat(src.fd(), src.name(), unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return untouched{&fs.PathError{Op: "open", Path: src.path, Err: err}}
	}
	in := os.NewFile(uintptr(fd), src.path)
	defer func() { _ = in.Close() }() // read only: nothing to lose
	fi, err := in.Stat()
	if err != nil {
		return untouched{err}
	}
	if !fi.Mode().IsRegular() {
		return untouched{fmt.Errorf("%q is not a regular file, folder or link", src.name())}
	}
	if err := unix.SetNonblock(fd, false); err != nil {
		return untouched{&fs.PathError{Op: "open", Path: src.path, Err: err}}
	}
	ofd, err := unix.Openat(dst.fd(), dst.name(), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(fi.Mode().Perm()|0o600))
	if err != nil {
		return untouched{&fs.PathError{Op: "open", Path: dst.path, Err: err}}
	}
	out := os.NewFile(uintptr(ofd), dst.path)
	defer func() {
		if err == nil {
			err = out.Chmod(fi.Mode().Perm())
		}
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = setTimes(dst, fi)
		}
	}()
	if _, err = io.Copy(out, progressReader{ctx: ctx, r: in, onBytes: onBytes}); err != nil {
		return err
	}
	return copyXattrs(ctx, in, out)
}

// copyXattrs gives out the extended attributes of in: Finder tags, a resource fork, and com.apple.quarantine, which
// makes Gatekeeper ask before a downloaded program runs. It goes before the mode, which may take the owner's write
// permission away. For a copy the attributes are a best effort, as in cp -p: one that cannot be read or that the
// destination does not take (a full attribute block, one removed in between) is left behind, and so is one above
// maxXattr, since a resource fork of gigabytes, which a tiny zip with an AppleDouble entry restores, would sit in
// memory whole. A move removes its source afterwards, so under wholeAttrs any of those fails the copy instead. Only an
// attribute the destination cannot hold at all (a filesystem without them, a security.* name only root sets) is left
// behind either way, as mv does. Esc stops between two attributes and cancels the copy.
func copyXattrs(ctx context.Context, in, out *os.File) error {
	whole := ctx.Value(wholeKey{}) != nil
	src, dst := int(in.Fd()), int(out.Fd())
	size, err := unix.Flistxattr(src, nil)
	if errors.Is(err, unix.ENOTSUP) || err == nil && size == 0 {
		return nil
	}
	list := make([]byte, max(size, 0))
	if err == nil {
		size, err = unix.Flistxattr(src, list)
	}
	if err != nil {
		return leftBehind(whole, &fs.PathError{Op: "listxattr", Path: in.Name(), Err: err})
	}
	for _, name := range strings.Split(strings.TrimRight(string(list[:size]), "\x00"), "\x00") {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := copyXattr(src, dst, name); err != nil {
			refused := errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES)
			if refused && !errors.Is(err, errSource) { // the destination cannot hold it
				continue
			}
			if err := leftBehind(whole, &fs.PathError{Op: "copy attribute " + name, Path: in.Name(), Err: err}); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyXattr copies one extended attribute; an error of setting it is the destination's, any other is the source's.
func copyXattr(src, dst int, name string) error {
	n, err := fgetxattr(src, name, nil)
	if err != nil {
		return fmt.Errorf("read: %w", errors.Join(err, errSource))
	}
	if n > maxXattr {
		return fmt.Errorf("larger than %d MiB", maxXattr>>20)
	}
	value := make([]byte, n)
	if n, err = fgetxattr(src, name, value); err != nil {
		return fmt.Errorf("read: %w", errors.Join(err, errSource))
	}
	return fsetxattr(dst, name, value[:n], 0)
}

// errSource marks an error of reading an attribute: EPERM there is not the destination refusing it.
var errSource = errors.New("reading the source")

// leftBehind is nil for a copy, which may leave an attribute behind, and err for a move, which may not.
func leftBehind(whole bool, err error) error {
	if whole {
		return err
	}
	return nil
}

// setTimes gives the item the copy made the modification time of its source, without following a link in its place.
func setTimes(dst item, fi fs.FileInfo) error {
	t := unix.NsecToTimespec(fi.ModTime().UnixNano())
	if err := unix.UtimesNanoAt(dst.fd(), dst.name(), []unix.Timespec{t, t}, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return &fs.PathError{Op: "chtimes", Path: dst.path, Err: err}
	}
	return nil
}

// maxXattr is the largest extended attribute a copy keeps.
const maxXattr = 16 << 20

// fgetxattr and fsetxattr are unix.Fgetxattr and unix.Fsetxattr; tests swap them for a huge attribute and for a
// destination that refuses one.
var fgetxattr, fsetxattr = unix.Fgetxattr, unix.Fsetxattr

// wholeKey marks a context whose copy must keep every extended attribute; see wholeAttrs.
type wholeKey struct{}

// wholeAttrs is ctx for a copy that must keep every extended attribute: one that removes its source afterwards.
func wholeAttrs(ctx context.Context) context.Context { return context.WithValue(ctx, wholeKey{}, true) }
