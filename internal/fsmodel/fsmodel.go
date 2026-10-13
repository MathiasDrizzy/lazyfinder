// Package fsmodel reads directories into immutable listings and formats them the way Finder does.
// It has no terminal dependency.
package fsmodel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Column identifies a sortable, displayable column.
type Column string

// The columns of the file list, in the default order.
const (
	ColName     Column = "name"
	ColModified Column = "modified"
	ColCreated  Column = "created"
	ColSize     Column = "size"
	ColKind     Column = "kind"
)

// AllColumns lists the columns in their default order.
var AllColumns = []Column{ColName, ColModified, ColCreated, ColSize, ColKind}

// Entry is one row of a listing. It is a plain value.
type Entry struct {
	Name     string
	Path     string
	IsDir    bool // true for directories and for symlinks that point to directories
	IsLink   bool
	Size     int64
	Modified time.Time
	Created  time.Time
	Kind     string
	Mode     os.FileMode
}

// Listing is the content of one directory. Methods return new values and never modify the receiver.
type Listing struct {
	Dir     string
	Entries []Entry
}

// ReadOptions controls what ReadDir returns.
type ReadOptions struct {
	ShowHidden bool
	Sort       Sort
}

// Sort describes an ordering. The zero value is not valid; use DefaultSort.
type Sort struct {
	By           Column
	Descending   bool
	FoldersFirst bool
}

// DefaultSort is name ascending with folders first, like Finder.
func DefaultSort() Sort { return Sort{By: ColName, FoldersFirst: true} }

// ReadDir lists dir. Entries that cannot be inspected (broken symlinks, races) are still listed.
// The error names the directory so it can be shown to the user as is.
func ReadDir(dir string, opts ReadOptions) (Listing, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return Listing{}, fmt.Errorf("cannot read folder %q: %w", filepath.Base(dir), underlying(err))
	}
	entries := make([]Entry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if !opts.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		entries = append(entries, inspect(dir, de))
	}
	l := Listing{Dir: dir, Entries: entries}
	s := opts.Sort
	if s.By == "" {
		s = DefaultSort()
	}
	return l.Sorted(s), nil
}

func underlying(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}

func inspect(dir string, de os.DirEntry) Entry {
	path := filepath.Join(dir, de.Name())
	e := Entry{Name: de.Name(), Path: path}
	li, err := os.Lstat(path)
	if err != nil {
		e.Kind = KindOf(e.Name, false, false)
		return e
	}
	e.IsLink = li.Mode()&os.ModeSymlink != 0
	info := li
	broken := false
	if e.IsLink {
		if ti, err := os.Stat(path); err == nil {
			info = ti
		} else {
			broken = true
		}
	}
	e.IsDir = info.IsDir()
	e.Mode = info.Mode()
	e.Modified = info.ModTime()
	e.Created = birthTime(info)
	if !e.IsDir && !broken {
		e.Size = info.Size()
	}
	e.Kind = KindOf(e.Name, e.IsDir, e.IsLink)
	return e
}

// Sorted returns a copy ordered by s.
func (l Listing) Sorted(s Sort) Listing {
	out := make([]Entry, len(l.Entries))
	copy(out, l.Entries)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if s.FoldersFirst && a.IsDir != b.IsDir {
			return a.IsDir
		}
		less, equal := compare(a, b, s.By)
		if equal {
			return naturalLess(a.Name, b.Name)
		}
		if s.Descending {
			return !less
		}
		return less
	})
	return Listing{Dir: l.Dir, Entries: out}
}

func compare(a, b Entry, by Column) (less, equal bool) {
	switch by {
	case ColModified:
		return a.Modified.Before(b.Modified), a.Modified.Equal(b.Modified)
	case ColCreated:
		return a.Created.Before(b.Created), a.Created.Equal(b.Created)
	case ColSize:
		return a.Size < b.Size, a.Size == b.Size
	case ColKind:
		ka, kb := strings.ToLower(a.Kind), strings.ToLower(b.Kind)
		return ka < kb, ka == kb
	default:
		return naturalLess(a.Name, b.Name), strings.EqualFold(a.Name, b.Name)
	}
}

// naturalLess compares case-insensitively and orders digit runs by value ("file2" < "file10").
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {
			si := i
			for i < len(ra) && unicode.IsDigit(ra[i]) {
				i++
			}
			sj := j
			for j < len(rb) && unicode.IsDigit(rb[j]) {
				j++
			}
			na, nb := strings.TrimLeft(string(ra[si:i]), "0"), strings.TrimLeft(string(rb[sj:j]), "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ra[i] != rb[j] {
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}

// HumanSize formats n like Finder (SI units: 1 KB = 1000 bytes).
func HumanSize(n int64) string {
	switch {
	case n == 0:
		return "Zero bytes"
	case n == 1:
		return "1 byte"
	case n < 1000:
		return fmt.Sprintf("%d bytes", n)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	v := float64(n)
	u := -1
	for v >= 1000 && u < len(units)-1 {
		v /= 1000
		u++
	}
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	return s + " " + units[u]
}

// FormatDate formats t relative to now: "Today at 09:05", "Yesterday at 23:59" or "2 Mar 2026".
// The zero time renders as "--".
func FormatDate(t, now time.Time) string {
	if t.IsZero() {
		return "--"
	}
	t, now = t.In(now.Location()), now.In(now.Location())
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	today := time.Date(ny, nm, nd, 0, 0, 0, 0, now.Location())
	switch {
	case ty == ny && tm == nm && td == nd:
		return "Today at " + t.Format("15:04")
	case ty == today.AddDate(0, 0, -1).Year() && tm == today.AddDate(0, 0, -1).Month() && td == today.AddDate(0, 0, -1).Day():
		return "Yesterday at " + t.Format("15:04")
	}
	return t.Format("2 Jan 2006")
}
